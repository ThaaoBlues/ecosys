package magasin

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"qsync/bdd"
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

func InstallApp(data io.ReadCloser) error {
	var json_data bdd.ToutEnUnConfig
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
