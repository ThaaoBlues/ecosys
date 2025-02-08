package setup

import (
	"archive/zip"
	"ecosys/globals"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/emersion/go-autostart"
)

var VERSION = "0.0.9-Beta"
var REPO_OWNER = "thaaoblues"
var REPO_NAME = "ecosys"
var REPO_URL = fmt.Sprintf("https://github.com/%s/%s", REPO_OWNER, REPO_NAME)

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

	zipFilePath := "webui.zip"

	// URL of the ZIP file to download
	zipURL := fmt.Sprintf("%s/raw/master/%s", REPO_URL, zipFilePath)

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
	if err := Unzip(zipFilePath, globals.EcosysWriteableDirectory); err != nil {
		fmt.Printf("Failed to unzip file: %v\n", err)
		return
	}

}

// Function to get the latest release tag from GitHub
func getLatestReleaseTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", REPO_OWNER, REPO_NAME)
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
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// Main function to check versions and take action if they do not match
func checkAndCompareVersion(versionFilePath string) error {
	latestTag, err := getLatestReleaseTag()
	if err != nil {
		return fmt.Errorf("error getting latest release tag: %v", err)
	}

	localVersion, err := readLocalVersionFile(versionFilePath)
	if err != nil {
		return fmt.Errorf("error reading local version file: %v", err)
	}

	if latestTag != localVersion {
		updateEcosys()
	} else {
		fmt.Println("Versions match. No action needed.")
	}

	return nil
}

func updateEcosys() {
	// download ecosys main exe and restart it
	// ecosys_linux_x64 for linux
	// ecosys_windows_x64 for windows
	// in the latest release of the ecosys github repo
	// after download, just start newer ecosys in another process and stop this program

	// Determine system-specific executable name
	var binaryName string
	switch runtime.GOOS {
	case "linux":
		binaryName = "ecosys_linux_x64"
	case "windows":
		binaryName = "ecosys_windows_x64.exe"
	default:
		log.Fatalf("Unsupported OS: %s", runtime.GOOS)
		return
	}

	// Define URL for the latest release binary file
	latestBinaryURL := fmt.Sprintf("%s/releases/latest/download/%s", REPO_URL, binaryName)
	binaryFilePath := filepath.Join(globals.EcosysWriteableDirectory, binaryName)

	// Download the latest binary
	fmt.Println("Downloading latest ecosys binary...")
	if err := DownloadFile(latestBinaryURL, binaryFilePath); err != nil {
		log.Fatalf("Failed to download latest ecosys binary: %v", err)
		return
	}

	// Make the binary executable (Linux/Mac only)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryFilePath, 0755); err != nil {
			log.Fatalf("Failed to set binary permissions: %v", err)
			return
		}
	}

	// Start the new binary in a separate process
	// and set up self-deletion based on OS
	fmt.Println("Starting the latest version of ecosys...")
	var cmd *exec.Cmd

	exPath, _ := os.Executable()
	if runtime.GOOS == "windows" {
		// Windows: Use a PowerShell command to delete the old binary after starting the new one
		cmd = exec.Command("cmd", "/C", "start", "/B", binaryFilePath, "&", "timeout", "/T", "2", "&", "del", "/Q", exPath)
	} else {
		// Linux/macOS: Use a shell command to delete the old binary after starting the new one
		cmd = exec.Command("sh", "-c", fmt.Sprintf("sleep 2 && rm -f %s &", exPath), "&", binaryFilePath)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start new ecosys process: %v", err)
		return
	}

	// Terminate the current program
	os.Exit(0)
}

func CheckUpdates() {
	versionFilePath := filepath.Join(globals.EcosysWriteableDirectory, "version")

	err := checkAndCompareVersion(versionFilePath)
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

	if !globals.ExistsInFilesystem(filepath.Join(globals.EcosysWriteableDirectory, "webui")) {
		DownloadWebuiFiles()

		// assume shortcuts are not created if web ui files are not
		switch runtime.GOOS {
		case "linux":
			CreateDesktopShortcutLinux()
		case "windows":
			CreateDesktopShortcutWindows()
		default:
			log.Fatalf("Unsupported OS: %s", runtime.GOOS)
			return
		}

	}

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	app := &autostart.App{
		Name:        "Ecosys",
		DisplayName: "Ecosys synchronization app",
		Exec:        []string{ex},
	}

	if !app.IsEnabled() {

		log.Println("Enabling app autostart...")

		if err := app.Enable(); err != nil {
			log.Fatal(err)
		}
	}

	// to disable autostart :
	/*
		if err := app.Disable(); err != nil {
			log.Fatal(err)
		}
	*/

}

func CleanupTempFiles() {
	// Get the current directory
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Read the directory contents
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	// Iterate through the files
	for _, file := range files {
		if !file.IsDir() {
			// Check if the file has the .btf or .nlock extension
			if strings.HasSuffix(file.Name(), ".btf") || strings.HasSuffix(file.Name(), ".nlock") {
				// Get the full path of the file
				filePath := filepath.Join(dir, file.Name())

				// Remove the file
				err := os.Remove(filePath)
				if err != nil {
					log.Printf("Failed to remove file: %s, error: %v\n", file.Name(), err)
				} else {
					fmt.Printf("Removed file: %s\n", file.Name())
				}
			}
		}
	}
}

func CreateDesktopShortcutLinux() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	shortcutConfig := `[Desktop Entry]
Name=Ecosys
Exec=` + ex + `
Terminal=false
Icon=` + filepath.Join(globals.EcosysWriteableDirectory, "webui", "icon.svg") + `
Type=Application
Comment=Ecosys synchronization and airdrop app`
	//file := os.OpenFile("ecosys.desktop", os.O_CREATE|os.O_RDWR)

	homeDir, err := os.UserHomeDir()

	if err != nil {
		panic(err)
	}

	f, err := os.Create(filepath.Join(homeDir, ".local/share/applications/ecosys.desktop"))
	if err != nil {
		panic(err)
	}

	_, err = f.Write([]byte(shortcutConfig))

	if err != nil {
		panic(err)
	}

}

func CreateDesktopShortcutWindows() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	// Create a shortcut on desktop
	CreateDesktopShortcut("Ecosys",
		ex,
		filepath.Join(globals.EcosysWriteableDirectory, "webui", "icon.ico"),
	)

}
