package setup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"qsync/globals"
	"strings"
)

var VERSION = "0.0.1-Pre-Alpha"

func MakeDirectories() {
	err := os.Mkdir("largages_aeriens", 0755)
	if err != nil {
		log.Fatal("Error while creating directories in setup", err)
	}

}

// DownloadFile downloads a file from the specified URL and saves it to the specified path.
func DownloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Unzip extracts a ZIP archive to a target directory.
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// RemoveFolder deletes a folder and all its contents.
func RemoveFolder(path string) error {
	return os.RemoveAll(path)
}

func DownloadWebuiFiles() {

	// downloading all the web gui files

	// URL of the ZIP file to download
	zipURL := "http://github.com/thaaoblues/qsync/webui.zip"

	// Local path to save the downloaded ZIP file
	zipFilePath := "webui.zip"
	// Folder to extract the ZIP contents
	folderName := strings.TrimSuffix(zipFilePath, filepath.Ext(zipFilePath))

	// Download the ZIP file
	if err := DownloadFile(zipURL, zipFilePath); err != nil {
		fmt.Printf("Failed to download file: %v\n", err)
		return
	}

	// Remove existing folder if it exists
	if _, err := os.Stat(folderName); err == nil {
		fmt.Println("Removing existing folder...")
		if err := RemoveFolder(folderName); err != nil {
			fmt.Printf("Failed to remove folder: %v\n", err)
			return
		}
	}

	// Unzip the downloaded file
	if err := Unzip(zipFilePath, folderName); err != nil {
		fmt.Printf("Failed to unzip file: %v\n", err)
		return
	}

}

// Function to get the latest release tag from GitHub
func getLatestReleaseTag(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API request failed with status: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

// Function to read the version from a local file
func readLocalVersionFile(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// Main function to check versions and take action if they do not match
func checkAndCompareVersion(owner, repo, versionFilePath string) error {
	latestTag, err := getLatestReleaseTag(owner, repo)
	if err != nil {
		return fmt.Errorf("error getting latest release tag: %v", err)
	}

	localVersion, err := readLocalVersionFile(versionFilePath)
	if err != nil {
		return fmt.Errorf("error reading local version file: %v", err)
	}

	if latestTag != localVersion {
		updateQsync()
	} else {
		fmt.Println("Versions match. No action needed.")
	}

	return nil
}

func updateQsync() {
	// download qsync main exe and restart it

}

func CheckUpdates() {
	owner := "thaaoblues"
	repo := "qsync"
	versionFilePath := filepath.Join(globals.QSyncWriteableDirectory, "version")

	err := checkAndCompareVersion(owner, repo, versionFilePath)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Setup() {

	f, err := os.Create("version")

	if err != nil {
		log.Fatal("Error while creating version file", err)
	}

	f.WriteString(VERSION)
	f.Close()

	DownloadWebuiFiles()

}
