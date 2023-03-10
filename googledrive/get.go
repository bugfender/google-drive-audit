package googledrive

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/drive/v3"
)

type FilesByID map[string]File

type File struct {
	ID          string
	Name        string
	Permissions map[string]FilePermission // key:email
	SharerEmail string
	Parent      string
	MimeType    string
	OpenURL     string
}

type FilePermission struct {
	ID        string
	Type      string
	Role      string
	Inherited bool
}

func (d *Client) GetAllFilesWithPermissions(ctx context.Context, status chan<- int) (FilesByID, error) {
	allFiles := make(map[string]File) //key:id
	for _, userEmail := range d.userEmails {
		srv, err := d.getDriveClientForUser(ctx, userEmail)
		if err != nil {
			return nil, err
		}

		// all drives the user can see
		drives, err := getDrives(srv)
		if err != nil {
			return nil, err
		}
		for _, d := range drives {
			// add the drive itself
			allFiles[d.Id] = File{
				ID:   d.Id,
				Name: d.Name,
			}
			// all files within the drive
			err = addFilesForQuery(ctx, allFiles, srv, srv.Files.List().
				Corpora("drive").
				DriveId(d.Id).
				Q("trashed=false"), status)
			if err != nil {
				return nil, err
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

		// all files within my drive
		err = addFilesForQuery(ctx, allFiles, srv, srv.Files.List().
			Corpora("user").
			Q("trashed=false and 'me' in owners"), status)
		if err != nil {
			return nil, err
		}
	}
	return allFiles, nil
}

func addFilesForQuery(ctx context.Context, allFiles FilesByID, srv *drive.Service, call *drive.FilesListCall, status chan<- int) error {
	call = call.
		Context(ctx).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		PageSize(1000).
		Fields("nextPageToken, incompleteSearch, files(id, driveId, mimeType, name, owners(emailAddress), parents, permissions(id, emailAddress, type, role), shared, sharingUser(emailAddress), webViewLink)")
	var nextPageToken string
	for {
		if nextPageToken != "" {
			call.PageToken(nextPageToken)
		}
		r, err := call.Do()
		if err != nil {
			return fmt.Errorf("unable to retrieve files: %v", err)
		}
		if r.IncompleteSearch {
			return fmt.Errorf("unable to retrieve all files, incomplete search")
		}
		for _, file := range r.Files {
			if _, contains := allFiles[file.Id]; contains {
				continue // skip known files
			}
			// We got 3 places where to get permissions from:
			permissions := make(map[string]FilePermission)
			// 1. file.Permissions (for files in drive)
			for _, p := range file.Permissions {
				permissions[p.EmailAddress] = FilePermission{p.Id, p.Type, p.Role, false}
			}
			// 2. permissions API (for files in Team Drive)
			if len(permissions) == 0 && !file.Shared {
				// get permissions with separate calls
				ps, err := getPermissionsForFile(ctx, srv, file)
				if err != nil {
					return err
				}
				for email, permission := range ps {
					permissions[email] = permission
				}
			}
			// 3. owners
			for _, owner := range file.Owners {
				permissions[owner.EmailAddress] = FilePermission{"", "user", "owner", false}
			}
			// sharer
			var sharer string
			if file.SharingUser != nil {
				sharer = file.SharingUser.EmailAddress
			}
			if len(permissions) == 0 {
				return fmt.Errorf("did not get any file permissions: %s - %s", file.Id, file.Name)
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
				SharerEmail: sharer,
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
	return nil
}

// getPermissionsForFile lists the permissions of a file. Ignores insufficientFilePermissions error.
func getPermissionsForFile(ctx context.Context, srv *drive.Service, file *drive.File) (map[string]FilePermission, error) {
	result := make(map[string]FilePermission)
	var nextPageToken string
	for {
		call := srv.Permissions.List(file.Id).
			Context(ctx).
			SupportsAllDrives(true).
			Fields("nextPageToken, permissions(id, emailAddress, permissionDetails(role, permissionType, inherited))")
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
		for _, p := range r.Permissions {
			for _, pd := range p.PermissionDetails {
				result[p.EmailAddress] = FilePermission{p.Id, pd.PermissionType, pd.Role, pd.Inherited}
			}
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
