/*
 * @file            networking/core.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2023-09-11 14:08:11
 * @lastModified    2024-08-26 22:42:25
 * Copyright ©Théo Mougnibas All rights reserved
 */

package networking

import (
	"bufio"
	"bytes"
	"ecosys/backend_api"
	"ecosys/bdd"
	"ecosys/delta_binaire"
	"ecosys/globals"
	"io"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
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

	//now, for faster data processing, we will setup a buffered reader

	reader := bufio.NewReader(conn)

	// get the device id and secure sync id from header

	header_buff := make([]byte, HEADER_LENGTH)
	// prevent the fucking java sockets to break my app
	// because it fucking sends zeros before the messages
	// java go fuck yourself.
	padding_buff := make([]byte, 1)
	padding_buff[0] = 0
	for padding_buff[0] == 0 {
		_, err := reader.Read(padding_buff)

		if err != nil {
			log.Println("Error in ConnectToDevice() while reading header, request must be malformed.")
			return
		}
	}

	_, err := reader.Read(header_buff)
	//log.Println("HEADER BUFF", header_buff)

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
		log.Println("header_buff = ", header_buff)
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
	init := []byte("")

	body_buff := bytes.NewBuffer(init)

	_, err = body_buff.ReadFrom(reader)
	if err != nil {
		if err != io.EOF {
			log.Fatal("Error in ConnectToDevice() while reading body:", err)
		}
		//break
	}

	var data globals.QEvent = globals.DeSerializeQevent(body_buff.String(), secure_id)
	log.Println("EVENT FLAG : " + data.Flag)

	// check if this is a regular file event of a special request
	//log.Println("EVENT : ", data)
	switch string(data.Flag) {

	case "[MODIFICATION_DONE]":

		version_id, err := strconv.ParseInt(data.FileType, 10, 64)
		if err != nil {
			log.Fatal("Error while parsing file version_id in [MODIFICATION_DONE] : ", err)
		}

		acces.RemoveDeviceFromRetardOneFile(
			device_id,
			data.FilePath,
			version_id,
		)

	case "[BEGIN_UPDATE]":

		if acces.IsDeviceLinked(device_id) {
			log.Println("LOCKING FILESYSTEM")
			acces.SetFileSystemPatchLockState(true)
		}

	case "[END_OF_UPDATE]":
		/*if acces.IsDeviceLinked(device_id) {
			log.Println("UNLOCKING FILESYSTEM")
			acces.SetFileSystemPatchLockState(device_id, false)
		}*/

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

		var path string
		if data.FileType == "[APPLICATION]" {

			path = filepath.Join(globals.EcosysWriteableDirectory, "apps", data.FilePath)
			// check if the app is downloaded
			if !globals.ExistsInFilesystem(path) {
				// replace the original secure_id generated for the app
				// by the one from the other device so we can link them
				acces.SecureId = backend_api.AskInput(
					"[CHOOSE_APP_TO_LINK]",
					"",
				)
				path = acces.GetRootSyncPath()

			}

			remote_task_creation_date, err := strconv.ParseInt(data.NewFilePath, 10, 64)

			if err != nil {
				log.Fatal("Error while parsing task creation date from request : ", err)
			}

			local_task_creation_date := acces.GetSyncCreationDateFromPathMatch(path)

			if remote_task_creation_date < local_task_creation_date {
				acces.UpdateSyncId(path, secure_id)
			} else {

				// task is older than remote one, sending link request
				// the other way around- Faire une app mobile pour tester les intents et une belle synchro pc-android

				log.Println("Sync task on this machine is older than the remote one, sending request to invert the link procedure...")
				acces.GetSecureIdFromRootPathMatch(path)

				ip_addr := acces.GetDeviceIP(device_id)
				acces.LinkDevice(device_id, ip_addr)

				var event globals.QEvent
				event.Flag = "[LINK_DEVICE]"
				event.SecureId = acces.SecureId
				event.FilePath = "[APPLICATION]"
				event.NewFilePath = strconv.FormatInt(local_task_creation_date, 10)

				var queue globals.GenArray[globals.QEvent]
				queue.Add(event)
				var device_ids globals.GenArray[string]
				device_ids.Add(device_id)

				SendDeviceEventQueueOverNetwork(device_ids, acces.SecureId, queue, ip_addr)

				return
			}

		} else {
			path = backend_api.AskInput("[CHOOSELINKPATH]", "Choose a path where new sync files will be stored.")

			if path != "[CANCELLED]" {
				log.Println("Future sync will be stored at : ", path)
				acces.CreateSyncFromOtherEnd(path, secure_id)

			}

		}

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

		if data.FileType == "[APPLICATION]" {
			_ = backend_api.AskInput("[ALERT_USER]",
				"Applications successfully linked to each others ! Please restart ecosys before changing anything",
			)
		}
	case "[UNLINK_DEVICE]":
		acces.UnlinkDevice(device_id)

	case "[OTDL]":

		// goroutine because it will later ask and wait approval for the user
		go HandleLargageAerien(data, conn.RemoteAddr().String())

	case "[MOTDL]":

		// goroutine because it will later ask and wait approval for the user
		go HandleMultipleLargageAerien(data, conn.RemoteAddr().String())

	default:

		log.Println("DEFAULT")

		// check if update has already been made or not
		// as multiple devices may send the same patch
		acces.SecureId = secure_id

		switch data.FileType {
		case "file":

			// voir ici parce que dans certains evenements ça pourrait foirer
			// par exemple un remove puis un update reçu en retard ne passerait pas
			// et resterait dans les retards chez l'autre  :/
			if acces.CheckFileExists(data.FilePath) {
				// remove event has always a version_id of 0
				if (acces.GetFileLastVersionId(data.FilePath) > data.VersionToPatch) && (data.Flag != "[REMOVE]") {
					// don't do outdated modifications
					go func() {
						sendModificationDoneEvent(device_id, data.Flag, secure_id, data.FilePath, data.NewFilePath, acces, conn)
					}()
					return
				}
			} else {
				// un ev qui arrive en retard apres une suppression
				if data.Flag != "[CREATE]" {
					// don't do outdated modifications
					go func() {
						sendModificationDoneEvent(device_id, data.Flag, secure_id, data.FilePath, data.NewFilePath, acces, conn)
					}()
					return
				}
			}

		// pareil, à voir ici

		case "folder":
			// as a folder event is always related with moves/deletions or creation
			if acces.CheckFileExists(data.FilePath) == (data.Flag == "[CREATE]") {
				// don't do outdated modifications
				go func() {
					sendModificationDoneEvent(device_id, data.Flag, secure_id, data.FilePath, data.NewFilePath, acces, conn)
				}()
				return
			}
		default:

		}

		//store the event for offline linked devices
		// so the update does not have to come from the same device for every others
		acces.StoreReceivedEventForOthersDevices(data)

		// handle the regular file event
		HandleEvent(secure_id, device_id, data, conn)

	}

	//conn.Close()

}

