package magasin

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"qsync/bdd"
)

type ToutEnUnConfig struct {
	AppName               string // well... the app's name ?
	AppDownloadUrl        string // the url where to download the app
	NeedsInstaller        bool   // if we need to run the binary installer
	AppLauncherPath       string // the path to the main executable of your app
	AppInstallerPath      string // the installer path
	AppUninstallerPath    string // the uninstaller path
	AppSyncDataFolderPath string // the folder where the data to synchronize is stored
}

func RunInstaller(path string) {

}

func DownloadFromUrl(url string, installer_path string) error {
	out, err := os.Create(installer_path)
	defer out.Close()

	if err != nil {
		return err
	}

	resp, err := http.Get(url)
	defer resp.Body.Close()

	if err != nil {
		return err
	}

	_, err = io.Copy(out, resp.Body)

	if err != nil {
		return err
	}

	return nil
}

func InstallApp(data io.ReadCloser) error {
	var json_data ToutEnUnConfig
	err := json.NewDecoder(data).Decode(&json_data)

	// by default, the app will be installed to <qsync_installation_root>/apps

	if err != nil {
		return err
	}

	err = DownloadFromUrl(json_data.AppDownloadUrl, json_data.AppInstallerPath)

	if err != nil {
		return err
	}

	if json_data.NeedsInstaller {
		RunInstaller(json_data.AppInstallerPath)
	}

	var acces bdd.AccesBdd

	acces.InitConnection()

	acces.AddToutEnUn(json_data)

	acces.CreateSync(json_data.AppSyncDataFolderPath)

	return nil

}
