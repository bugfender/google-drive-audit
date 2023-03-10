package googledrive

import (
	"context"
)

func (d *Client) RemovePermissions(ctx context.Context, ownerEmail string, fileID string, permissionID string) error {
	srv, err := d.getDriveClientForUserWithWritePermissions(ctx, ownerEmail)
	if err != nil {
		return err
	}

	return srv.Permissions.Delete(fileID, permissionID).SupportsAllDrives(true).Context(ctx).Do()
}