// used to process a request when it is a regular file event
func HandleEvent(secure_id string, device_id string, event globals.QEvent, conn net.Conn) {

	// first, we lock the filesystem watcher so it don't notify the changes we are doing
	// as it would do a ping-pong effect
	//log.Println(event)
	var acces bdd.AccesBdd
	acces.SecureId = secure_id

	acces.InitConnection()

	acces.SetFileSystemPatchLockState(true)

	// mise de la sync root après le chemin relatif reçu pour pouvoir
	// utiliser directement la variable
	// avant ce bloc, event.FilePath est un chemin relatif vers le fichier.
	relative_path := event.FilePath

	// as in backup mode, files can be supressed freely
	// the remote device can still have a file that no longer exists
	// in this filesystem
	if !(acces.IsSyncInBackupMode() && !acces.IsFile(relative_path)) {
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

			//first, check if all folders leading to the file are present
			dirs := strings.Split(relative_path, "/")
			tmp := acces.GetRootSyncPath()
			for i := 0; i < len(dirs)-1; i++ {
				tmp := path.Join(tmp, dirs[i])
				if !globals.ExistsInFilesystem(tmp) {
					os.Mkdir(tmp, 0755)
				}
			}

			// then, do the file creation
			if event.FileType == "file" {
				event.Delta.PatchFile()
				acces.CreateFile(relative_path, event.FilePath, "[SENT_FROM_OTHER_DEVICE]")

			} else {
				os.Mkdir(event.FilePath, 0755)
				acces.CreateFolder(relative_path)

			}

		case "[UPDATE]":

			acces.IncrementFileVersion(relative_path)
			acces.UpdateCachedFile(relative_path, event.FilePath)
			event.Delta.PatchFile()
		default:
			log.Fatal("ecosys network loop received an unknown event type : ", event)
		}
	}

	go func() {
		// wait for last event to be detected and skipped by filesystem watcher
		time.Sleep(2 * time.Second)
		acces.SetFileSystemPatchLockState(false)
		// send back a modification confirmation, so the other end can remove this machine device_id
		// from concerned sync task retard entries
		sendModificationDoneEvent(device_id, event.Flag, secure_id, relative_path, event.NewFilePath, acces, conn)
	}()

}

