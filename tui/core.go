package tui

import (
	"fmt"
	"log"
	"qsync/backend_api"
	"qsync/bdd"
	"qsync/filesystem"
	"qsync/globals"
	"qsync/magasin"
	"qsync/networking"
	"strconv"
	"time"

	"github.com/sqweek/dialog"
)

var LOGO string = `
________/\\\___________/\\\\\\\\\\\_____________________________________________        
_____/\\\\/\\\\______/\\\/////////\\\___________________________________________       
 ___/\\\//\////\\\___\//\\\______\///_____/\\\__/\\\_____________________________      
  __/\\\______\//\\\___\////\\\___________\//\\\/\\\___/\\/\\\\\\_______/\\\\\\\\_     
   _\//\\\______/\\\_______\////\\\_________\//\\\\\___\/\\\////\\\____/\\\//////__    
    __\///\\\\/\\\\/___________\////\\\_______\//\\\____\/\\\__\//\\\__/\\\_________   
     ____\////\\\//______/\\\______\//\\\___/\\_/\\\_____\/\\\___\/\\\_\//\\\________  
      _______\///\\\\\\__\///\\\\\\\\\\\/___\//\\\\/______\/\\\___\/\\\__\///\\\\\\\\_ 
       _________\//////_____\///////////______\////________\///____\///_____\////////__`

var MENU string = `

[0] - Start QSync
[1] - Create a sync task
[2] - Link another machine to a sync task on yours
[3] - List current sync task and their id
[4] - List devices using qsync on your network
[5] - Open QSync Magasin
[6] - Send something to another device : "Largage Aérien"
[7] - Send a whole folder to another device : "Multi Largage Aérien"
[8] - Allow/Disallow people to send you Largage Aerien
`

var PROMPT string = "\n>> "

var PROCESSING_EVENT bool
var CURRENT_EVENT_FLAG string

func Prompt() string {

	// if we are here, no backend events have been detected
	// we can safely display the regular menu's prompt
	fmt.Print(PROMPT)
	var query string
	_, err := fmt.Scanln(&query)

	if err != nil && err.Error() != "unexpected newline" {
		log.Fatal("Error while reading user query in Prompt() : ", err)
	}

	if PROCESSING_EVENT {
		backend_api.GiveInput(CURRENT_EVENT_FLAG, query)
		query = "[BACKEND_OVERRIDE]"
		PROCESSING_EVENT = false
	}
	return query
}

func AskConfirmation(msg string, validation string) bool {
	fmt.Println(msg)
	return Prompt() == validation
}

func ClearTerm() {
	fmt.Print("\033[H\033[2J")
}

