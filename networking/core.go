package networking

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	backendapi "qsync/backend_api"
	"qsync/bdd"
	"qsync/delta_binaire"
	"strings"
	"time"
)

const HEADER_LENGTH int = 83

type QEvent struct {
	Flag        string
	FileType    string
	Delta       delta_binaire.Delta
	FilePath    string
	NewFilePath string
	SecureId    string
}

func NetWorkLoop() {

	ln, err := net.Listen("tcp", ":8274")
	if err != nil {
		log.Fatal("Error while initializing socket server : ", err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {

			log.Fatal("Error while accepting a client socket connection : ", err)
		}
		go ConnectToDevice(conn)
	}
}

func ConnectToDevice(conn net.Conn) {

	var acces bdd.AccesBdd

	acces.InitConnection()

	// get the device id and secure sync id from header

	header_buff := make([]byte, HEADER_LENGTH)
	_, err := conn.Read(header_buff)

	if err != nil {
		log.Fatal("Error in ConnectToDevice() while reading header")
	}

	//log.Println("Request header : ", string(header_buff))

	var device_id string = strings.Split(string(header_buff), ";")[0]

	var secure_id string = strings.Split(string(header_buff), ";")[1]

	acces.SecureId = secure_id

	// in case of a link packet, the device is not yet registered in the database
	// so it can throw an error
	if acces.IsDeviceLinked(device_id) {
		// makes sure it is marked as connected
		if !acces.GetDeviceConnectionState(device_id) {

			// needs split as RemoteAddr ads port to the address
			acces.SetDeviceConnectionState(device_id, true, strings.Split(conn.RemoteAddr().String(), ":")[0])

		}
	}

	// read the body of the request and store it in a buffer
	var body_buff []byte
	for {
		buffer := make([]byte, 1024) // You can adjust the buffer size as needed
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Fatal("Error in ConnectToDevice() while reading body:", err)
			}
			break
		}
		body_buff = append(body_buff, buffer[:n]...)
	}

	//log.Println("Request body : ", string(body_buff))

	var data QEvent
	err = json.Unmarshal(body_buff, &data)
	if err != nil {
		log.Fatal("Error while parsing request Qevent payload (might be malformed JSON) .")
	}
	// check if this is a regular file event of a special request
	switch string(data.Flag) {

	case "[MODIFICATION_DONE]":
		SetEventNetworkLockForDevice(device_id, false)
	case "[SETUP_DL]":

		// init an event queue with all elements from the root sync directory
		BuildSetupQueue(secure_id, device_id)

	// this event is triggered when another device
	// is trying to link to you with a new sync task that you may not have
	// the device sending this is building a SETUP_DL queue to send at you
	case "[LINK_DEVICE]":
		// as this is triggered by another machine telling this one to create a sync task,
		// we must prepare the environnement to accept this
		// by creating a new sync task with the same path (replace this later by asking to the user)
		// and same secure_id
		log.Println("Initializing env to welcome the other end folder content")
		acces.SecureId = secure_id
		path := backendapi.AskInput("[CHOOSELINKPATH]", "Choose a path where new sync files will be stored.")
		log.Println("Future sync will be stored at : ", path)
		acces.CreateSyncFromOtherEnd(path, secure_id)
		log.Println("Linking device : ", device_id)
		acces.LinkDevice(device_id, strings.Split(conn.RemoteAddr().String(), ":")[0])

	case "[UNLINK_DEVICE]":
		acces.UnlinkDevice(device_id)

	default:

		// regular file event
		HandleEvent(secure_id, device_id, body_buff)
		// send back a modification confirmation, so the other end can remove this machine device_id
		// from concerned sync task retard entries
		buff := []byte(acces.GetMyDeviceId() + ";" + acces.SecureId + ";" + "[MODIFICATION_DONE]")
		conn.Write(buff)
	}

}

