package main

import (
	"qsync/networking"
	"qsync/tui"
)

//import "qsync/bdd"

func main() {

	/*var acces bdd.AccesBdd

	acces.InitConnection()*/

	//log.Println(acces.GetMyDeviceId())

	//acces.GetSecureId("/home/h3x0/dev/projects/qsync/test_files")

	//log.Print(acces.WasFile("/home/h3x0/dev/projects/qsync/test_files/folder"))

	//acces.CreateFile("/home/h3x0/dev/projects/qsync/test_files/prout_new")

	//path := "/home/h3x0/dev/projects/qsync/test_files"
	//delta := delta_binaire.BuilDelta(path, acces.GetFileSizeFromBdd(path), acces.GetFileContent(path))
	//log.Println(delta)
	//acces.UpdateFile(path, delta)
	//log.Print(acces.GetFileDelta(1, path))
	//fsmon.StartWatcher(path)

	var zc networking.ZeroConfService

	// register this device
	go zc.Register()
	go networking.NetWorkLoop()

	tui.DisplayMenu()

}
