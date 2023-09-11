package filesystem

import (
	"log"
	"os"
	"path/filepath"
	bdd "qsync/bdd"
	"qsync/delta_binaire"
	"qsync/networking"
	"time"

	"github.com/fsnotify/fsnotify"
)

func StartWatcher(rootPath string) {
	// Initialize the database connection
	bdd := bdd.AccesBdd{}
	bdd.InitConnection()
	bdd.GetSecureId(rootPath)

	defer bdd.CloseConnection()

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
		log.Fatal(err)
	}

	// Process filesystem events
	for {
		select {
		case event := <-watcher.Events:
			if !bdd.IsThisFileSystemBeingPatched() { // Check if the filesystem is not locked
				log.Println("NEW FILESYSTEM EVENT : ", event)
				switch {
				case event.Op&fsnotify.Create == fsnotify.Create:
					handleCreateEvent(&bdd, event.Name, watcher)
				case event.Op&fsnotify.Write == fsnotify.Write:
					handleWriteEvent(&bdd, event.Name)
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					handleRemoveEvent(&bdd, event.Name)
				case event.Op&fsnotify.Rename == fsnotify.Rename:
					handleRenameEvent(&bdd, event.Name)
				}
			}
		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}

func handleCreateEvent(bdd *bdd.AccesBdd, path string, watcher *fsnotify.Watcher) {
	if bdd.IsFile(path) {
		bdd.CreateFile(path)

		// creates a delta with full file content
		delta := delta_binaire.BuilDelta(path, bdd.GetFileSizeFromBdd(path), []byte(""))
		// add this delta to retard table
		bdd.UpdateFile(path, delta)

		var event networking.QEvent
		event.Flag = "CREATE"
		event.SecureId = bdd.SecureId
		event.FileType = "file"
		event.FilePath = path
		event.Delta = delta

		queue := make([]networking.QEvent, 1)
		queue = append(queue, event)

		networking.SendDeviceEventQueueOverNetwork(bdd.GetOnlineDevices(), bdd.SecureId, queue)

	} else {
		// add a watcher into this new folder
		watcher.Add(path)
		// notify changes as usual
		bdd.CreateFolder(path)
		bdd.AddFolderToRetard(path)

		var event networking.QEvent
		event.Flag = "CREATE"
		event.SecureId = bdd.SecureId
		event.FileType = "folder"
		event.FilePath = path

		queue := make([]networking.QEvent, 1)
		queue = append(queue, event)

		networking.SendDeviceEventQueueOverNetwork(bdd.GetOnlineDevices(), bdd.SecureId, queue)
	}
}

func handleWriteEvent(bdd *bdd.AccesBdd, path string) {
	if bdd.IsFile(path) {
		delta := delta_binaire.BuilDelta(path, bdd.GetFileSizeFromBdd(path), bdd.GetFileContent(path))
		bdd.UpdateFile(path, delta)

		var event networking.QEvent
		event.Flag = "UPDATE"
		event.SecureId = bdd.SecureId
		event.FileType = "file"
		event.FilePath = path
		event.Delta = delta

		queue := make([]networking.QEvent, 1)
		queue = append(queue, event)

		networking.SendDeviceEventQueueOverNetwork(bdd.GetOnlineDevices(), bdd.SecureId, queue)
	}
}

func handleRemoveEvent(bdd *bdd.AccesBdd, path string) {
	var file_type string
	if bdd.IsFile(path) {
		bdd.RmFile(path)

		file_type = "file"

	} else {
		bdd.RmFolder(path)
		file_type = "folder"
	}

	var event networking.QEvent
	event.Flag = "REMOVE"
	event.SecureId = bdd.SecureId
	event.FileType = file_type
	event.FilePath = path

	queue := make([]networking.QEvent, 1)
	queue = append(queue, event)

	networking.SendDeviceEventQueueOverNetwork(bdd.GetOnlineDevices(), bdd.SecureId, queue)
}

func handleRenameEvent(bdd *bdd.AccesBdd, path string) {
	newPath := computeNewPath(bdd, path)

	if newPath != "" {

		// determining if we moved a file or a directory
		var file_type string

		if !bdd.IsFile(newPath) {
			file_type = "folder"
		} else {
			file_type = "file"
		}

		// moving it in database

		bdd.Move(path, newPath, file_type)

		// sending move event to connected devices

		var event networking.QEvent
		event.Flag = "MOVE"
		event.SecureId = bdd.SecureId
		event.FileType = file_type
		event.FilePath = path
		event.NewFilePath = newPath

		queue := make([]networking.QEvent, 1)
		queue = append(queue, event)

		networking.SendDeviceEventQueueOverNetwork(bdd.GetOnlineDevices(), bdd.SecureId, queue)
	}
}

func computeNewPath(bdd *bdd.AccesBdd, path string) string {
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
		if bdd.WasFile(path) {
			bdd.RmFile(path)

			file_type = "file"

		} else {
			bdd.RmFolder(path)
			file_type = "folder"
		}

		var event networking.QEvent
		event.Flag = "REMOVE"
		event.SecureId = bdd.SecureId
		event.FileType = file_type
		event.FilePath = path

		queue := make([]networking.QEvent, 1)
		queue = append(queue, event)

		networking.SendDeviceEventQueueOverNetwork(bdd.GetOnlineDevices(), bdd.SecureId, queue)

	}

	// if lastest creation time is from before 5 seconds ago, we are facing a REMOVE event
	sec := 5
	fiveSecAgo := time.Now().Add(time.Duration(-sec) * time.Second)
	if latestCreationTime.Before(fiveSecAgo) {

		var file_type string
		if bdd.WasFile(path) {
			bdd.RmFile(path)

			file_type = "file"

		} else {
			bdd.RmFolder(path)
			file_type = "folder"
		}

		var event networking.QEvent
		event.Flag = "REMOVE"
		event.SecureId = bdd.SecureId
		event.FileType = file_type
		event.FilePath = path

		queue := make([]networking.QEvent, 1)
		queue = append(queue, event)

		networking.SendDeviceEventQueueOverNetwork(bdd.GetOnlineDevices(), bdd.SecureId, queue)

		return ""
	}

	// Construct the new path using the latest file name
	newPath := filepath.Join(dir, latestFile)

	return newPath
}
