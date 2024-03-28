package main

import (
	"qsync/networking"
	"qsync/tui"
)

func main() {

	var zc networking.ZeroConfService

	// register this device
	go zc.Register()
	// keep an up to date list of linked devices that are on our network
	go zc.UpdateDevicesConnectionStateLoop()
	// loop accepting and treating requests from other devices
	go networking.NetWorkLoop()

	tui.DisplayMenu()

}
