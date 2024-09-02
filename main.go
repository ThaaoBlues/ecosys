/*
 * @file            main.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2023-09-11 14:08:11
 * @lastModified    2024-09-01 22:13:24
 * Copyright ©Théo Mougnibas All rights reserved
 */

package main

import (
	"ecosys/bdd"
	"ecosys/filesystem"
	"ecosys/globals"
	"ecosys/networking"
	"ecosys/setup"
	"ecosys/webui"
	"log"
	"os"
	"path/filepath"
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

	// make sure we are working at the root of the ecosys executable
	os.Chdir(exPath)

	setup.CleanupTempFiles()
	if networking.IsNetworkAvailable() {
		setup.Setup()
		setup.CheckUpdates()
	}

	var zc networking.ZeroConfService
	log_file, _ := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	defer log_file.Close()

	log.SetOutput(log_file)

	// register this device
	go zc.Register()
	// keep an up to date list ofmtf linked devices that are on our network
	go zc.UpdateDevicesConnectionStateLoop()
	// loop accepting and treating requests from other devices
	go networking.NetWorkLoop()

	//tui.DisplayMenu()

	go webui.StartWebUI()
	globals.OpenUrlInWebBrowser("http://127.0.0.1:8275")

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

	// just wait the end of times

	select {}
}
