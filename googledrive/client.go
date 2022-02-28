package googledrive

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"golang.org/x/oauth2"

	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Client struct {
	userEmails             []string
	serviceAccountFilePath string
}

type File struct {
	ID          string
	Name        string
	Permissions map[string]string // email -> role
	Parent      string
	MimeType    string
	OpenURL     string
}

func NewClient(ctx context.Context, domain string, administratorEmail string, serviceAccountFilePath string) (*Client, error) {
	ts, err := tokenSourceForSubject(ctx, serviceAccountFilePath, &administratorEmail, admin.AdminDirectoryUserReadonlyScope)
	if err != nil {
		return nil, err
	}
	srv, err := admin.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("NewService: %v", err)
	}
	var nextPageToken string
	users := make([]string, 0)
	for {
		call := srv.Users.List().
			Domain(domain).
			Fields("nextPageToken, users(primaryEmail)")
		if nextPageToken != "" {
			call.PageToken(nextPageToken)
		}
		usersResponse, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve users: %v", err)
		}
		for _, user := range usersResponse.Users {
			users = append(users, user.PrimaryEmail)
		}
		if usersResponse.NextPageToken == "" {
			break
		}
		nextPageToken = usersResponse.NextPageToken
	}
	return &Client{users, serviceAccountFilePath}, nil
}

func (d *Client) getDriveClientForUser(ctx context.Context, userEmail string) (*drive.Service, error) {
	ts, err := tokenSourceForSubject(ctx, d.serviceAccountFilePath, &userEmail, drive.DriveReadonlyScope)
	if err != nil {
		return nil, err
	}

	srv, err := drive.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("NewService: %v", err)
	}
	return srv, nil
}

func tokenSourceForSubject(ctx context.Context, serviceAccountFilePath string, subject *string, scopes ...string) (oauth2.TokenSource, error) {
	jsonCredentials, err := ioutil.ReadFile(serviceAccountFilePath)
	if err != nil {
		return nil, err
	}

	config, err := google.JWTConfigFromJSON(jsonCredentials, scopes...)
	if err != nil {
		return nil, fmt.Errorf("JWTConfigFromJSON: %v", err)
	}
	if subject != nil {
		config.Subject = *subject
	}
	ts := config.TokenSource(ctx)
	return ts, nil
}

func (d *Client) GetAllFilesWithPermissions(ctx context.Context, status chan<- int) (map[string]File, error) {
	defer close(status)
	allFiles := make(map[string]File)
	// all users
	for _, userEmail := range d.userEmails {
		srv, err := d.getDriveClientForUser(ctx, userEmail)
		if err != nil {
			return nil, err
		}
		// all drives
		drives, err := getDrives(srv)
		if err != nil {
			return nil, err
		}
		for _, d := range drives {
			allFiles[d.Id] = File{
				ID:   d.Id,
				Name: d.Name,
			}
		}

		// my drive
		myDriveFile, err := srv.Files.Get("root").Fields("id").Context(ctx).Do() // id="root" is undocumented
		if err != nil {
			return nil, err
		}
		allFiles[myDriveFile.Id] = File{
			ID:   myDriveFile.Id,
			Name: userEmail + " My Drive",
		}

		// all files for each user
		var nextPageToken string
		for {
			call := srv.Files.List().
				Corpora("allDrives").
				SupportsAllDrives(true).
				IncludeItemsFromAllDrives(true).
				Q("trashed=false").
				PageSize(1000).
				Fields("nextPageToken, incompleteSearch, files(id, driveId, mimeType, name, owners(emailAddress), parents, permissions(emailAddress, role), shared, sharingUser(emailAddress), webViewLink)")
			if nextPageToken != "" {
				call.PageToken(nextPageToken)
			}
			r, err := call.Do()
			if err != nil {
				return nil, fmt.Errorf("unable to retrieve files: %v", err)
			}
			if r.IncompleteSearch {
				return nil, fmt.Errorf("unable to retrieve all files, incomplete search")
			}
			for _, file := range r.Files {
				if _, contains := allFiles[file.Id]; contains {
					continue // skip known files
				}
				// We got 3 places where to get permissions from:
				permissions := make(map[string]string)
				// 1. file.Permissions
				for _, p := range file.Permissions {
					permissions[p.EmailAddress] = p.Role
				}
				// 2. permissions API
				if len(permissions) == 0 && !file.Shared {
					// get permissions with separate calls
					ps, err := getPermissionsForFile(srv, file)
					if err != nil {
						return nil, err
					}
					for _, p := range ps {
						permissions[p.EmailAddress] = p.Role
					}
				}
				// 3. owners
				for _, owner := range file.Owners {
					permissions[owner.EmailAddress] = "owner"
				}
				if len(permissions) == 0 {
					return nil, fmt.Errorf("did not get any file permissions: %s - %s", file.Id, file.Name)
				}
				// a file can have multiple parents (deprecated). We'll only choose one
				var parent string
				if len(file.Parents) > 0 {
					parent = file.Parents[0]
				} else {
					parent = file.DriveId // if file has no parent but belongs to a drive, that's the parent. DriveId is only populated for team drives
				}
				allFiles[file.Id] = File{
					ID:          file.Id,
					Name:        file.Name,
					Permissions: permissions,
					Parent:      parent,
					MimeType:    file.MimeType,
					OpenURL:     file.WebViewLink,
				}
				if status != nil {
					status <- len(allFiles) // update status
				}
			}
			if r.NextPageToken == "" { // done
				break
			}
			nextPageToken = r.NextPageToken
		}
	}
	// extract values
	return allFiles, nil
}

// getPermissionsForFile lists the permissions of a file. Ignores insufficientFilePermissions error.
func getPermissionsForFile(srv *drive.Service, file *drive.File) ([]*drive.Permission, error) {
	var result []*drive.Permission
	var nextPageToken string
	for {
		call := srv.Permissions.List(file.Id).
			SupportsAllDrives(true).
			Fields("nextPageToken, permissions(emailAddress, role, permissionDetails(role))")
		if nextPageToken != "" {
			call.PageToken(nextPageToken)
		}
		r, err := call.Do()
		if err != nil {
			if strings.HasSuffix(err.Error(), "insufficientFilePermissions") {
				return result, nil
			}
			return nil, fmt.Errorf("unable to retrieve permissions for file %s %s: %v", file.Id, file.Name, err)
		}
		for _, i := range r.Permissions {
			result = append(result, i)
		}
		if r.NextPageToken == "" { // done
			break
		}
		nextPageToken = r.NextPageToken
	}
	return result, nil
}

func getDrives(srv *drive.Service) ([]*drive.Drive, error) {
	var result []*drive.Drive
	var nextPageToken string
	for {
		call := srv.Drives.List().
			Fields("nextPageToken, drives(id, name)")
		if nextPageToken != "" {
			call.PageToken(nextPageToken)
		}
		d, err := call.Do()
		if err != nil {
			return nil, err
		}
		for _, i := range d.Drives {
			result = append(result, i)
		}
		if d.NextPageToken == "" { // done
			break
		}
		nextPageToken = d.NextPageToken
	}
	return result, nil
}