func sendModificationDoneEvent(device_id string, flag string, secure_id string, relative_path string, new_file_path string, acces bdd.AccesBdd, conn net.Conn) {
	var ev globals.QEvent
	ev.FilePath = relative_path
	ev.SecureId = secure_id
	switch flag {
	case "[MOVE]":
		// event.NewFilePath is still relative
		ev.FileType = strconv.FormatInt(acces.GetFileLastVersionId(new_file_path), 10)

	case "[REMOVE]":
		ev.FileType = "0"

	default:
		ev.FileType = strconv.FormatInt(acces.GetFileLastVersionId(relative_path), 10)

	}
	ev.Flag = "[MODIFICATION_DONE]"
	var connected_devices globals.GenArray[string]
	connected_devices.Add(device_id)
	var event_queue globals.GenArray[globals.QEvent]
	event_queue.Add(ev)
	ip_addr := strings.Split(conn.RemoteAddr().String(), ":")[0]

	// to avoid reusing addr
	conn.Close()

	SendDeviceEventQueueOverNetwork(connected_devices, acces.SecureId, event_queue, ip_addr)
}

func SendStartUpdateEvent(secure_id string, ip_addr string) {

	var update_start_event globals.QEvent

	update_start_event.Flag = "[BEGIN_UPDATE]"
	update_start_event.SecureId = secure_id

	var acces bdd.AccesBdd
	acces.InitConnection()
	// /!\ the device_id we send is our own so the other end can identify ourselves
	write_buff := []byte(acces.GetMyDeviceId() + ";" + secure_id + globals.SerializeQevent(update_start_event))

	conn, err := net.Dial("tcp", ip_addr+":8274")

	if err != nil {
		log.Fatal("Error while dialing "+ip_addr+" from SendDeviceEventQueueOverNetwork() : ", err)
	}
	_, err = conn.Write(write_buff)

	if err != nil {
		log.Fatal("Error while writing to "+ip_addr+" from SendDeviceEventQueueOverNetwork() : ", err)
	}

	conn.Close()

	acces.CloseConnection()
}

func SendStopUpdateEvent(secure_id string, ip_addr string) {

	var update_stop_event globals.QEvent

	update_stop_event.Flag = "[END_OF_UPDATE]"
	update_stop_event.SecureId = secure_id

	var acces bdd.AccesBdd
	acces.InitConnection()
	// /!\ the device_id we send is our own so the other end can identify ourselves
	write_buff := []byte(acces.GetMyDeviceId() + ";" + secure_id + globals.SerializeQevent(update_stop_event))

	conn, err := net.Dial("tcp", ip_addr+":8274")

	if err != nil {
		log.Fatal("Error while dialing "+ip_addr+" from SendDeviceEventQueueOverNetwork() : ", err)
	}
	_, err = conn.Write(write_buff)

	if err != nil {
		log.Fatal("Error while writing to "+ip_addr+" from SendDeviceEventQueueOverNetwork() : ", err)
	}

	conn.Close()

	acces.CloseConnection()
}