func HandleMenuQuery(query string) {

	var acces bdd.AccesBdd

	acces.InitConnection()

	defer acces.CloseConnection()

	switch query {

	case "0":

		fmt.Println(("Starting watcher ..."))
		tasks := acces.ListSyncAllTasks()
		for i := 0; i < tasks.Size(); i++ {
			filesystem.StartWatcher(tasks.Get(i).Path)
		}

	case "1":

		path, err := dialog.Directory().Title("Select Folder").Browse()
		if err != nil {
			fmt.Println("Folder selection cancelled.")
			return
		}

		acces.CreateSync(path)

		fmt.Println("Sync task created. It can be started with the others from the menu.")

	case "2":

		fmt.Println("Select below the sync task you want to provide for another device :")
		tasks := acces.ListSyncAllTasks()
		for i := 0; i < tasks.Size(); i++ {
			task := tasks.Get(i)
			fmt.Println("{")
			fmt.Println("Path : ", task.Path)
			fmt.Println("Secure id : ", task.SecureId)
			fmt.Println("}")
		}

		index, err := strconv.Atoi(Prompt())

		if err != nil {
			log.Fatal("An error occured while scanning for a integer in HandleMenuQuery() : ", err)
		}

		if index > tasks.Size() {
			log.Fatal("The number you provied was not corresponding to any task.")
		}

		acces.GetSecureIdFromRootPath(tasks.Get(index).Path)

		fmt.Println("Mapping available devices on your local network...")

		// list qsync devices across the network
		devices := acces.GetNetworkMap()
		for i := 0; i < devices.Size(); i++ {
			fmt.Printf("[%d] ", i)
			fmt.Println(devices.Get(i)["hostname"])
		}

		// send a link device request to the one the user choose

		index, err = strconv.Atoi(Prompt())

		if err != nil {
			log.Fatal("An error occured while scanning for a integer in HandleMenuQuery() : ", err)
		}

		device_id := devices.Get(index)["device_id"]

		var event globals.QEvent
		event.Flag = "[LINK_DEVICE]"
		event.SecureId = acces.SecureId
		event.FilePath = ""

		var queue globals.GenArray[globals.QEvent]
		queue.Add(event)
		var device_ids globals.GenArray[string]
		device_ids.Add(device_id)

		networking.SendDeviceEventQueueOverNetwork(device_ids, acces.SecureId, queue, devices.Get(index)["ip_addr"])

		// link the device into this db
		acces.LinkDevice(device_id, devices.Get(index)["ip_addr"])
		log.Println("device linked")

		/*log.Println("Press any key once you have put the destination path on your other machine.")
		Prompt()
		// build a custom queue so this device can download all the data contained in your folder
		networking.BuildSetupQueue(acces.SecureId, device_id)*/

		fmt.Println("The selected device has successfully been linked to a sync task.")

	case "3":
		tasks := acces.ListSyncAllTasks()
		for i := 0; i < tasks.Size(); i++ {
			task := tasks.Get(i)
			fmt.Println("{")
			fmt.Println("Path : ", task.Path)
			fmt.Println("Secure id : ", task.SecureId)
			fmt.Println("}")
		}

	case "4":
		// list qsync devices across the network

		devices := acces.GetNetworkMap()
		for i := 0; i < devices.Size(); i++ {
			fmt.Printf("[%d] ", i)
			fmt.Println(devices.Get(i)["hostname"])
		}

	case "5":
		// open QSync store
		go magasin.StartServer()
		time.Sleep(1 * time.Second)
		magasin.OpenUrlInWebBrowser("http://127.0.0.1:8275")

	case "6":

		filepath, err := dialog.File().Title("Select Folder").Load()
		if err != nil {
			fmt.Println("Folder selection cancelled.")
			return
		}
		fmt.Println("Select a device on the network : ")
		devices := acces.GetNetworkMap()
		for i := 0; i < devices.Size(); i++ {
			fmt.Printf("[%d] ", i)
			fmt.Println(devices.Get(i)["hostname"])
		}
		index, err := strconv.Atoi(Prompt())
		if err != nil || index > devices.Size() {
			log.Fatal("Vous n'avez pas saisi un nombre valide !")
		}

		fmt.Println("Sending " + filepath + " to " + devices.Get(index)["hostname"])
		networking.SendLargageAerien(filepath, devices.Get(index)["ip_addr"], false)

	case "7":
		folder_path, err := dialog.Directory().Title("Select Folder").Browse()
		if err != nil {
			fmt.Println("Folder selection cancelled.")
			return
		}
		fmt.Println("Select a device on the network : ")
		devices := acces.GetNetworkMap()
		for i := 0; i < devices.Size(); i++ {
			fmt.Printf("[%d] ", i)
			fmt.Println(devices.Get(i)["hostname"])
		}
		index, err := strconv.Atoi(Prompt())
		if err != nil || index > devices.Size() {
			log.Fatal("Vous n'avez pas saisi un nombre valide !")
		}

		// zipping folder content to send it via largage aerien
		filepath := "multilargage.zip"
		err = globals.ZipFolder(folder_path, filepath)

		if err != nil {
			log.Fatal("Error while zipping folder ", err)
		} else {
			log.Printf("Successfully zipped folder %s into %s\n", folder_path, filepath)
		}

		fmt.Println("Sending " + filepath + " to " + devices.Get(index)["hostname"])
		networking.SendLargageAerien(filepath, devices.Get(index)["ip_addr"], true)

		// now, the zip file is not useful anymore
		//os.Remove(filepath)

	case "8":
		var is_allowing bool

		is_allowing = acces.AreLargageAerienAllowed()

		if is_allowing {
			fmt.Println("You are currently allowing people to send you Largages Aerien.\n\n Changing this to disallow it")
			acces.SwitchLargageAerienAllowingState()
			fmt.Println("Setting changed, you are now disallowing people to send you Largages Aerien.")

		} else {
			fmt.Println("You are currently prohibitng people to send you Largages Aerien.\n\n Changing this to allow it")
			acces.SwitchLargageAerienAllowingState()
			fmt.Println("Setting changed, you are now allowing people to send you Largages Aerien.")

		}

	case "[BACKEND_OVERRIDE]":
		break

	default:
		fmt.Println("This option does not exists :/")
		HandleMenuQuery(Prompt())
	}

}

func DisplayMenu() {

	fmt.Print(LOGO)
	fmt.Print(MENU)

	// interactive events callbacks
	callbacks := make(map[string]func(string))

	callbacks["[CHOOSELINKPATH]"] = func(context string) {
		// simulate new prompt as the real one is displayed before the text
		fmt.Print("\n" + context + "\n\n>> ")
		// don't give back response, as it is handled by the regular prompt-loop
		PROCESSING_EVENT = true
		CURRENT_EVENT_FLAG = "[CHOOSELINKPATH]"

		// wait user input in regular prompt system
		for PROCESSING_EVENT {
			time.Sleep(time.Millisecond * 500)
		}

		// let the backend process and suppress the event file

		time.Sleep(1 * time.Second)
	}

	// air dropping something
	callbacks["[OTDL]"] = func(context string) {
		// simulate new prompt as the real one is displayed before the text
		fmt.Print("\n" + context + "\n\n>> ")

		// don't give back response, as it is handled by the regular prompt-loop
		PROCESSING_EVENT = true
		CURRENT_EVENT_FLAG = "[OTDL]"

		// wait user input in regular prompt system
		for PROCESSING_EVENT {
			time.Sleep(time.Millisecond * 500)
		}

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	callbacks["[MOTDL]"] = func(context string) {
		// simulate new prompt as the real one is displayed before the text
		fmt.Print("\n" + context + "\n\n>> ")

		// don't give back response, as it is handled by the regular prompt-loop
		PROCESSING_EVENT = true
		CURRENT_EVENT_FLAG = "[MOTDL]"

		// wait user input in regular prompt system
		for PROCESSING_EVENT {
			time.Sleep(time.Millisecond * 500)
		}

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	go backend_api.WaitEventLoop(callbacks)

	for {
		HandleMenuQuery(Prompt())
	}

}
