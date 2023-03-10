package audit

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"google-drive-audit/googledrive"
	"google-drive-audit/util"
	"io"
	"log"
	"os"
)

func Audit(ctx context.Context, domain, administratorEmail string, databaseFilename string, showProgress bool, serviceAccountFilePath string) error {
	var (
		files map[string]googledrive.File
		err   error
	)
	gd, err := googledrive.NewClient(ctx, domain, administratorEmail, serviceAccountFilePath)
	if err != nil {
		return fmt.Errorf("error creating service: %v", err)
	}
	// get files
	var status chan int
	if showProgress {
		status = make(chan int)
		go func() { // print status
			for s := range status {
				_, _ = fmt.Fprintf(os.Stderr, "\u001B[2K\rFiles processed: %d", s)
			}
			_, _ = fmt.Fprint(os.Stderr, "\n")
		}()
	}
	files, err = gd.GetAllFilesWithPermissions(ctx, status)
	if err != nil {
		return err
	}
	return saveToDisk(databaseFilename, files)
}

func WriteFileReport(_ context.Context, databaseFilename string, w io.Writer) error {
	f, err := loadFromDisk(databaseFilename)
	if err != nil {
		return err
	}
	files := *f

	// resolve full path
	type FileWithPath struct {
		googledrive.File
		FullPath string
	}
	allFiles := make(map[string]FileWithPath, 0)
	filemap := make(map[string]googledrive.File)
	for _, file := range files {
		filemap[file.ID] = file
	}
	for _, file := range files {
		fullPath := file.Name
		currentFile := file
		for {
			if currentFile.Parent == "" {
				break
			} else if parentFile, ok := filemap[currentFile.Parent]; ok { // navigate to parent
				fullPath = parentFile.Name + "/" + fullPath
				currentFile = parentFile
			} else { // edge case: parent is defined but not found
				fullPath = "$" + currentFile.Parent + "/" + fullPath
				break
			}
		}
		allFiles[file.ID] = FileWithPath{File: file, FullPath: fullPath}
	}

	// print
	csvWriter := csv.NewWriter(w)
	csvWriter.UseCRLF = true
	defer csvWriter.Flush()
	err = csvWriter.Write([]string{"File name", "Role", "Permission Type", "Person", "Permission ID", "Inherited Permission", "Type", "URL"})
	if err != nil {
		return err
	}

	for _, file := range allFiles {
		for email, perm := range file.Permissions {
			err = csvWriter.Write([]string{file.FullPath, perm.Role, perm.Type, email, perm.ID, boolToString(perm.Inherited), file.MimeType, file.OpenURL})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func UnshareFiles(ctx context.Context, domain, administratorEmail string, serviceAccountFilePath string, databaseFilename string, emailToDelete string, dryRun bool) error {
	f, err := loadFromDisk(databaseFilename)
	if err != nil {
		return err
	}
	files := *f

	gd, err := googledrive.NewClient(ctx, domain, administratorEmail, serviceAccountFilePath)
	if err != nil {
		return fmt.Errorf("error creating service: %v", err)
	}

	// collect owner, fileID and permissionIDs
	type PermissionToDelete struct {
		OwnerEmail   string
		FileID       string
		PermissionID string
	}
	for _, file := range files {
		var ownerEmail string
		var permissionIDToDelete string
		for email, perm := range file.Permissions {
			switch perm.Role {
			case "owner", "organizer":
				ownerEmail = email
			}
			if !perm.Inherited && email == emailToDelete {
				permissionIDToDelete = perm.ID
			}
		}
		if ownerEmail != "" && permissionIDToDelete != "" { // delete
			log.Printf("Delete permission: (id=%s permission-id=%s), owner=%s\n", file.ID, permissionIDToDelete, ownerEmail)
			if dryRun {
				log.Println("(dry run, not doing)")
			} else {
				err = gd.RemovePermissions(ctx, ownerEmail, file.ID, permissionIDToDelete)
				if err != nil {
					return fmt.Errorf("error deleting permission: %v", err)
				}
			}
		}
	}
	return nil
}

func boolToString(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

func saveToDisk(filename string, files googledrive.FilesByID) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer util.PrintIfError(f.Close)

	err = json.NewEncoder(f).Encode(files)
	return err
}

func loadFromDisk(filename string) (*googledrive.FilesByID, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer util.PrintIfError(f.Close)

	var filesAndPermissions googledrive.FilesByID
	err = json.NewDecoder(f).Decode(&filesAndPermissions)
	return &filesAndPermissions, err
}