func SendDeviceEventQueueOverNetwork(connected_devices globals.GenArray[string], secure_id string, event_queue globals.GenArray[globals.QEvent], ip_addr ...string) {

	// for all devices connected concerned by the sync task, send the data with the right event flag
	// all others are handled in retard database table from the filesystem in a function call right before

	for i := 0; i < connected_devices.Size(); i++ {
		device_id := connected_devices.Get(i)

		var acces bdd.AccesBdd
		acces.InitConnection()

		if len(ip_addr) == 0 {
			ip_addr = append(ip_addr, acces.GetDeviceIP(device_id))
		}

		//SendStartUpdateEvent(secure_id, ip_addr[0])

		for i := 0; i < event_queue.Size(); i++ {

			event := event_queue.Get(i)

			//skipempty events that occurs when troncating a file :/
			if event.Flag == "[UPDATE]" && len(event.Delta.Instructions) == 0 {
				continue
			}

			SetEventNetworkLockForDevice(device_id, true)

			acces.SecureId = secure_id

			// we let the possibility to specify the address in the function arguments
			// as in the case of a [LINK_DEVICE] request, we don't have the IP address registered in the db

			// /!\ the device_id we send is our own so the other end can identify ourselves

			d := net.Dialer{Timeout: 1 * time.Second}
			conn, err := d.Dial("tcp", ip_addr[0]+":8274")

			if err != nil {
				log.Println("Error while dialing "+ip_addr[0]+" from SendDeviceEventQueueOverNetwork() : ", err)

				if IsEventFilesystemRelated(event.Flag) {
					// an error occured, adding this event to retard table
					log.Println("Adding event to retard table for this device")

					acces.SetDeviceConnectionState(device_id, false)
					MODTYPES := map[string]string{
						"[CREATE]": "c",
						"[REMOVE]": "d",
						"[UPDATE]": "p",
						"[MOVE]":   "m",
					}
					acces.RefreshCorrespondingRetardRow(event.FilePath, MODTYPES[event.Flag])

					// don't forget to release lock !!!
					SetEventNetworkLockForDevice(device_id, false)
				}

				return
			}

			formatted_event_file, err := globals.SerializeQeventToFile(event)
			if err != nil {
				log.Fatal("Error while serializing qevent to temporary file : ", err)
			}

			write_buff := []byte(acces.GetMyDeviceId() + ";" + secure_id)

			_, err = conn.Write(write_buff)
			if err != nil {
				log.Fatal("Error while sending request headers :", err)
			}

			n := 0
			size, err := formatted_event_file.Seek(0, io.SeekEnd)
			if err != nil {
				log.Fatal("Error while checking qevent temporary file size : ", err)
			}

			formatted_event_file.Seek(0, io.SeekStart)

			write_buff = make([]byte, delta_binaire.CalculateBufferSize(size))
			for n != -1 {
				n, err = formatted_event_file.Read(write_buff)
				if err != nil && err != io.EOF {
					log.Fatal("Error while reading qevent temporary file : ", err)
				}

				if err == io.EOF {
					err = nil
					break
				}

				_, err = conn.Write(write_buff)
				if err != nil {
					log.Fatal("Error while sending request body :", err)
				}
			}

			if err != nil && !IsEventFilesystemRelated(event.Flag) {
				log.Println("Error while writing to "+ip_addr[0]+" from SendDeviceEventQueueOverNetwork() : ", err)

				acces.SetDeviceConnectionState(device_id, false)
				MODTYPES := map[string]string{
					"[CREATE]": "c",
					"[REMOVE]": "d",
					"[UPDATE]": "p",
					"[MOVE]":   "m",
				}
				acces.RefreshCorrespondingRetardRow(event.FilePath, MODTYPES[event.Flag])

				// don't forget to release lock !!!
				SetEventNetworkLockForDevice(device_id, false)
			}

			conn.Close()

			log.Println("Event sent !")
			SetEventNetworkLockForDevice(device_id, false)

			// wait for the network lock to be released for this device
			for GetEventNetworkLockForDevice(device_id) {
				time.Sleep(1 * time.Second)
			}

			// finally, remove temporary file
			formatted_event_file.Close()
			err = os.Remove(formatted_event_file.Name())
			if err != nil {
				log.Fatal("Error while removing temporary event file : ", err)
			}

		}

		//SendStopUpdateEvent(secure_id, ip_addr[0])
		acces.CloseConnection()

	}

}

