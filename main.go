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
	// keep an up to date list of linked devices that are on our network
	go zc.UpdateDevicesConnectionStateLoop()
	// loop accepting and treating requests from other devices
	go networking.NetWorkLoop()

	tui.DisplayMenu()

	/*go func() {
		user_data := backendapi.AskInput("test", "Donnez une entrée test")
		fmt.Println("user_data : ", user_data)
	}()

	time.Sleep(1 * time.Second)

	// get context outside of the callback function (will crash if no inputs are asked)
	fmt.Println("contexte outside of callback: ", backendapi.ReadInputContext("test"))

	callbacks := make(map[string]func(string))

	callbacks["test"] = func(context string) {
		fmt.Println("context in callback : ", context)
		backendapi.GiveInput("test", "OUI OUI HEHEHEHEHEH")

	}

	backendapi.WaitEventLoop(callbacks)*/

}