// used to process a request when it is a regular file event
func HandleEvent(secure_id string, device_id string, buffer []byte) {

	var event QEvent

	err := json.Unmarshal(buffer, &event)
	if err != nil {
		log.Fatal("Error while decoding json data from request buffer in HandleEvent()", err)
	}

	// first, we lock the filesystem watcher so it don't notify the changes we are doing
	// as it would do a ping-pong effect

	var acces bdd.AccesBdd
	acces.SecureId = secure_id

	acces.InitConnection()

	acces.SetFileSystemPatchLockState(device_id, true)

	// mise de la sync root après le chemin relatif reçu pour pouvoir
	// utiliser directement la variable
	// avant ce bloc, event.FilePath est un chemin relatif vers le fichier.
	relative_path := event.FilePath
	new_relative_path := event.NewFilePath
	event.Delta.FilePath = path.Join(acces.GetRootSyncPath(), event.FilePath)
	event.FilePath = path.Join(acces.GetRootSyncPath(), event.FilePath)

	switch event.Flag {
	case "MOVE":
		acces.Move(relative_path, new_relative_path, event.FileType)
		MoveInFilesystem(event.FilePath, event.NewFilePath)
	case "REMOVE":
		if event.FileType == "file" {
			acces.RmFile(event.FilePath)

		} else {
			acces.RmFolder(event.FilePath)
		}

		RemoveFromFilesystem(event.FilePath)

	case "CREATE":

		log.Println("Creating file : ", event.FilePath)
		if event.FileType == "file" {
			event.Delta.PatchFile()
			acces.CreateFile(relative_path)

		} else {
			os.Mkdir(event.FilePath, 0755)
			acces.CreateFolder(relative_path)

		}

	case "UPDATE":

		acces.IncrementFileVersion(relative_path)
		event.Delta.PatchFile()
	default:
		log.Fatal("Qsync network loop received an unknown event type : ", event)
	}

	acces.SetFileSystemPatchLockState(device_id, false)

}

func SendDeviceEventQueueOverNetwork(connected_devices []string, secure_id string, event_queue []QEvent, ip_addr ...string) {

	// for all devices connected concerned by the sync task, send the data with the right event flag
	// all others are handled in retard database table from the filesystem in a function call right before

	for _, device_id := range connected_devices {
		for _, event := range event_queue {

			log.Println("SENDING EVENT : ", event)

			SetEventNetworkLockForDevice(device_id, true)

			event_json, err := json.Marshal(event)
			if err != nil {
				log.Fatal("An error occured in SendDeviceEventQueueOverNetwork() while creating a json object from the event struct : ", err)
			}

			var acces bdd.AccesBdd
			acces.InitConnection()
			acces.SecureId = secure_id

			// we let the possibility to specify the address in the function arguments
			// as in the case of a [LINK_DEVICE] request, we don't have the IP address registered in the db
			if len(ip_addr) == 0 {
				ip_addr = append(ip_addr, acces.GetDeviceIP(device_id))
			}

			// /!\ the device_id we send is our own so the other end can identify ourselves
			write_buff := []byte(acces.GetMyDeviceId() + ";" + secure_id + string(event_json))

			conn, err := net.Dial("tcp", ip_addr[0]+":8274")

			if err != nil {
				log.Fatal("Error while dialing "+ip_addr[0]+" from SendDeviceEventQueueOverNetwork() : ", err)
			}
			_, err = conn.Write(write_buff)

			if err != nil {
				log.Fatal("Error while writing to "+ip_addr[0]+" from SendDeviceEventQueueOverNetwork() : ", err)
			}

			conn.Close()
			SetEventNetworkLockForDevice(device_id, false)

			// wait for the network lock to be released for this device
			for GetEventNetworkLockForDevice(device_id) {
				time.Sleep(1 * time.Second)
			}
		}
	}

}

