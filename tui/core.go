package tui

import (
	"fmt"
	"log"
	"qsync/bdd"
	"qsync/filesystem"
	"qsync/networking"
	"strconv"
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
[5] - 

`

var PROMPT string = "\n>> "

func Prompt() string {

	fmt.Print(PROMPT)
	var query string
	_, err := fmt.Scanln(&query)

	if err != nil {
		log.Fatal("Error while reading user query in Prompt() : ", err)
	}

	return query
}

func HandleMenuQuery(query string) {

	var acces bdd.AccesBdd

	acces.InitConnection()

	switch query {

	case "0":

		fmt.Println(("Starting watcher ..."))
		path := "/home/h3x0/dev/projects/qsync/test_files"

		filesystem.StartWatcher(path)

	case "1":

		fmt.Println("Enter below the path of the folder you want to synchronize :")

		var path string = Prompt()

		acces.CreateSync(path)

	case "2":

		fmt.Println("Enter below the path of the folder you want to synchronize :")

		var path string = Prompt()

		acces.GetSecureId(path)

		fmt.Println("Mapping available devices on your local network...")
		// list qsync devices across the network
		devices := networking.GetNetworkDevices()
		for i := 0; i < len(devices); i++ {
			fmt.Printf("[%d] ", i)
			fmt.Println(devices[i])
		}

		// send a link device packet to the one the user choose

		index, err := strconv.Atoi(Prompt())

		if err != nil {
			log.Fatal("An error occured while scanning for a integer in HandleMenuQuery() : ", err)
		}

		device_id := devices[index]["id"]

		var event networking.QEvent
		event.Flag = "[LINK_DEVICE]"
		event.SecureId = acces.SecureId

		queue := make([]networking.QEvent, 1)
		queue = append(queue, event)

		networking.SendDeviceEventQueueOverNetwork([]string{device_id}, acces.SecureId, queue)

		// link the device into this db
		acces.LinkDevice(device_id)

		// build a custom queue so this device can download all the data contained in your folder
		networking.BuildSetupQueue(acces.SecureId, device_id)

		fmt.Println("The selected device has successfully been linked to a sync task. You may now want to link the other device ")

	case "3":

		for _, task := range acces.ListSyncAllTasks() {
			fmt.Println("{")
			fmt.Println("Path : ", task.Path)
			fmt.Println("Secure id : ", task.SecureId)
			fmt.Println("}")
		}

	case "4":
		// list qsync devices across the network
		devices := networking.GetNetworkDevices()
		for i := 0; i < len(devices); i++ {
			fmt.Printf("[%d] ", i)
			fmt.Println(devices[i])
		}
	default:
		fmt.Println("This option does not exists :/")
		HandleMenuQuery(Prompt())
	}

}

func DisplayMenu() {
	fmt.Print(LOGO)
	fmt.Print(MENU)
	HandleMenuQuery(Prompt())
}
