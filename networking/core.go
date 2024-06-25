package networking

import (
	"bytes"
	"io"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"qsync/backend_api"
	"qsync/bdd"
	"qsync/delta_binaire"
	"qsync/globals"
	"strings"
	"time"

	"github.com/skratchdot/open-golang/open"
)

const HEADER_LENGTH int = 83

func NetWorkLoop() {

	ln, err := net.Listen("tcp", ":8274")
	if err != nil {
		log.Fatal("Error while initializing socket server : ", err)
	}
	for {
		conn, err := ln.Accept()
		log.Println("Accepted client")
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
	// prevent the fucking java sockets to break my app
	// because it fucking sends zeros before the messages
	// java go fuck yourself.
	padding_buff := make([]byte, 1)
	padding_buff[0] = 0
	for padding_buff[0] == 0 {
		_, err := conn.Read(padding_buff)
		if err != nil {
			log.Println("Error in ConnectToDevice() while reading header, request must be malformed.")
			return
		}
	}

	_, err := conn.Read(header_buff)
	log.Println("HEADER BUFF", header_buff)

	// as the padding got the first element of the header, we must shift the header slice by one
	header_buff = append([]byte{padding_buff[0]}, header_buff...)
	header_buff = header_buff[:len(header_buff)-1]

	if err != nil {
		log.Fatal("Error in ConnectToDevice() while reading header")
	}

	//log.Println("Request header : ", string(header_buff))
	var device_id string
	var secure_id string
	if len(strings.Split(string(header_buff), ";")) == 2 {
		device_id = strings.Split(string(header_buff), ";")[0]

		secure_id = strings.Split(string(header_buff), ";")[1]
	} else {
		log.Println("A malformed request has been refused.")
		return
	}

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

	// append the first char of the event flag as the header shift got it erased
	// OUI C'EST DU BRICOLAGE OKAY
	init := []byte("[")
	body_buff := bytes.NewBuffer(init)

	_, err = body_buff.ReadFrom(conn)
	if err != nil {
		if err != io.EOF {
			log.Fatal("Error in ConnectToDevice() while reading body:", err)
		}
		//break
	}

	//log.Println("Request body : ", string(body_buff))

	var data globals.QEvent = globals.DeSerializeQevent(body_buff.String())

	// check if this is a regular file event of a special request
	//log.Println("DECODED EVENT : ", data)
	switch string(data.Flag) {

	case "[MODIFICATION_DONE]":
		SetEventNetworkLockForDevice(device_id, false)
	case "[SETUP_DL]":
		if acces.IsDeviceLinked(device_id) {
			log.Println("GOT FLAG, BUILDING SETUP QUEUE...")

			// init an event queue with all elements from the root sync directory
			BuildSetupQueue(secure_id, device_id)
		}

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

		path := backend_api.AskInput("[CHOOSELINKPATH]", "Choose a path where new sync files will be stored.")
		log.Println("Future sync will be stored at : ", path)
		acces.CreateSyncFromOtherEnd(path, secure_id)
		log.Println("Linking device : ", device_id)
		acces.LinkDevice(device_id, strings.Split(conn.RemoteAddr().String(), ":")[0])

		// now that we are ready, ask for the sauce ;)
		var queue globals.GenArray[globals.QEvent]
		var event globals.QEvent
		event.Flag = "[SETUP_DL]"
		event.SecureId = secure_id

		queue.Add(event)

		// list of device_id used to set network lock
		var dummy_device globals.GenArray[string]

		dummy_device.Add(device_id)
		SendDeviceEventQueueOverNetwork(dummy_device, secure_id, queue, strings.Split(conn.RemoteAddr().String(), ":")[0])

	case "[UNLINK_DEVICE]":
		acces.UnlinkDevice(device_id)

	case "[OTDL]":

		// goroutine because it will later ask and wait approval for the user
		go HandleLargageAerien(data, conn.RemoteAddr().String())

	case "[MOTDL]":

		// goroutine because it will later ask and wait approval for the user
		go HandleMultipleLargageAerien(data, conn.RemoteAddr().String())

	default:

		// regular file event
		HandleEvent(secure_id, device_id, data)
		// send back a modification confirmation, so the other end can remove this machine device_id
		// from concerned sync task retard entries
		buff := []byte(acces.GetMyDeviceId() + ";" + acces.SecureId + ";" + "[MODIFICATION_DONE]")
		conn.Write(buff)
	}

}

// used to process a request when it is a regular file event
func HandleEvent(secure_id string, device_id string, event globals.QEvent) {

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
	case "[MOVE]":
		acces.Move(relative_path, new_relative_path, event.FileType)
		MoveInFilesystem(event.FilePath, event.NewFilePath)
	case "[REMOVE]":
		if event.FileType == "file" {
			acces.RmFile(event.FilePath)

		} else {
			acces.RmFolder(event.FilePath)
		}

		RemoveFromFilesystem(event.FilePath)

	case "[CREATE]":

		log.Println("Creating file : ", event.FilePath)
		if event.FileType == "file" {
			event.Delta.PatchFile()
			acces.CreateFile(relative_path, filepath.Join(acces.GetRootSyncPath(), relative_path), "[SENT_FROM_OTHER_DEVICE]")

		} else {
			os.Mkdir(event.FilePath, 0755)
			acces.CreateFolder(relative_path)

		}

	case "[UPDATE]":

		acces.IncrementFileVersion(relative_path)
		event.Delta.PatchFile()
	default:
		log.Fatal("Qsync network loop received an unknown event type : ", event)
	}

	acces.SetFileSystemPatchLockState(device_id, false)

}

