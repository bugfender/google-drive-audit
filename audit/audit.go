package audit

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"google-drive-audit/googledrive"
	"google-drive-audit/util"
	"io"
	"os"
)

func ExportFileListToCSV(ctx context.Context, w io.Writer, domain, administratorEmail string, fromCache, showProgress bool, serviceAccountFilePath string) error {
	var (
		files map[string]googledrive.File
		err   error
	)
	if !fromCache {
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
		err = saveToDisk(files)
		if err != nil {
			return err
		}
	} else {
		files, err = loadFromDisk()
		if err != nil {
			return err
		}
	}

	// resolve full path
	type FileWithPath struct {
		googledrive.File
		FullPath string
	}
	allFiles := make([]FileWithPath, 0)
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
		allFiles = append(allFiles, FileWithPath{File: file, FullPath: fullPath})
	}

	// print
	csvWriter := csv.NewWriter(w)
	csvWriter.UseCRLF = true
	defer csvWriter.Flush()
	err = csvWriter.Write([]string{"File name", "Person", "Role", "Type", "URL"})
	if err != nil {
		return err
	}

	for _, file := range allFiles {
		for email, role := range file.Permissions {
			err = csvWriter.Write([]string{file.FullPath, email, role, file.MimeType, file.OpenURL})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func saveToDisk(files map[string]googledrive.File) error {
	f, err := os.Create("files.json")
	if err != nil {
		return err
	}
	defer util.PrintIfError(f.Close)

	err = json.NewEncoder(f).Encode(files)
	return err
}

func loadFromDisk() (map[string]googledrive.File, error) {
	f, err := os.Open("files.json")
	if err != nil {
		return nil, err
	}
	defer util.PrintIfError(f.Close)

	var files map[string]googledrive.File
	err = json.NewDecoder(f).Decode(&files)
	return files, err
}
