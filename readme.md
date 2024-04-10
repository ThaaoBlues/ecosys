# QSYNC
Qsync is a engine made to provide files synchronisation in real time or not between linked devices.
It automatically detects linked devices on the same network and make sync the wanted folders transparently. 
In the long term, the goal is to provide a support for a symbiosis between devices and applications states so one can use any of its devices indistinctl.

On desktop os :
We may achieve this goal by making a qsync addons market where people puts json files with details about how each files is stored in each OS of a specific app. This would allow qsync to adapt and synchronise existing apps. 

On Android : 
As you may know, android has some very strict policies on apps data, so it cannot be accessed by others applications. We are thinking about a way to overcome this.

## QSync "Magasin" :
We project to make a store where anyone can publish and/or download QSync addons (named "grapins" in all my frenchness bc it grabs apps data ) and QSync apps (named "tout en un" or 'all in one' in english).


## QSync "grapins" :
Addons are really simple but may evolve as time passes. You must provide a json object with those informations :

```go
type GrapinConfig struct {
	AppName               string
	AppSyncDataFolderPath string
	NeedsFormat           bool
	SupportedPlatforms    []string
}
```

The supported platform must be within ["Linux","Windows","Android"]
NeedsFormat is a flag telling qsync to replace some parts of the application path contextually such as :
* %username% for the user's name
* %version% for an app specific version ( In first versions qsync will just walk in the parent folder and take the first child. This behavior could change if someone find a better one.)
* %any% qsync will just walk in the parent folder and take the first child.

> Path separator must be "/"

## QSync "tout en un" :
"Tout en un" is the ultimate integration with qsync, you can develop your app and integrate qsync capabilities to sync all the data you want across the user's devices.
This is really simple, all the files you want to be synchronised must be within the folder specified by the json description of your app, you are free to use it to do whatever you want. Publish your app on our store and QSync will take care of the rest.

By default, the app will be installed to <qsync_installation_root>/apps


You can specify informations about your app in a json format matching :
All the path asked are relative to your app's root
```go
type ToutEnUnConfig struct {
	AppName               string // well... the app's name ?
	AppDownloadUrl        string // the url where to download the app
	NeedsInstaller        bool   // if we need to run the binary installer
	AppLauncherPath       string // the path to the main executable of your app 
	AppInstallerPath      string // the installer path
	AppUninstallerPath    string // the uninstaller path
	AppSyncDataFolderPath string // the folder where the data to synchronize is stored
}
```



## base de données
- on cartographie le systeme de fichiers visés et l'enregistre une première fois
- on met tout en mode création dans la bdd
- dans la table, il y a plusieurs versions d'un mme fichier suivant l'état de synchronisation d'autres appareils
- si un appareil est en retard de n versions, on lui envoie les deltas 1 par 1 en supprimant de la bdd le delta de la version qui vient d'etre patch si aucun autre appareil n'est aussi en retard. On ne stoque que la dernière version du fichier de manière complète dans la bdd (utilisée pour attached de nouvelles machines ou simplement calculer le delta de la dernière version), les versions spécifiques aux appareils en retard n'ont que le delta binaire correspondant de stoqué.


table retard
---------------------------------------------------------------------------
ID | version_id | file_path       | mod_type | devices_to_patch          | type   |
1  | 123        | "test/test.log" | "p"      | "238532123;2347668378"    | "file" |

mod_types : 
- c -> creation, just send the entire file directly from user's filesystem, no need of delta
- d -> delete, remove the file from the other's device filesystem
- p -> patch, only modification types that needs a delta to patch the remote file.

devices_to_patch :
- liste d'identifiants uniques des ordinateurs à patch, chaque id unique est conservé sur la machine concernée et envoyé à la demande de syncronisation ou d'attachement


table delta (if mod type is p)
-----------------------------------------------------------------------
ID | path                | version_id | delta         |
1  |   "test/test.log"   |124         | [{},{},{}]    |

table filesystem
-----------------------------------------------------------------------
ID | path            | version_id | type    | size | secure_id                  | data                 |
1  | "test"          |   0        |"folder" | 0    | "hx9x3587545ag675gqs891g"  | NULL                 |
2  | "test/test.log" |   124      | "file"  | 1738 | "hx9x3587545ag675gqs891g"  | the content in bytes |