func SendDeviceEventQueueOverNetwork(connected_devices globals.GenArray[string], secure_id string, event_queue globals.GenArray[globals.QEvent], ip_addr ...string) {

	// for all devices connected concerned by the sync task, send the data with the right event flag
	// all others are handled in retard database table from the filesystem in a function call right before

	for i := 0; i < connected_devices.Size(); i++ {
		device_id := connected_devices.Get(i)
		for i := 0; i < event_queue.Size(); i++ {
			event := event_queue.Get(i)

			SetEventNetworkLockForDevice(device_id, true)

			formatted_event := globals.SerializeQevent(event)

			var acces bdd.AccesBdd
			acces.InitConnection()
			acces.SecureId = secure_id

			// we let the possibility to specify the address in the function arguments
			// as in the case of a [LINK_DEVICE] request, we don't have the IP address registered in the db
			if len(ip_addr) == 0 {
				ip_addr = append(ip_addr, acces.GetDeviceIP(device_id))
			}

			// /!\ the device_id we send is our own so the other end can identify ourselves
			write_buff := []byte(acces.GetMyDeviceId() + ";" + secure_id + string(formatted_event))

			conn, err := net.Dial("tcp", ip_addr[0]+":8274")

			if err != nil {
				log.Fatal("Error while dialing "+ip_addr[0]+" from SendDeviceEventQueueOverNetwork() : ", err)
			}
			_, err = conn.Write(write_buff)

			if err != nil {
				log.Fatal("Error while writing to "+ip_addr[0]+" from SendDeviceEventQueueOverNetwork() : ", err)
			}

			conn.Close()

			log.Println("Event sent !")
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
		file, err := os.Create(filepath.Join(globals.QSyncWriteableDirectory, device_id+".nlock"))

		if err != nil {
			log.Fatal("Error while creating a network lock file in SetEventNetworkLockForDevice() : ", err)
		}

		file.Close()
	} else {

		err := os.Remove(filepath.Join(globals.QSyncWriteableDirectory, device_id+".nlock"))
		log.Println("removing network lock after sending event")
		if err != nil {
			log.Fatal("Error while removing a network lock file in SetEventNetworkLockForDevice() : ", err)
		}

	}

}

func GetEventNetworkLockForDevice(device_id string) bool {

	var acces bdd.AccesBdd
	return acces.IsFile(filepath.Join(globals.QSyncWriteableDirectory, device_id+".nlock"))

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

	var acces bdd.AccesBdd

	acces.SecureId = secure_id
	acces.InitConnection()

	rootPath := acces.GetRootSyncPath()

	var queue globals.GenArray[globals.QEvent]

	var devices globals.GenArray[string]
	devices.Add(device_id)

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

				var event globals.QEvent
				event.Flag = "[CREATE]"
				event.SecureId = secure_id
				event.FileType = "folder"
				event.FilePath = relative_path

				queue.Add(event)

			} else {
				// creates a delta with full file content
				delta := delta_binaire.BuilDelta(relative_path, absolute_path, 0, []byte(""))

				var event globals.QEvent
				event.Flag = "[CREATE]"
				event.SecureId = secure_id
				event.FileType = "file"
				event.FilePath = relative_path
				event.Delta = delta

				queue.Add(event)
			}

			// We must send events one by one even if it is Network-heavy to not
			// overflow the ram when multiple files are in the folder
			SendDeviceEventQueueOverNetwork(devices, acces.SecureId, queue)
			queue.PopLast()

		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

}

