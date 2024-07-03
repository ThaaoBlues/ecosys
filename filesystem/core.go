/*
 * @file            filesystem/core.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2023-09-11 14:08:11
 * @lastModified    2024-07-03 22:52:26
 * Copyright ©Théo Mougnibas All rights reserved
 */

package filesystem

import (
	"log"
	"os"
	"path/filepath"
	"qsync/bdd"
	"qsync/delta_binaire"
	"qsync/globals"
	"qsync/networking"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func StartWatcher(rootPath string) {
	// Initialize the database connection
	acces := bdd.AccesBdd{}
	acces.InitConnection()
	acces.GetSecureIdFromRootPath(rootPath)

	defer acces.CloseConnection()

	// Start the recursive filesystem watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Set the root path for the watcher
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("Error accessing path:", path, err)
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		log.Fatal("Error while walking path  (rootPath="+rootPath+") :", err)
	}

	// Process filesystem events
	for {
		select {
		case event := <-watcher.Events:
			log.Println("NEW FILESYSTEM EVENT (rootPath="+rootPath+" ) : ", event)
			// get only the relative path
			relative_path := strings.Replace(event.Name, rootPath, "", 1)

			// make sure the relative path does not start with a "/"
			relative_path = strings.TrimPrefix(relative_path, "/")

			switch {
			case event.Has(fsnotify.Create):
				// in the case of a directory being sent to us by another end, we still
				// have to partly process the event to add it to the watcher :)
				handleCreateEvent(&acces, event.Name, relative_path, watcher)
			case event.Has(fsnotify.Write):
				if !acces.IsThisFileSystemBeingPatched() {
					log.Println("Handling event for file : " + relative_path)
					handleWriteEvent(&acces, event.Name, relative_path)
				}
			case event.Has(fsnotify.Remove):
				// backup mode allow to store files on another machine while
				// still freeing up space on the device if we want
				if !acces.IsThisFileSystemBeingPatched() {
					handleRemoveEvent(&acces, event.Name, relative_path)
				}
			case event.Has(fsnotify.Rename):
				if !acces.IsThisFileSystemBeingPatched() {
					handleRenameEvent(&acces, event.Name, relative_path)
				}
			default:
				log.Println("Unhandled event (maybe in later versions ) : ", event)

			}

		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}

func handleCreateEvent(acces *bdd.AccesBdd, absolute_path string, relative_path string, watcher *fsnotify.Watcher) {

	var queue globals.GenArray[globals.QEvent]

	if acces.IsFile(absolute_path) && !acces.IsThisFileSystemBeingPatched() {

		acces.CreateFile(relative_path, absolute_path, "[ADD_TO_RETARD]")

		// creates a delta with full file content
		delta := delta_binaire.BuilDelta(relative_path, absolute_path, 0, []byte(""))

		var event globals.QEvent
		event.Flag = "[CREATE]"
		event.SecureId = acces.SecureId
		event.FileType = "file"
		event.FilePath = relative_path
		event.Delta = delta

		queue.Add(event)

		networking.SendDeviceEventQueueOverNetwork(acces.GetSyncOnlineDevices(), acces.SecureId, queue)

	} else {
		// add a watcher into this new folder
		log.Println("Adding " + absolute_path + " to the directories to watch.")
		watcher.Add(absolute_path)

		if !acces.IsThisFileSystemBeingPatched() {
			// notify changes as usual
			acces.CreateFolder(relative_path)
			acces.AddFolderToRetard(relative_path)

			var event globals.QEvent
			event.Flag = "[CREATE]"
			event.SecureId = acces.SecureId
			event.FileType = "folder"
			event.FilePath = relative_path

			queue.Add(event)

			networking.SendDeviceEventQueueOverNetwork(acces.GetSyncOnlineDevices(), acces.SecureId, queue)
		}

	}
}

func handleWriteEvent(acces *bdd.AccesBdd, absolute_path string, relative_path string) {

	var queue globals.GenArray[globals.QEvent]
	if acces.IsFile(absolute_path) {
		delta := delta_binaire.BuilDelta(relative_path, absolute_path, acces.GetFileSizeFromBdd(relative_path), acces.GetFileContent(relative_path))
		acces.UpdateFile(relative_path, delta)

		log.Println("BUILT FILE DELTA : ", delta)

		var event globals.QEvent
		event.Flag = "[UPDATE]"
		event.SecureId = acces.SecureId
		event.FileType = "file"
		event.FilePath = relative_path
		event.Delta = delta

		queue.Add(event)
		dev := acces.GetSyncOnlineDevices()
		log.Println("Sending event to " + strconv.FormatInt(int64(dev.Size()), 10) + " connected devices")
		networking.SendDeviceEventQueueOverNetwork(acces.GetSyncOnlineDevices(), acces.SecureId, queue)
		log.Println("Event sent.")
	}
}

func handleRemoveEvent(acces *bdd.AccesBdd, absolute_path string, relative_path string) {
	var file_type string
	var queue globals.GenArray[globals.QEvent]

	if acces.WasFile(relative_path) {
		acces.RmFile(relative_path)
		file_type = "file"

	} else {
		acces.RmFolder(relative_path)
		file_type = "folder"
	}

	if !acces.IsSyncInBackupMode() {
		var event globals.QEvent
		event.Flag = "[REMOVE]"
		event.SecureId = acces.SecureId
		event.FileType = file_type
		event.FilePath = relative_path

		queue.Add(event)

		networking.SendDeviceEventQueueOverNetwork(acces.GetSyncOnlineDevices(), acces.SecureId, queue)
	}

}

func handleRenameEvent(acces *bdd.AccesBdd, absolute_path string, relative_path string) {
	new_absolute_path := computeNewPath(acces, absolute_path)

	new_relative_path := strings.Replace(new_absolute_path, acces.GetRootSyncPath(), "", 1)
	var queue globals.GenArray[globals.QEvent]

	if new_relative_path != "" {

		// determining if we moved a file or a directory
		var file_type string

		if !acces.IsFile(new_absolute_path) {
			file_type = "folder"
		} else {
			file_type = "file"
		}

		// moving it in database

		acces.Move(relative_path, new_relative_path, file_type)

		// sending move event to connected devices

		var event globals.QEvent
		event.Flag = "[MOVE]"
		event.SecureId = acces.SecureId
		event.FileType = file_type
		event.FilePath = relative_path
		event.NewFilePath = new_relative_path

		queue.Add(event)
		networking.SendDeviceEventQueueOverNetwork(acces.GetSyncOnlineDevices(), acces.SecureId, queue)
	}
}

func computeNewPath(acces *bdd.AccesBdd, path string) string {
	dir := filepath.Dir(path)

	// List all files in the directory
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal("Error listing files:", err)
		return ""
	}

	// Find the file with the latest creation date
	var latestCreationTime time.Time
	var latestFile string

	for _, file := range files {
		finfo, err := file.Info()

		if err != nil {
			log.Fatal("Error while fetching file informations : ", err)
		}

		if !file.IsDir() && finfo.ModTime().After(latestCreationTime) {
			latestCreationTime = finfo.ModTime()
			latestFile = file.Name()
		}
	}

	// Ensure a file with the latest creation date was found,
	// if not, it means it is a REMOVE event on the last file of this folder

	// still check if it is a folder in case -\_(-_-)_/-
	if latestFile == "" {
		var file_type string
		if acces.WasFile(path) {
			acces.RmFile(path)

			file_type = "file"

		} else {
			acces.RmFolder(path)
			file_type = "folder"
		}

		var event globals.QEvent
		event.Flag = "[REMOVE]"
		event.SecureId = acces.SecureId
		event.FileType = file_type
		event.FilePath = path

		var queue globals.GenArray[globals.QEvent]
		queue.Add(event)

		networking.SendDeviceEventQueueOverNetwork(acces.GetSyncOnlineDevices(), acces.SecureId, queue)

	}

	// if lastest creation time is from before 5 seconds ago, we are facing a REMOVE event
	sec := 5
	fiveSecAgo := time.Now().Add(time.Duration(-sec) * time.Second)
	if latestCreationTime.Before(fiveSecAgo) {

		var file_type string
		if acces.WasFile(path) {
			acces.RmFile(path)

			file_type = "file"

		} else {
			acces.RmFolder(path)
			file_type = "folder"
		}

		var event globals.QEvent
		event.Flag = "[REMOVE]"
		event.SecureId = acces.SecureId
		event.FileType = file_type
		event.FilePath = path

		var queue globals.GenArray[globals.QEvent]
		queue.Add(event)

		networking.SendDeviceEventQueueOverNetwork(acces.GetSyncOnlineDevices(), acces.SecureId, queue)

		return ""
	}

	// Construct the new path using the latest file name
	newPath := filepath.Join(dir, latestFile)

	return newPath
}
