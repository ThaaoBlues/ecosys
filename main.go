package main

import (
	"log"
	"os"
	"path/filepath"
	"qsync/globals"
	"qsync/networking"
	"qsync/setup"

	"qsync/webui"
)

func main() {
	// as in this main function we are always on desktop
	// assume the directory where qsync the executable is
	// has read/write access
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	globals.SetQsyncWriteableDirectory(exPath)

	setup.CleanupTempFiles()
	setup.Setup()
	setup.CheckUpdates()

	var zc networking.ZeroConfService
	log_file, _ := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	defer log_file.Close()

	log.SetOutput(log_file)

	if networking.IsNetworkAvailable() {
		// register this device
		go zc.Register()
		// keep an up to date list ofmtf linked devices that are on our network
		go zc.UpdateDevicesConnectionStateLoop()
		// loop accepting and treating requests from other devices
		go networking.NetWorkLoop()

	}

	//tui.DisplayMenu()

	webui.StartWebUI()
	globals.OpenUrlInWebBrowser("http://127.0.0.1:8275")

}