func SetEventNetworkLockForDevice(device_id string, value bool) {

	if value {
		file, err := os.Create(filepath.Join(globals.EcosysWriteableDirectory, device_id+".nlock"))

		if err != nil {
			log.Fatal("Error while creating a network lock file in SetEventNetworkLockForDevice() : ", err)
		}

		file.Close()
	} else {

		err := os.Remove(filepath.Join(globals.EcosysWriteableDirectory, device_id+".nlock"))
		log.Println("removing network lock after sending event")
		if err != nil && !os.IsNotExist(err) {
			log.Fatal("Error while removing a network lock file in SetEventNetworkLockForDevice() : ", err)
		}

	}

}

func GetEventNetworkLockForDevice(device_id string) bool {

	var acces bdd.AccesBdd
	return acces.IsFile(filepath.Join(globals.EcosysWriteableDirectory, device_id+".nlock"))

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
		log.Println("relative path : " + relative_path)
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
	user_response := backend_api.AskInput("[OTDL]", "Accept the largage aérien ? (coming from "+ip_addr+") \n File name : "+file_name+"\nFile would be saved to the folder : "+filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien\n\n"))
	if user_response == "true" {
		// make sure we have the right directory set-up
		ex := globals.ExistsInFilesystem(filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien"))

		if !ex {
			os.Mkdir(filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien"), 0775)
		}

		// build the path to the largage_aerien folder
		data.Delta.FilePath = filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien", file_name)

		// write the file. As this is probably a full file, the binary delta is just the file content
		data.Delta.PatchFile()

		// run the file if it is not an executable ( for security and conveniance reasons)

		if !globals.IsExecutable(data.Delta.FilePath) {
			err := open.Run(data.Delta.FilePath)
			if err != nil {
				open.Run(filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien"))
			}
		}

	}
}

func HandleMultipleLargageAerien(data globals.QEvent, ip_addr string) {
	// makes sure we are not given a path for some reasons
	file_name := filepath.Base(data.Delta.FilePath)

	backend_api.NotifyDesktop("Incoming Largage Aérien !! " + "(coming from " + ip_addr + ") \n File name : " + file_name)
	user_response := backend_api.AskInput("[MOTDL]", "Accept the MULTIPLE largage aérien ? (coming from "+ip_addr+") \n File name : "+file_name+"\nFile would be saved to the folder : "+filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien\n\n"))

	// veryfiy user response AND that we are not tricked to untar something random
	if user_response == "true" && file_name == "multilargage.tar" {
		// make sure we have the right directory set-up
		ex := globals.ExistsInFilesystem(filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien"))

		if !ex {
			os.Mkdir(filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien"), 0775)
		}

		// build the path to the largage_aerien folder
		data.Delta.FilePath = filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien", file_name)

		// write the file. As this is probably a full file, the binary delta is just the file content
		data.Delta.PatchFile()

		//unpacking tar file in a randomly generated folder with date
		now := time.Now()
		date_str := now.Format("2006-01-02-15h04min-05s")

		folder_name := file_name + "_" + date_str
		folder_path := filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien", folder_name)
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
	event.VersionToPatch = 0

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

func IsEventFilesystemRelated(flag string) bool {

	ret := flag != "[MOTDL]"
	ret = ret && flag != "[OTDL]"
	ret = ret && flag != "[LINK_DEVICE]"
	ret = ret && flag != "[UNLINK_DEVICE]"
	ret = ret && flag != "[MODIFICATION_DONE]"
	return ret
}
