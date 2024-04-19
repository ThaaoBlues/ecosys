package magasin

import (
	"encoding/json"
	"io"
	"os"
	"qsync/bdd"
	"qsync/globals"
	"strings"
)

// Function to format the path based on certain placeholders
func formatPath(path string) string {
	// Replace %username% with the actual username (assuming current user)
	username, _ := os.UserHomeDir()
	path = strings.ReplaceAll(path, "%username%", username)

	// Replace %any% with the first child folder in the parent directory
	parentDir := getParentDirectory(path)
	childDir := getFirstChildDirectory(parentDir)
	path = strings.ReplaceAll(path, "%any%", childDir)
	path = strings.ReplaceAll(path, "%version%", childDir)

	return path
}

// Function to get the parent directory of a given path
func getParentDirectory(path string) string {
	index := strings.LastIndex(path, "/")
	if index == -1 {
		return ""
	}
	return path[:index]
}

// Function to get the first child directory of a given parent directory
func getFirstChildDirectory(parentDir string) string {
	files, err := os.ReadDir(parentDir)
	if err != nil {
		return ""
	}
	for _, file := range files {
		if file.IsDir() {
			return file.Name()
		}
	}
	return ""
}

func InstallGrapin(data io.ReadCloser) error {
	var config globals.GrapinConfig
	err := json.NewDecoder(data).Decode(&config)

	if err != nil {
		return err
	}

	// Modify the application path based on the NeedsFormat flag
	if config.NeedsFormat {
		config.AppSyncDataFolderPath = formatPath(config.AppSyncDataFolderPath)
	}

	var acces bdd.AccesBdd

	acces.InitConnection()
	defer acces.CloseConnection()

	acces.CreateSync(config.AppSyncDataFolderPath)
	acces.AddGrapin(config)

	return nil

}