table sync
-----------------------------------------------------------------------
ID | secure_id                 | linked_devices_id     | root      |
1  | "hx9x3587545ag675gqs891g" | "238532123;234766837" | "C:/test" |

table linked_devices
-----------------------------------------------------------------------
ID | device_id | is_connected | receiving_update                |
1  | 238532123 | true         | {"hx9x3587545ag675gqs891g":true}|
2  | 234766837 | false        |{"hx9x3587545ag675gqs891g":false}|


secure_id : identifiant du système de fichiers concerné par la tache de syncronisation




## communications :

- Each device uses mDNS with a unique device_id as additionnal data
    * service name : ._qsync._tcp

- When a device is finding a qsync zeroconf service, it checks its own database to see if the given device_id corresponds to a/some sync task(s)

- When the zeroconf service library gets a service close event, we set de device is_connected state to false

- if a device that was previously marked as not connected is here, we update the database and set is_connected to true

- Hop in bitch ! We are sending him an Hello packet 

- If the Hello packet succeed,

- Then, we send all pending updates in the "retard" table that mentions this device_id
- We remove the concerned device_id from all mentions on "retard" table 
- Those actions are actually made one version-delta update with one retard mention erase etc...

- Else we set his is_connected state to false

We don't need to maintain the socket between events, we close it and the machine that needs to say something will connect to the other if it is still marked as connected.

Each request will contain an header of 256 bit that will represents the device_id of the sender

The typical request must look like :

3354HJfjysqgydfk6778Yhgqdsièfoiuhkj(device_id);2556ZJfjgfotydfk6778Yhdddsaèfoiuhkj(secure_id)type QEvent struct {
	Flag          string
	delta         delta_binaire.Delta
	file_path     string
	new_file_path string
	sync_id       string
} (json string)

no newline between device_id and QEvent data



[IN CASE OF A FILE EVENT]

- first, we check if any device is updating this filesystem by checking the receiving_update field of the linked_devices table join sync table

- if an update is occuring : WE IGNORE THE EVENT AND GO BACK TO OUR THINGS --> THE RIDE STOPS HERE

- if not :

- we loop throught the linked_device table join the sync table and check if a device that is connected is linked for this sync task


- If yes, we send an Hello packet

- If the Hello packet succeed,

- If no disconnection is done while transferring data, 

- We are sending him the update

- Else, an error occurred or the device is not marked as connected, we add a line in the "retard" table if it does not already exists for this event, if it already exists we just append the device_id to the devices_to_patch list



[WHEN SENDING AN EVENT QUEUE]
- loop through the targets ids
- loop through each event of the queue
- lock the network for the current targeted device
- send the event
- wait for the lock to be released ( by a [MODIFICATION_DONE]  event )
- continue the loop



[IN CASE OF A FILE EVENT PACKET RECEIVED]

- If we haven't done it by a zeroconf event, we set the device is_connected state to true if it is on the linked_devices table AND WE SET THE RECEIVING_UPDATE VARIABLE TO TRUE SO THAT ALL MODIFICATIONS EVENT ARE IGNORED WHILE WE ARE PATCHING THE FILESYSTEM

- then we patch the filesystem

- and we release the lock by setting the receiving_update variable of the given device to false


[CREATE LINK PACKED RECEIVED]
- add this device and its id to the database

[REMOVE LINK PACKED RECEIVED]
- remove this device from database


[SETUP PACKED RECEIVED]
- Loop through all required files and build a setup download queue for the target distant machine


To avoid all conflicts of path, the secure_id is shared between when you link a device and will be used to identify the correct sync task


## BACK-TO-FRONT COMMUNICATION
when a request is treated by the backend and it necessitate an user input, it creates a "<flag>.btf" file
with a context inside that can be displayed to the user. You just have to append write the user response directly after the context(no newline in between).
As example, here is a program that shows a bit how it is working :
```go

	// backend
	go func() {
		// here, "test" will be the event flag
		user_data := backendapi.AskInput("test", "Hey ! I need you to write me your name !")
		fmt.Println("user_data : ", user_data)
	}()


	// let the backend ask the user input (overkill)
	time.Sleep(1 * time.Second)

	// get context outside of the callback function (will crash if no inputs are asked)
	fmt.Println("contexte outside of callback: ", backendapi.ReadInputContext("test"))


	// mostly used : using callback functions
	callbacks := make(map[string]func(string))

	// the map keys are the flag ! don't put anything random
	callbacks["test"] = func(context string) {
		fmt.Println("context in callback : ", context)
		// give to backend the user response
		backendapi.GiveInput("test", "Josette")
	}

	backendapi.WaitEventLoop(callbacks) // can be put into a goroutine

```

