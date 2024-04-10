package magasin

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"qsync/bdd"
	"strings"
)

func RunInstaller(path string) {

}

func DownloadFromUrl(url string, installer_path string) error {
	out, err := os.Create(installer_path)
	if err != nil {
		return err
	}

	defer out.Close()

	resp, err := http.Get(url)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)

	if err != nil {
		return err
	}

	return nil
}

// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func InstallApp(data io.ReadCloser) error {
	var json_data bdd.ToutEnUnConfig
	err := json.NewDecoder(data).Decode(&json_data)

	// by default, the app will be installed to <qsync_installation_root>/apps/<app_name>

	if err != nil {
		return err
	}

	self_path, err := os.Executable()
	if err != nil {
		log.Fatal("An error occured while determining the path to qsync executable in InstallApp()", err)
	}

	apps_path := filepath.Join(self_path, "apps")
	ex, err := exists(apps_path)

	if err != nil {
		log.Fatal("An error occured while checking if a path exists in InstallApp()", err)
	}

	if !ex {
		os.Mkdir(apps_path, fs.ModeDir)
	}

	sanitized_app_name := strings.ReplaceAll(json_data.AppName, " ", "_")
	new_app_root_path := filepath.Join(self_path, sanitized_app_name)
	json_data.AppLauncherPath = filepath.Join(new_app_root_path, json_data.AppLauncherPath)

	ex, err = exists(new_app_root_path)

	if !ex {
		os.Mkdir(new_app_root_path, fs.ModeDir)
	}

	if err != nil {
		log.Fatal("An error occured while checking if a path exists in InstallApp()", err)
	}

	// pre-determined installer name so there are no problem ( on linux .exe does not do anything but required on windows)
	json_data.AppInstallerPath = filepath.Join(new_app_root_path, sanitized_app_name+".exe")

	err = DownloadFromUrl(json_data.AppDownloadUrl, json_data.AppInstallerPath)

	if err != nil {
		return err
	}

	if json_data.NeedsInstaller {
		RunInstaller(json_data.AppInstallerPath)
	}

	var acces bdd.AccesBdd

	acces.InitConnection()
	defer acces.CloseConnection()

	acces.CreateSync(json_data.AppSyncDataFolderPath)

	acces.AddToutEnUn(json_data)

	return nil

}

func UninstallToutenun(config bdd.MinGenConfig) error {

	cmd := exec.Command(config.UninstallerPath)

	err := cmd.Run()
	return err
}
