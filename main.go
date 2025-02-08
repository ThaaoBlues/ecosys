/*
 * @file            main.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2023-09-11 14:08:11
 * @lastModified    2024-10-31 22:28:32
 * Copyright ©Théo Mougnibas All rights reserved
 */

package main

import (
	"ecosys/bdd"
	"ecosys/filesystem"
	"ecosys/globals"
	"ecosys/networking"
	"ecosys/setup"
	"net/http"

	//"ecosys/tui"
	"ecosys/webui"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jeandeaual/go-locale"
	webview "github.com/webview/webview_go"
	//"github.com/rivo/tview"
)

func main() {
	// as in this main function we are always on desktop
	// assume the directory where ecosys the executable is
	// has read/write access
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	globals.SetecosysWriteableDirectory(exPath)

	log_file, _ := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	defer log_file.Close()

	log.SetOutput(log_file)

	// make sure we are working at the root of the ecosys executable
	os.Chdir(exPath)

	setup.CleanupTempFiles()
	if networking.IsNetworkAvailable() {
		setup.Setup()
		setup.CheckUpdates()

		// wait for potential internet connection
		go func() {
			for !networking.IsNetworkAvailable() {
				// 5 second timer before each try
				time.Sleep(5 * time.Second)
			}
			var zc networking.ZeroConfService
			// register this device
			go zc.Register()
			// keep an up to date list ofmtf linked devices that are on our network
			go zc.UpdateDevicesConnectionStateLoop()
			// loop accepting and treating requests from other devices
			go networking.NetWorkLoop()

		}()

	}

	// first, check if ecosys is already running or not, if yes, just open a webview
	// the api has an endpoint to check internet connection, so if we get a response
	// it means that the internal web server of ecosys is already running
	_, err = http.Get("http://127.0.0.1:8275/check-internet")

	// probably timeout error, means we have to start all ecosys internals
	if err != nil {
		log.Println("Network check failed, nothing unusual as it is to check if ecosys server was already running")
		// web ui still used as an api even if we use tui
		go webui.StartWebUI()

		//start ecosys
		var acces bdd.AccesBdd
		acces.InitConnection()
		defer acces.CloseConnection()

		acces.ClearAllFileSystemLockInDb()

		tasks := acces.ListSyncAllTasks()
		for i := 0; i < tasks.Size(); i++ {
			acces.SecureId = tasks.Get(i).SecureId
			go filesystem.StartWatcher(tasks.Get(i).Path)
		}

		lang, err := locale.GetLanguage()
		if err != nil {
			lang = "en"
		}
		globals.SetCurrentLangIfAvailable(lang)
	} else {
		log.Println("Ecosys was already running, we will only open a new webview instance.")
	}

	w := webview.New(true)
	defer w.Destroy()
	w.SetTitle("Ecosys")
	w.Navigate("http://127.0.0.1:8275")
	w.SetSize(1920, 1080, webview.HintNone)

	/*app := tview.NewApplication()

	ui := tui.CreateUI(app)

	if err := app.SetRoot(ui, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}*/

	w.Run()

}