### events flag (files to watch):
* [CHOOSELINKPATH](.btf) triggered when user has to choose a path that to receive the files of a sync task on another device

## Example of usage of qsync is shown as a basic synchronisation app with 'main.go' and 'tui/core.go' file:

> main.go
```go
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


```

> tui/core.go
```go

package tui

import (
	"fmt"
	"log"
	backendapi "qsync/backend_api"
	"qsync/bdd"
	"qsync/filesystem"
	"qsync/globals"
	"qsync/networking"
	"strconv"
	"time"
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

	if err != nil && err.Error() != "unexpected newline" {
		log.Fatal("Error while reading user query in Prompt() : ", err)
	}

	return query
}

func AskConfirmation(msg string, validation string) bool {
	fmt.Println(msg)
	return Prompt() == validation
}

func HandleMenuQuery(query string) {

	var acces bdd.AccesBdd

	acces.InitConnection()

	switch query {

	case "0":

		fmt.Println(("Starting watcher ..."))
		for _, task := range acces.ListSyncAllTasks() {
			filesystem.StartWatcher(task.Path)
		}

	case "1":

		fmt.Println("Enter below the path of the folder you want to synchronize :")

		var path string = Prompt()

		acces.CreateSync(path)

		fmt.Println("Sync task created. It can be started with the others from the menu.")

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

		device_id := devices[index]["device_id"]

		var event globals.QEvent
		event.Flag = "[LINK_DEVICE]"
		event.SecureId = acces.SecureId
		event.FilePath = path

		queue := []globals.QEvent{event}

		networking.SendDeviceEventQueueOverNetwork([]string{device_id}, acces.SecureId, queue, devices[index]["ip_addr"])

		// link the device into this db
		acces.LinkDevice(device_id, devices[index]["ip_addr"])
		log.Println("device linked")

		log.Println("Press any key once you have put the destination path on your other machine.")
		Prompt()
		// build a custom queue so this device can download all the data contained in your folder
		networking.BuildSetupQueue(acces.SecureId, device_id)

		fmt.Println("The selected device has successfully been linked to a sync task.")

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

	// interactive events callbacks
	callbacks := make(map[string]func(string))

	callbacks["[CHOOSELINKPATH]"] = func(context string) {
		fmt.Println(context + " : ")
		backendapi.GiveInput("[CHOOSELINKPATH]", Prompt())
		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	go backendapi.WaitEventLoop(callbacks)

	for {
		HandleMenuQuery(Prompt())
	}

}


```






## /!\ we called the sync task id secure_id just because it should avoid collision and path problems, not because it is "secure"

TODO
- Tester intensivement les events avec un autre appareil (fichier plus grand, moins grand, tronqué, supprimé ...)

- faire les fonctions d'ajout et d'installations d'app tout en un et de grapins
- ajouter une table avec les applications installées dans la bdd
- ajouter une page pour lancer les applications installées dans le magasin
- créer un racourcis vers les applications à l'installation


- trouver un moyen de sécuriser les communications entre appareils.
	* hypothèse chaque appareil va posséder une clé unique "device_key", donnée de manière symétrique pendant la création du lien entre deux machines. L'identifiant de l'appareil sera la seule donnée non chiffrée dans les échanges, elle permettra d'aller chercher la device_key associée à cet appareil et de déchiffrer le reste du message.
	* l'intégrité du message se vérifiera avec un hash de la totalité du message (device_id compris) qui sera rajouté en fin de requète

	https://fr.wikipedia.org/wiki/Mode_d%27op%C3%A9ration_(cryptographie)#Compteur_avec_code_d%27authentification_de_message_de_cha%C3%AEnage_de_blocs_de_chiffrement_%28CCM%29
	