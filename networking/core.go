package networking

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"qsync/bdd"
	"qsync/delta_binaire"
	"strings"
	"time"
)

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

func InitNetWorkProc() {

}

func ConnectToDevice(conn net.Conn) {

	var acces bdd.AccesBdd

	acces.InitConnection()

	// get the device id and secure sync id from header

	header_buff := make([]byte, 82)
	_, err := conn.Read(header_buff)

	if err != nil {
		log.Println("Error in ConnectToDevice() while reading header")
	}

	log.Println(header_buff)

	var device_id string = strings.Split(string(header_buff), ";")[0]
	var secure_id string = strings.Split(string(header_buff), ";")[1]

	// makes sure it is marked as connected
	if !acces.GetDeviceConnectionState(device_id) {

		acces.SetDeviceConnectionState(device_id, true)

	}

	// read the body of the request and store it in a buffer
	var body_buff []byte
	for {
		buffer := make([]byte, 1024) // You can adjust the buffer size as needed
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Println("Error in ConnectToDevice() while reading body:", err)
			}
			break
		}
		body_buff = append(body_buff, buffer[:n]...)
	}

	switch string(body_buff) {

	case "[MODIFICATION_DONE]":
		SetEventNetworkLockForDevice(device_id, false)
	case "[SETUP_DL]":

		// init an event queue with all elements from the root sync directory
		BuildSetupQueue(secure_id, device_id)

	case "[LINK_DEVICE]":

		acces.LinkDevice(device_id)

	case "[UNLINK_DEVICE]":
		acces.UnlinkDevice(device_id)

	default:
		HandleEvent(secure_id, device_id, body_buff)
		// send back a modification confirmation, so the other end can remove this machine device_id
		// from concerned sync task retard entries
		buff := []byte(acces.GetMyDeviceId() + ";" + acces.SecureId + ";" + "[MODIFICATION_DONE]")
		conn.Write(buff)
	}

}

func HandleEvent(secure_id string, device_id string, buffer []byte) {

	var event QEvent

	err := json.Unmarshal(buffer, &event)
	if err != nil {
		log.Println("Error while decoding json data from request buffer in HandleEvent()", err)
	}

	// first, we lock the filesystem watcher so it don't notify the changes we are doing
	// as it would do a ping-pong effect

	var acces bdd.AccesBdd
	acces.SecureId = secure_id

	acces.InitConnection()

	acces.SetFileSystemPatchLockState(device_id, true)

	switch event.Flag {
	case "MOVE":
		acces.Move(event.FilePath, event.NewFilePath, event.FileType)
		MoveInFilesystem(event.FilePath, event.NewFilePath)
	case "REMOVE":
		if event.FileType == "file" {
			acces.RmFile(event.FilePath)

		} else {
			acces.RmFolder(event.FilePath)
		}

		RemoveFromFilesystem(event.FilePath)

	case "CREATE":
		if event.FileType == "file" {
			acces.CreateFile(event.FilePath)
			event.Delta.PatchFile()
		} else {
			acces.CreateFolder(event.FilePath)
			os.Mkdir(event.FilePath, 0755)
		}

	case "UPDATE":
		acces.IncrementFileVersion(event.FilePath)
		event.Delta.PatchFile()
	default:
		log.Fatal("Qsync network loop received an unknown event type : ", event)
	}

	acces.SetFileSystemPatchLockState(device_id, false)

}

func SendDeviceEventQueueOverNetwork(connected_devices []string, secure_id string, event_queue []QEvent) {

	// for all devices connected concerned by the sync task, send the data with the right event flag
	// all others are handled in retard database table from the filesystem in a function call right before

	for _, device_id := range connected_devices {
		for _, event := range event_queue {

			SetEventNetworkLockForDevice(device_id, true)

			event_json, err := json.Marshal(event)
			if err != nil {
				log.Fatal("An error occured in SendDeviceEventQueueOverNetwork() while creating a json object from the event struct : ", err)
			}

			var acces bdd.AccesBdd

			ip_addr := acces.GetDeviceIP(device_id)

			write_buff := []byte(device_id + ";" + secure_id + string(event_json))

			conn, err := net.Dial("tcp", ip_addr+":8274")

			if err != nil {
				log.Fatal("Error while dialing "+ip_addr+" from SendDeviceEventQueueOverNetwork() : ", err)
			}
			_, err = conn.Write(write_buff)

			if err != nil {
				log.Fatal("Error while writing to "+ip_addr+" from SendDeviceEventQueueOverNetwork() : ", err)
			}

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

	rootPath := bdd.GetRootSyncPath()

	var queue []QEvent

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("Error accessing path:", path, err)
			return err
		}

		if info.IsDir() {
			// creates a delta with full file content

			var event QEvent
			event.Flag = "CREATE"
			event.SecureId = secure_id
			event.FileType = "folder"
			event.FilePath = path

			queue = append(queue, event)

		} else {
			// creates a delta with full file content
			delta := delta_binaire.BuilDelta(path, bdd.GetFileSizeFromBdd(path), []byte(""))

			var event QEvent
			event.Flag = "CREATE"
			event.SecureId = secure_id
			event.FileType = "file"
			event.FilePath = path
			event.Delta = delta

			queue = append(queue, event)
		}

		return nil
	})

	var devices []string
	devices = append(devices, device_id)
	SendDeviceEventQueueOverNetwork(devices, bdd.SecureId, queue)

	if err != nil {
		log.Fatal(err)
	}

}