func SetEventNetworkLockForDevice(device_id string, value bool) {

	if value {
		file, err := os.Create(device_id + ".nlock")

		if err != nil {
			log.Fatal("Error while creating a network lock file in SetEventNetworkLockForDevice() : ", err)
		}

		file.Close()
	} else {

		err := os.Remove(device_id + ".nlock")

		if err != nil {
			log.Fatal("Error while removing a network lock file in SetEventNetworkLockForDevice() : ", err)
		}

	}

}

func GetEventNetworkLockForDevice(device_id string) bool {

	var acces bdd.AccesBdd
	return acces.IsFile(device_id + ".nlock")

}

func RemoveFromFilesystem(path string) {
	// actually removes the file/folder from the system, not only in database

	stat, err := os.Stat(path)

	if err != nil {
		log.Fatal("Error while removing a file/folder from your system : ", err)
	}

	if stat.IsDir() {
		err = os.RemoveAll(path)

	} else {
		err = os.Remove(path)

	}

	if err != nil {
		log.Fatal("Error while removing a file/folder from your system : ", err)
	}
}

func MoveInFilesystem(old_path string, new_path string) {

	stat, err := os.Stat(old_path)

	if err != nil {
		log.Fatal("An error occured while moving entity in filesystem : ", err)
	}

	// moves a file
	if !stat.IsDir() {
		err = os.Rename(old_path, new_path)
		if err != nil {
			log.Fatal("An error occured while moving entity in filesystem : ", err)

		}

		// moves entire directory
	} else {

		// Check if the source directory exists
		srcInfo, err := os.Stat(old_path)
		if err != nil {
			log.Fatal("An error occured while moving entity in filesystem : ", err)
		}

		// Create the destination directory if it doesn't exist
		if _, err := os.Stat(new_path); os.IsNotExist(err) {
			if err := os.MkdirAll(new_path, srcInfo.Mode()); err != nil {
				log.Fatal("An error occured while moving entity in filesystem : ", err)
			}
		}

		// List the contents of the source directory
		files, err := os.ReadDir(old_path)
		if err != nil {
			log.Fatal("An error occured while moving entity in filesystem : ", err)
		}

		// Move each file and subdirectory
		for _, file := range files {
			srcPath := filepath.Join(old_path, file.Name())
			dstPath := filepath.Join(new_path, file.Name())

			if file.IsDir() {
				MoveInFilesystem(srcPath, dstPath)
			} else {
				err = os.Rename(srcPath, dstPath)
				if err != nil {
					log.Fatal("An error occured while moving entity in filesystem : ", err)
				}
			}
		}

		// Remove the source directory after moving its contents
		os.Remove(old_path)

	}

}

func BuildSetupQueue(secure_id string, device_id string) {

	var bdd bdd.AccesBdd

	bdd.SecureId = secure_id
	bdd.InitConnection()

	rootPath := bdd.GetRootSyncPath()

	var queue []QEvent

	err := filepath.Walk(rootPath, func(absolute_path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal("Error accessing path:", absolute_path, err)
			return err
		}
		relative_path := strings.Replace(absolute_path, rootPath, "", 1)
		if relative_path != "" {
			if info.IsDir() {
				// creates a delta with full file content

				// only keep the relative path

				var event QEvent
				event.Flag = "CREATE"
				event.SecureId = secure_id
				event.FileType = "folder"
				event.FilePath = relative_path

				queue = append(queue, event)

			} else {
				// creates a delta with full file content
				delta := delta_binaire.BuilDelta(relative_path, absolute_path, 0, []byte(""))

				var event QEvent
				event.Flag = "CREATE"
				event.SecureId = secure_id
				event.FileType = "file"
				event.FilePath = relative_path
				event.Delta = delta

				queue = append(queue, event)
			}

		}

		return nil
	})

	//log.Println("setup event queue : ", queue)
	var devices []string
	devices = append(devices, device_id)
	SendDeviceEventQueueOverNetwork(devices, bdd.SecureId, queue)

	if err != nil {
		log.Fatal(err)
	}

}
