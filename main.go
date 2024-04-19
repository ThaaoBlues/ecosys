package main

import (
	"log"
	"os"
	"qsync/globals"
	"qsync/networking"
	"qsync/tui"
)

func main() {

	var zc networking.ZeroConfService
	log_file, _ := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	defer log_file.Close()

	log.SetOutput(log_file)

	// register this device
	go zc.Register()
	// keep an up to date list of linked devices that are on our network
	go zc.UpdateDevicesConnectionStateLoop()
	// loop accepting and treating requests from other devices
	go networking.NetWorkLoop()

	// as in this main function we are always on desktop
	// assume the directory where qsync the executable is
	// has read/write access
	globals.SetQsyncWriteableDirectory("")

	tui.DisplayMenu()

}
