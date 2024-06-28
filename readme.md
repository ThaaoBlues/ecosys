# QSYNC
Qsync is a engine made to provide files synchronisation in real time or not between linked devices.
It automatically detects linked devices on the same network and make sync the wanted folders transparently. 
Qsync also support one-time files transferts (called "Largage Aerien" in reference to AirDrop).
In the long term, the goal is to provide a support for a symbiosis between devices and applications states so one can use any of its devices indistinctly.

On desktop os :
We may achieve this goal by making a qsync addons market where people puts json files with details about how each files is stored in each OS of a specific app. This would allow qsync to adapt and synchronise existing apps. 

On Android : 
As you may know, android has some very strict policies on apps data, so it cannot be accessed by others applications. To know more about how we can still use files to sync applications data, see [android_app.md](https://github.com/ThaaoBlues/qsync/blob/master/android_app.md)

## QSync "Magasin" :
We project to make a store where anyone can publish and/or download QSync addons (named "grapins" in all my frenchness bc it grabs apps data ) and QSync apps (named "tout en un" or 'all in one' in english).

### To sync apps from one device to another, the best is to install the app on both and then take one to link on the other via qsync synchronisations list. 

# How do I know that Qsync already has linked my app from another device ?
Wether you are on android or on desktop, when you download an app you can everything is made for you, from the creation of the folder where you put the files you want to sync to the registration of your app in qsync database. The only thing you must do is on android, as the usage of an alternative app store is not user friendly, you must start qsync app from a specific intent to let it know that the user installed an app that works with qsync. This procedure is specified in the android_app.md file

## More details on how to write an android app that works with Qsync in the android_app.md file
## The following description is usefull for everyone but the examples are for desktop apps


## QSync "grapins" :
Addons are really simple but may evolve as time passes. You must provide a json object with those informations :

```go
type GrapinConfig struct {
	AppName               string
	AppSyncDataFolderPath string
	NeedsFormat           bool
	SupportedPlatforms    []string
	AppDescription        string // well that's the app's descriptions
	AppIconURL            string
}
```

The supported platform must be within ["Linux","Win32","Android"]
NeedsFormat is a flag telling qsync to replace some parts of the application path contextually such as :
* %username% for the user's name
* %version% for an app specific version ( In first versions qsync will just walk in the parent folder and take the first child. This behavior could change if someone find a better one.)
* %any% qsync will just walk in the parent folder and take the first child.

> Path separator must be "/"

## QSync "tout en un" :

### On PC :
"Tout en un" is the ultimate integration with qsync, you can develop your app and integrate qsync capabilities to sync all the data you want across the user's devices.
This is really simple, all the files you want to be synchronised must be within the folder specified by the json description of your app, you are free to use it to do whatever you want. Publish your app on our store and QSync will take care of the rest.

By default, the app will be installed to <qsync_installation_root>/apps/<app_name>
If the folder where the files to sync is not created after we check (and started if any) the installer, qsync will create it.

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
	AppDescription        string // well that's the app's descriptions
	AppIconURL            string
	SupportedPlatforms    []string

}
```
> For examples, see [magasin_database.json](https://raw.githubusercontent.com/ThaaoBlues/qsync/master/magasin_database.json)


### On android :
We use ContentProviders to let apps modify content on their assigned folder in the Qsync private directory.
We list qsync friendly apps on a specific tab in the Qsync app, clicking on it will open its play store page or open the app if already installed.
Apps are listed as normal synchronisations, just the name is the app's name and the string "(application)" is present to let the user know that this is a sync task used by an app. 


#### THOSE ARE RELATIVE PATH : DO NOT PUT YOUR APP FOLDER NAME, JUST PUT ANYTHING BELOW YOUR APP'S ROOT



## QSync "Largage Aérien"
An airdrop-like one time upload that the other device will accept or not.

In this feature, the binary delta is built the same way as the others event, just it will contain a full file.
All the file path fields in the data structure of the event must be only the file's name.
QSync recieve the file in its dedicated folder (<qsync_root>/largage_aerien).
If the user do not want to receive it, nothing is wrote as the file is kept in RAM as a binary delta.
The flag used for the event is "[OTDL]"

### Multiples largages aeriens 
Like the usual "Largage Aeriens" but to send a whole folder under the flag [MOTDL] and will be untar at the end.


## Backup mode
You can activate a backup mode to any synchronisation, it will allow you to delete anything subject to the said synchronisation on your device without the deletion being recorded and sent to others devices.
It is usefull for cases like backing up your pictures and save space on your smartphone.


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

3354HJfjysqgydfk6778Yhgqdsièfoiuhkj(device_id);2556ZJfjgfotydfk6778Yhdddsaèfoiuhkj(secure_id);[OTDL](flag);file(filetype);ab(delta instruction),112(data),114,111,117,116,10,0;test(deltafilepath);test(filepath);(newfilepath);(secure_id->redundancy field not used)

> /!\ these fields are repurposed in some events to transport usefull values when their primary purpose is not usefull



[IN CASE OF A FILE EVENT]

- first, we check if any device is updating this filesystem by checking the receiving_update field of the linked_devices table join sync table

- if an update is occuring : WE IGNORE THE EVENT AND GO BACK TO OUR THINGS --> THE RIDE STOPS HERE

- if not :

- we loop throught the linked_device table join the sync table and check if a device that is connected is linked for this sync task

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

* [OTDL]/[MODTL](.btf) trigerred when user has to choose to accept or not an incoming largage aerien

* [ALERT_USER](.btf) used to alert the user of something

## INFOS DE COMPILATION ET AUTRES BUGS GOLANG
- gomobile n'aime pas les passages par valeur de structures
- gomobile n'aime pas les slice
- j'abandonne gomobile c'est plus simple de tout réécrire en java
- bordel de merde les parser de json sont lent de fou, j'ai du utiliser ma propre serialisation de données



## /!\ we called the sync task id secure_id just because it should avoid collision and path problems, not because it is "secure"

TODO (voir mes notes, sec)

POUR PC :
- Mettre la palette de couleurs sur le magasin
- Tester intensivement les events avec un autre appareil (fichier plus grand, moins grand, tronqué, supprimé ...)
- Finir et tester le launcher/setup
- tester l'ajout/suppression/lancement des applications
- peut-etre utiliser des popup natives pour la confirmation des largages aeriens à la place de la webui

POUR L'APP :
- voir pourquoi rien ne passe dans la table de retard (la liste de offlineDevices est-elle vide ?)
- Fix le fichier qui s'ouvre mal avec la popup
- Tester intensivement la synchronisation
- faire une popup qui invite à télécharger sur ses autres appareils
- mettre un texte quand la list d'appareils est vide
- trouver un moyen de s'assurer qu'une application accedant à son dossier dans le fileProvider est bien légitime
- faire plusieurs apps de base (agenda, notes ...)




- trouver un moyen de sécuriser les communications entre appareils.
	* hypothèse chaque appareil va posséder une clé unique "device_key", donnée de manière symétrique pendant la création du lien entre deux machines. L'identifiant de l'appareil sera la seule donnée non chiffrée dans les échanges, elle permettra d'aller chercher la device_key associée à cet appareil et de déchiffrer le reste du message.
	* l'intégrité du message se vérifiera avec un hash de la totalité du message (device_id compris) qui sera rajouté en fin de requète

	https://fr.wikipedia.org/wiki/Mode_d%27op%C3%A9ration_(cryptographie)#Compteur_avec_code_d%27authentification_de_message_de_cha%C3%AEnage_de_blocs_de_chiffrement_%28CCM%29
	