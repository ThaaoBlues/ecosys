## delta binaire


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

## /!\ we called the sync task id secure_id just because it should avoid collision a path problems, not because it is "secure"

TODO
- Tester avec un autre appareil
- Tester la fonction UnlinkDevice