func HandleLargageAerien(data globals.QEvent, ip_addr string) {
	// makes sure we are not given a path for some reasons
	file_name := filepath.Base(data.Delta.FilePath)

	backend_api.NotifyDesktop("Incoming Largage Aérien !! " + "(coming from " + ip_addr + ") \n File name : " + file_name)
	user_response := backend_api.AskInput("[OTDL]", "Accept the largage aérien ? (coming from "+ip_addr+") \n File name : "+file_name+"\nFile would be saved to the folder : "+filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien\n\n"))
	if user_response == "1" || user_response == "true" || user_response == "y" || user_response == "Y" || user_response == "yes" || user_response == "YES" || user_response == "oui" {
		// make sure we have the right directory set-up
		ex, err := globals.Exists(filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien"))

		if err != nil {
			log.Fatal("Error while trying to check if the largage_aerien folder exsists in HandleLargageAerien() : ", err)

		}

		if !ex {
			os.Mkdir(filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien"), 0775)
		}

		// build the path to the largage_aerien folder
		data.Delta.FilePath = filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien", file_name)

		// write the file. As this is probably a full file, the binary delta is just the file content
		data.Delta.PatchFile()
		err = open.Run(data.Delta.FilePath)
		if err != nil {
			open.Run(filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien"))
		}

	}
}

func HandleMultipleLargageAerien(data globals.QEvent, ip_addr string) {
	// makes sure we are not given a path for some reasons
	file_name := filepath.Base(data.Delta.FilePath)

	backend_api.NotifyDesktop("Incoming Largage Aérien !! " + "(coming from " + ip_addr + ") \n File name : " + file_name)
	user_response := backend_api.AskInput("[MOTDL]", "Accept the MULTIPLE largage aérien ? (coming from "+ip_addr+") \n File name : "+file_name+"\nFile would be saved to the folder : "+filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien\n\n"))
	if user_response == "1" || user_response == "true" || user_response == "y" || user_response == "Y" || user_response == "yes" || user_response == "YES" || user_response == "oui" {
		// make sure we have the right directory set-up
		ex, err := globals.Exists(filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien"))

		if err != nil {
			log.Fatal("Error while trying to check if the largage_aerien folder exsists in HandleLargageAerien() : ", err)

		}

		if !ex {
			os.Mkdir(filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien"), 0775)
		}

		// build the path to the largage_aerien folder
		data.Delta.FilePath = filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien", file_name)

		// write the file. As this is probably a full file, the binary delta is just the file content
		data.Delta.PatchFile()

		//unpacking tar file in a randomly generated folder with date
		now := time.Now()
		date_str := now.Format("2006-01-02-15h04min-05s")

		folder_name := file_name + "_" + date_str
		folder_path := filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien", folder_name)
		globals.UntarFolder(data.Delta.FilePath, folder_path)

		open.Run(folder_path)

	}
}

func SendLargageAerien(file_path string, device_ip string, multiple bool) {

	var queue globals.GenArray[globals.QEvent]
	file_name := filepath.Base(file_path)

	// creates a delta with full file content
	delta := delta_binaire.BuilDelta(file_name, file_path, 0, []byte(""))

	var event globals.QEvent
	if multiple {
		event.Flag = "[MOTDL]"
	} else {
		event.Flag = "[OTDL]"
	}
	event.SecureId = "le_ciel_me_tombe_sur_la_tete_000000000000"
	event.FileType = "file"
	event.FilePath = file_name
	event.Delta = delta

	queue.Add(event)

	// not used list of device_id
	var dummy_device globals.GenArray[string]
	// it still needs to have the size of the number of ip addresses we want to use
	// so we add the device ip addr as placeholder
	dummy_device.Add(device_ip)
	SendDeviceEventQueueOverNetwork(dummy_device, "le_ciel_me_tombe_sur_la_tete_000000000000", queue, device_ip)
}

func IsNetworkAvailable() bool {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		conn.Close()
	}
	return err == nil
}
