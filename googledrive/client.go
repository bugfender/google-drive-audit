package googledrive

import (
	"context"
	"fmt"
	"os"

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
	return d.getDriveClientForUserWithScope(ctx, userEmail, drive.DriveReadonlyScope)
}

func (d *Client) getDriveClientForUserWithWritePermissions(ctx context.Context, userEmail string) (*drive.Service, error) {
	return d.getDriveClientForUserWithScope(ctx, userEmail, drive.DriveScope)
}

func (d *Client) getDriveClientForUserWithScope(ctx context.Context, userEmail string, scope ...string) (*drive.Service, error) {
	ts, err := tokenSourceForSubject(ctx, d.serviceAccountFilePath, &userEmail, scope...)
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
	jsonCredentials, err := os.ReadFile(serviceAccountFilePath)
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
