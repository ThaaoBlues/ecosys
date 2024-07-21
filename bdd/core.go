/*
 * @file            bdd/core.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2023-09-11 14:08:11
 * @lastModified    2024-07-21 20:25:42
 * Copyright ©Théo Mougnibas All rights reserved
 */

package bdd

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"qsync/delta_binaire"
	"qsync/globals"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// AccesBdd represents access to the database.
type AccesBdd struct {
	BddName    string
	db_handler *sql.DB
	SecureId   string
}

// SyncInfos represents synchronization information with a path and its secure ID.
type SyncInfos struct {
	Path       string
	SecureId   string
	IsApp      bool
	Name       string
	BackupMode bool
}

type LinkDevice struct {
	SecureId    string
	IsConnected bool
}

// InitConnection initializes the database connection and creates necessary tables if they don't exist.
// This function is used everytime we create an AccesBdd object
func (acces *AccesBdd) InitConnection() {
	var err error
	acces.db_handler, err = sql.Open("sqlite3", filepath.Join(globals.QSyncWriteableDirectory, "qsync.db"))

	if err != nil {
		log.Fatal("An error occured while connecting to the qsync database.", err)
	}

	_, err = acces.db_handler.Exec(`CREATE TABLE IF NOT EXISTS retard(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version_id INTEGER,
		path TEXT,
		mod_type TEXT,
		devices_to_patch TEXT DEFAULT "",
		type TEXT,
		secure_id TEXT
	)`)

	if err != nil {
		log.Fatal("Error while creating table retard : ", err)
	}

	_, err = acces.db_handler.Exec(`CREATE TABLE IF NOT EXISTS delta(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT,
		version_id INTEGER,
		delta TEXT,
		secure_id TEXT
	)`)
	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}

	_, err = acces.db_handler.Exec(`CREATE TABLE IF NOT EXISTS filesystem(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT,
		version_id INTEGER,
		type TEXT,
		size INTEGER,
		secure_id TEXT,
		content BLOB
	)`)
	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}

	_, err = acces.db_handler.Exec(`CREATE TABLE IF NOT EXISTS sync(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		secure_id TEXT,
		linked_devices_id TEXT DEFAULT "",
		root TEXT,
		backup_mode BOOLEAN DEFAULT 0,
		is_being_patch BOOLEAN DEFAULT 0,
		creation_date INTEGER DEFAULT (strftime('%s','now'))
	)`)
	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}

	_, err = acces.db_handler.Exec(`CREATE TABLE IF NOT EXISTS linked_devices(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id TEXT,
		is_connected BOOLEAN,
		ip_addr TEXT
	)`)

	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}
	_, err = acces.db_handler.Exec(`CREATE TABLE IF NOT EXISTS mesid(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id TEXT,
		accepte_largage_aerien BOOLEAN DEFAULT TRUE
	)`)
	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}

	_, err = acces.db_handler.Exec(`CREATE TABLE IF NOT EXISTS apps(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		path TEXT,
		version_id INTEGER,
		type TEXT,
		secure_id TEXT,
		uninstaller_path TEXT
	)`)

	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}

	_, err = acces.db_handler.Exec(`CREATE TABLE IF NOT EXISTS reseau(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id TEXT,
		hostname TEXT,
		ip_addr TEXT
	)`)

	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}

	if !acces.IsMyDeviceIdGenerated() {
		acces.GenerateMyDeviceId()
	}
}

// CloseConnection closes the database connection.
func (acces *AccesBdd) CloseConnection() {
	acces.db_handler.Close()
}

// CheckFileExists checks if a file with a given path exists in the database.
func (acces *AccesBdd) CheckFileExists(path string) bool {
	rows, err := acces.db_handler.Query("SELECT id FROM filesystem WHERE path=? AND secure_id=? VALUES=(?,?)", path, acces.SecureId)

	if err != nil {
		log.Fatal("Error while querying database from CheckFileExists() : ", err)
	}
	defer rows.Close()

	var i int = 0

	for rows.Next() && i == 0 {
		/*var id int

		if err := rows.Scan(&id); err != nil {
			log.Fatal("Error while querying database in CheckFileExists() : ", err)
		}*/
		i++

	}

	return i > 0

}

// WasFile checks if a file with a given path was present in the past.
// This is made by checking the database as the filesystem may have been alterated
func (acces *AccesBdd) WasFile(path string) bool {

	row := acces.db_handler.QueryRow("SELECT type FROM filesystem WHERE path=? AND secure_id=?", path, acces.SecureId)

	var fileType string
	err := row.Scan(&fileType)

	// when we delete a folder the event is called n times
	// with n being the number of sub elements of the given folder
	// but the first event has already erased the rows from database
	// so no rows error is thrown
	if err == sql.ErrNoRows {
		return false
	}

	if err != nil {
		log.Fatal("Error while querying database from WasFile() : ", err)
	}

	return fileType == "file"

}

// IsFile checks if a given path represents a file (not a directory).
// does not acces the db, it uses the client real filesystem
func (acces *AccesBdd) IsFile(path string) bool {

	handler, err := os.Open(path)

	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		log.Fatal("Error while checking if file is directory or regular file.", err)
	}

	stat, err := handler.Stat()

	if err != nil {
		log.Fatal("Error while checking if file is directory or regular file.", err)
	}

	return !stat.IsDir()
}

// GetSecureIdFromRootPath retrieves the secure ID associated with a given root path.
func (acces *AccesBdd) GetSecureIdFromRootPath(rootpath string) {
	row := acces.db_handler.QueryRow("SELECT secure_id FROM sync WHERE root=?", rootpath)

	err := row.Scan(&acces.SecureId)

	if err == sql.ErrNoRows {
		log.Fatal("The provided path does not match any existing sync rootpath")
	}

	if err != nil {
		log.Fatal("Error while querying database GetSecureIdFromRootPath() : ", err)
	}

}

func (acces *AccesBdd) GetSecureIdFromRootPathMatch(rootpath string) {
	row := acces.db_handler.QueryRow("SELECT secure_id FROM sync WHERE root LIKE ?", rootpath+"?")

	err := row.Scan(&acces.SecureId)

	if err == sql.ErrNoRows {
		log.Fatal("The provided path does not match any existing sync rootpath")
	}

	if err != nil {
		log.Fatal("Error while querying database GetSecureIdFromRootPath() : ", err)
	}

}

// CreateFile adds a file to the database.
func (acces *AccesBdd) CreateFile(relative_path string, absolute_path string, flag string) {

	file_handler, err := os.Open(absolute_path)

	if err != nil {
		log.Fatal("Error while opening the file from real filesystem to seek changes. : ", err)
	}

	defer file_handler.Close()

	stat, err := file_handler.Stat()

	if err != nil {
		log.Fatal("Error while parsing file stats")
	}

	file_content, err := io.ReadAll(file_handler)

	if err != nil {
		log.Fatal("Error while reading file content")
	}

	// needs a special bytes buffer as just a byte slice does not implements the required methods for gzip
	var bytes_buffer bytes.Buffer

	gzip_writer := gzip.NewWriter(&bytes_buffer)

	gzip_writer.Write(file_content)

	gzip_writer.Close()

	_, err = acces.db_handler.Exec("INSERT INTO filesystem (path, version_id, type, size, secure_id,content) VALUES (?, 0, 'file', ?, ?,?)", relative_path, stat.Size(), acces.SecureId, bytes_buffer.Bytes())

	if err != nil {
		log.Fatal("Error while inserting into database ", err)
	}

	// Now, add this file to retard and delta etc... all for the others linked devices
	// get only offline devices

	if flag == "[ADD_TO_RETARD]" {
		delta := delta_binaire.BuilDelta(relative_path, absolute_path, 0, []byte(""))
		offline_devices := acces.GetSyncOfflineDevices()
		if offline_devices.Size() > 0 {
			new_version_id := acces.GetFileLastVersionId(relative_path) + 1

			acces.IncrementFileVersion(relative_path)

			// convert detla to json
			json_data, err := json.Marshal(delta)

			if err != nil {
				log.Fatal("Error while creating json object from delta type : ", err)
			}

			// add line in delta table

			_, err = acces.db_handler.Exec("INSERT INTO delta (path,version_id,delta,secure_id) VALUES(?,?,?,?)", relative_path, new_version_id, json_data, acces.SecureId)

			if err != nil {
				log.Fatal("Error while storing binary delta in database : ", err)
			}

			// add a line in retard table with all devices linked and the version number

			MODTYPES := map[string]string{
				"creation": "c",
				"delete":   "d",
				"patch":    "p",
				"move":     "m",
			}

			var str_ids string = ""
			for i := 0; i < offline_devices.Size(); i++ {
				str_ids += offline_devices.Get(i) + ";"
			}
			// remove the last semicolon
			str_ids = str_ids[:len(str_ids)-1]

			_, err = acces.db_handler.Exec("INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,\"file\",?)", new_version_id, relative_path, MODTYPES["creation"], str_ids, acces.SecureId)

			if err != nil {
				log.Fatal("Error while inserting new retard : ", err)
			}
		}

	}

	// update the cached file content to build the next delta (needs absolute path to the file)
	acces.UpdateCachedFile(relative_path, absolute_path)
}

// GetFileLastVersionId retrieves the last version ID of a file.
func (acces *AccesBdd) GetFileLastVersionId(path string) int64 {
	row := acces.db_handler.QueryRow("SELECT version_id FROM filesystem WHERE path=? AND secure_id=?", path, acces.SecureId)

	var version_id int64
	err := row.Scan(&version_id)
	if err != nil {
		log.Fatal("Error while querying database GetFileLastVersionId() : ", err)
	}

	return version_id
}

func (acces *AccesBdd) GetSyncOfflineDevices() globals.GenArray[string] {
	linked_devices := acces.GetSyncLinkedDevices()

	/*var str_ids string = ""
	for i := 0; i < linked_devices.Size(); i++ {
		str_ids += linked_devices.Get(i) + ","
	}

	if(str_ids > 1)
	// remove the last colon
	str_ids = str_ids[:len(str_ids)-1]

	query := "SELECT device_id,is_connected FROM linked_devices WHERE device_id IN ('"
	query += str_ids
	query += "')"

	rows, err := acces.db_handler.Query(query)

	if err != nil {
		log.Fatal("Error while querying database from GetSyncOfflineDevices() : ", err)
	}
	defer rows.Close()

	var offline_devices globals.GenArray[string]

	for rows.Next() {
		var device LinkDevice
		rows.Scan(&device.SecureId, &device.IsConnected)

		if !device.IsConnected {
			offline_devices.Add(device.SecureId)
		}
	}*/

	var offline_devices globals.GenArray[string]

	for i := 0; i < linked_devices.Size(); i++ {
		if !acces.GetDeviceConnectionState(linked_devices.Get(i)) {
			log.Println(linked_devices.Get(i))
			offline_devices.Add(linked_devices.Get(i))
		}
	}

	return offline_devices
}

// UpdateFile updates a file in the database with a new version.
// For that, a binary delta object is used.
func (acces *AccesBdd) UpdateFile(path string, delta delta_binaire.Delta) {

	// get only offline devices

	offline_devices := acces.GetSyncOfflineDevices()
	if offline_devices.Size() > 0 {
		new_version_id := acces.GetFileLastVersionId(path) + 1

		acces.IncrementFileVersion(path)

		// convert detla to a serialized representation
		//and add line in delta table

		_, err := acces.db_handler.Exec("INSERT INTO delta (path,version_id,delta,secure_id) VALUES(?,?,?,?)", path, new_version_id, delta.Serialize(), acces.SecureId)

		if err != nil {
			log.Fatal("Error while storing binary delta in database : ", err)
		}

		// add a line in retard table with all devices linked and the version number

		MODTYPES := map[string]string{
			"creation": "c",
			"delete":   "d",
			"patch":    "p",
			"move":     "m",
		}

		var str_ids string = ""
		for i := 0; i < offline_devices.Size(); i++ {
			str_ids += offline_devices.Get(i) + ";"
		}
		// remove the last semicolon
		str_ids = str_ids[:len(str_ids)-1]

		_, err = acces.db_handler.Exec(
			"INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,\"file\",?)",
			new_version_id,
			path,
			MODTYPES["patch"],
			str_ids,
			acces.SecureId)

		if err != nil {
			log.Fatal("Error while inserting new retard : ", err)
		}
	}

	// update the cached file content to build the next delta (needs absolute path to the file)
	acces.UpdateCachedFile(path, filepath.Join(acces.GetRootSyncPath(), path))
}

// rebuild an entry in retard table to add a potential missing device

func (acces *AccesBdd) RefreshCorrespondingRetardRow(path string, modtype string) {
	version_id := acces.GetFileLastVersionId(path)

	// remove the outdated retard entry

	acces.db_handler.Exec("DELETE FROM retard WHERE path=? AND version_id=? AND secure_id=?", path, version_id, acces.SecureId)

	// get only offline devices

	offline_devices := acces.GetSyncOfflineDevices()
	if offline_devices.Size() > 0 {
		// add a line in retard table with all devices linked and the version number

		var str_ids string = ""
		for i := 0; i < offline_devices.Size(); i++ {
			str_ids += offline_devices.Get(i) + ";"
		}
		// remove the last semicolon
		str_ids = str_ids[:len(str_ids)-1]

		_, err := acces.db_handler.Exec(
			"INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,\"file\",?)",
			version_id,
			path,
			modtype,
			str_ids,
			acces.SecureId)

		if err != nil {
			log.Fatal("Error while inserting new retard : ", err)
		}
	}

	// update the cached file content to build the next delta (needs absolute path to the file)
	acces.UpdateCachedFile(path, filepath.Join(acces.GetRootSyncPath(), path))
}

// GetFileContent retrieves the content of a file from the database.
// returned as byte array
func (acces *AccesBdd) GetFileContent(path string) []byte {
	row := acces.db_handler.QueryRow("SELECT content FROM filesystem WHERE path=? AND secure_id=?", path, acces.SecureId)

	var compressed_content []byte
	err := row.Scan(&compressed_content)
	if err != nil {
		log.Fatal("Error while querying database in GetFileContent() : ", err)
	}

	decompression_buffer := bytes.NewReader(compressed_content)

	reader, err := gzip.NewReader(decompression_buffer)

	if err != nil {
		log.Fatal("Error while creating new gzip reader", err)
	}

	decompressed_content, err := io.ReadAll(reader)

	if err != nil {
		log.Fatal("Error while reading decompression buffer", err)
	}

	return decompressed_content
}

// RmFile deletes a file from the database and adds it in delete mode to the retard table.
func (acces *AccesBdd) RmFile(path string) {
	_, err := acces.db_handler.Exec("DELETE FROM filesystem WHERE path=? AND secure_id=?", path, acces.SecureId)

	if err != nil {
		log.Fatal("Error while deleting from database ", err)
	}

	// now, purge all data involving this file from retard table
	_, err = acces.db_handler.Exec("DELETE FROM retard WHERE path=? AND secure_id=?", path, acces.SecureId)

	if err != nil {
		log.Fatal("Error while deleting from database ", err)
	}

	// and finally, add it in delete mode to the retard table
	MODTYPES := map[string]string{
		"creation": "c",
		"delete":   "d",
		"patch":    "p",
		"move":     "m",
	}
	linked_devices := acces.GetSyncLinkedDevices()

	if linked_devices.Size() > 0 {
		var str_ids string = ""
		for i := 0; i < linked_devices.Size(); i++ {
			str_ids += linked_devices.Get(i) + ";"
		}
		// remove the last semicolon
		str_ids = str_ids[:len(str_ids)-1]

		_, err = acces.db_handler.Exec("INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,\"file\",?)", 0, path, MODTYPES["delete"], str_ids, acces.SecureId)

		if err != nil {
			log.Fatal("Error while inserting new retard : ", err)
		}
	}

}

// CreateFolder adds a folder to the database.
func (acces *AccesBdd) CreateFolder(path string) {
	_, err := acces.db_handler.Exec("INSERT INTO filesystem (path, version_id, type, size, secure_id) VALUES (?, 0, 'folder', 0, ?)", path, acces.SecureId)

	if err != nil {
		log.Fatal("Error while inserting into database ", err)
	}
}

// RmFolder deletes a folder from the database and adds it in delete mode to the retard table.
func (acces *AccesBdd) RmFolder(path string) {

	_, err := acces.db_handler.Exec("DELETE FROM filesystem WHERE path LIKE ? AND secure_id=?", path+"%", acces.SecureId)

	if err != nil {
		log.Fatal("Error while deleting from database ", err)
	}

	// now, purge all data involving this folder from retard table
	_, err = acces.db_handler.Exec("DELETE FROM retard WHERE path LIKE ? AND secure_id=?", path+"%", acces.SecureId)

	if err != nil {
		log.Fatal("Error while deleting from database ", err)
	}

	// purge all data from delta table involving this folder
	_, err = acces.db_handler.Exec("DELETE FROM delta WHERE path LIKE ? AND secure_id=?", path+"%", acces.SecureId)

	if err != nil {
		log.Fatal("Error while deleting from database ", err)
	}

	// and finally, add it in delete mode to the retard table
	MODTYPES := map[string]string{
		"creation": "c",
		"delete":   "d",
		"patch":    "p",
		"move":     "m",
	}
	linked_devices := acces.GetSyncLinkedDevices()

	if linked_devices.Size() > 0 {
		var str_ids string = ""
		for i := 0; i < linked_devices.Size(); i++ {
			str_ids += linked_devices.Get(i) + ";"
		}
		// remove the last semicolon
		str_ids = str_ids[:len(str_ids)-1]

		_, err = acces.db_handler.Exec("INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,\"folder\",?)", 0, path, MODTYPES["delete"], str_ids, acces.SecureId)

		if err != nil {
			log.Fatal("Error while inserting new retard : ", err)
		}
	}

}

// Move updates the path of a file or folder in the database and adds a move event to the retard table.
func (acces *AccesBdd) Move(path string, new_path string, file_type string) {
	_, err := acces.db_handler.Exec("UPDATE filesystem SET path=? WHERE path=? AND secure_id=?", new_path, path, acces.SecureId)
	if err != nil {
		log.Fatal("Error while updating database in move()", err)
	}

	// add the move event to retard file
	MODTYPES := map[string]string{
		"creation": "c",
		"delete":   "d",
		"patch":    "p",
		"move":     "m",
	}

	linked_devices := acces.GetSyncLinkedDevices()

	if linked_devices.Size() > 0 {

		var str_ids string = ""
		for i := 0; i < linked_devices.Size(); i++ {
			str_ids += linked_devices.Get(i) + ";"
		}
		// remove the last semicolon
		str_ids = str_ids[:len(str_ids)-1]

		_, err = acces.db_handler.Exec("INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,?,?)", 0, path, MODTYPES["move"], str_ids, file_type, acces.SecureId)

		if err != nil {
			log.Fatal("Error while inserting new retard : ", err)
		}
	}

}

// CreateSync initializes a new synchronization entry in the database.
func (acces *AccesBdd) CreateSync(rootPath string) {

	// generating secure_id

	alphabet := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	s := make([]rune, 41)

	for i := range s {
		s[i] = alphabet[rand.Intn(len(alphabet))]
	}

	secure_id := string(s)

	acces.SecureId = secure_id

	// adding sync to database
	_, err := acces.db_handler.Exec("INSERT INTO sync (secure_id, root) VALUES(?,?)", secure_id, rootPath)

	if err != nil {
		log.Fatal("Error while inserting into database ", err)
	}

	// add files to the filesystem, necessitate that all files are on the newer version so it don't erase the ones fro

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("Error accessing path:", path, err)
			return err
		}
		log.Println("Registering : ", path)
		relative_path := strings.Replace(path, rootPath, "", 1)
		if info.IsDir() {
			acces.CreateFolder(relative_path)
		} else {
			acces.CreateFile(relative_path, path, "[ADD_TO_RETARD]")
		}

		return nil
	})

	log.Println("finished to map the folder.")

	if err != nil {
		log.Fatal("Error while registering files and folders for the first time.")
	}

}

// CreateSyncFromOtherEnd creates a synchronization entry in the database with the givens info.
// Used to connect from an existing task from another device
// Filesystem is not mapped by the function as a remote setup procedure
// is made around this call
func (acces *AccesBdd) CreateSyncFromOtherEnd(rootPath string, secure_id string) {
	acces.SecureId = secure_id

	// adding sync to database
	_, err := acces.db_handler.Exec("INSERT INTO sync (secure_id, root) VALUES(?,?)", secure_id, rootPath)

	if err != nil {
		log.Fatal("Error while inserting into database ", err)
	}
}

// RmSync removes a synchronization entry from the database.

func (acces *AccesBdd) RmSync() {
	_, err := acces.db_handler.Exec("DELETE FROM sync WHERE secure_id=?", acces.SecureId)

	if err != nil {
		log.Fatal("Error while deleting from database ", err)
	}
}

// LinkDevice links a device to the synchronization entry.
func (acces *AccesBdd) LinkDevice(device_id string, ip_addr string) {

	// link this device to an existing task
	_, err := acces.db_handler.Exec("UPDATE sync SET linked_devices_id=IFNULL(linked_devices_id, '') || ? WHERE secure_id=?", device_id+";", acces.SecureId)

	if err != nil {
		log.Fatal("Error while updating database in LinkDevice() : ", err)
	}

	// if the device is not registered as a target (from previous tasks), register it
	if !acces.IsDeviceLinked(device_id) {
		_, err = acces.db_handler.Exec("INSERT INTO linked_devices (device_id,is_connected,ip_addr) VALUES(?,TRUE,?)", device_id, ip_addr)

		if err != nil {
			log.Fatal("Error while updating database in LinkDevice() : ", err)
		}
	}

}

// UnlinkDevice unlinks a device from a synchronization entry.
func (acces *AccesBdd) UnlinkDevice(device_id string) {

	// remove id from the list
	_, err := acces.db_handler.Exec("DELETE FROM linked_devices WHERE device_id=?", device_id)

	if err != nil {
		log.Fatal("Error while updating database in UnlinkDevice() : ", err)
	}

	// remove id from the list in sync table
	_, err = acces.db_handler.Exec("UPDATE sync SET linked_devices_id=REPLACE(linked_devices_id,?,'')", device_id+";", acces.SecureId)

	if err != nil {
		log.Fatal("Error while updating database in UnlinkDevice() : ", err)
	}
}

// GetRootSyncPath retrieves the root path associated with the synchronization entry.
func (acces *AccesBdd) GetRootSyncPath() string {
	var rootPath string
	row := acces.db_handler.QueryRow("SELECT root FROM sync WHERE secure_id=?", acces.SecureId)

	err := row.Scan(&rootPath)
	if err != nil {
		log.Fatal("Error while querying database in GetRootSyncPath() : ", err)
	}

	return rootPath
}

// SetDeviceConnectionState updates the connection state of a linked device.
func (acces *AccesBdd) SetDeviceConnectionState(device_id string, value bool, ip_addr ...string) {

	if len(ip_addr) == 0 {
		_, err := acces.db_handler.Exec("UPDATE linked_devices SET is_connected=? WHERE device_id=?", value, device_id)

		if err != nil {
			log.Fatal("Error while updating database in SetDeviceConnectionState() : ", err)
		}
	} else {
		_, err := acces.db_handler.Exec("UPDATE linked_devices SET is_connected=?,ip_addr=? WHERE device_id=?", value, ip_addr[0], device_id)

		if err != nil {
			log.Fatal("Error while updating database in SetDeviceConnectionState() : ", err)
		}
	}

}

func (acces *AccesBdd) GetDeviceConnectionState(device_id string) bool {
	var connection_state bool
	row := acces.db_handler.QueryRow("SELECT is_connected FROM linked_devices WHERE device_id=?", device_id)

	err := row.Scan(&connection_state)
	if err != nil {
		log.Fatal("Error while querying database in GetDeviceConnectionState() : ", err)
	}

	return connection_state
}

func (acces *AccesBdd) ClearAllFileSystemLockInDb() {
	_, err := acces.db_handler.Exec("UPDATE sync SET is_being_patch=0")

	if err != nil {
		log.Fatal("Error while updating database in LinkDevice() : ", err)
	}
}

// search in the list of secure_id if the given one is receiving updates from another device
func (acces *AccesBdd) IsThisFileSystemBeingPatched() bool {

	var count int = 0
	err := acces.db_handler.QueryRow(
		"SELECT COUNT(*) FROM sync WHERE secure_id=? AND is_being_patch=?",
		acces.SecureId,
		true,
	).Scan(&count)

	if err != nil {
		log.Fatal(err)
	}

	return count > 0

}

func (acces *AccesBdd) SetFileSystemPatchLockState(value bool) {

	// lock the filesystem (simply add the secure_id of the sync task to the list)

	_, err := acces.db_handler.Exec(
		"UPDATE sync SET is_being_patch=? WHERE secure_id=?",
		value,
		acces.SecureId,
	)

	if err != nil {
		log.Fatal("Error while updating database in LinkDevice() : ", err)
	}

}

func (acces *AccesBdd) GetFileSizeFromBdd(path string) int64 {
	var size int64
	row := acces.db_handler.QueryRow("SELECT size FROM filesystem WHERE path=? AND secure_id=?", path, acces.SecureId)

	err := row.Scan(&size)
	if err != nil {
		log.Fatal("Error while querying database GetFileSizeFromBdd()", err)
	}

	return size
}

func (acces *AccesBdd) GetSyncLinkedDevices() globals.GenArray[string] {
	var devices_str string
	var devices_list globals.GenArray[string]

	row := acces.db_handler.QueryRow("SELECT linked_devices_id FROM sync WHERE secure_id=?", acces.SecureId)

	err := row.Scan(&devices_str)
	// as the secure_id could change while the files watcher is running
	// in the case of an app link
	// no rows could match the select at the create event of the setup for example
	// and the create event is still trigered as the filesystem lock does not
	// match the old secure_id
	if err != nil && err != sql.ErrNoRows {
		log.Fatal("Error while querying database in GetSyncLinkedDevices() :", err)
	}

	if err == nil {
		for _, val := range strings.Split(devices_str, ";") {
			devices_list.Add(val)
		}
	} else {
		devices_list.Add(";")
	}
	// remove the last slot (empty space) in the array
	// caused by the last semicolon of the string representation of the list
	devices_list.PopLast()

	return devices_list
}

func (acces *AccesBdd) GetFileDelta(version int64, path string) delta_binaire.Delta {

	var delta delta_binaire.Delta
	var json_data string

	row := acces.db_handler.QueryRow("SELECT delta FROM delta WHERE path=? AND version_id=?", path, version)

	err := row.Scan(&json_data)
	if err != nil {
		log.Fatal("Error while querying database  in GetFileDelta() : ", err)
	}

	err = json.Unmarshal([]byte(json_data), &delta)

	if err != nil {
		log.Fatal("Error while decoding json data for binary delta : ", err)
	}

	return delta
}

func (acces *AccesBdd) AddFolderToRetard(path string) {

	// add a line in retard table with all devices linked and the version number

	MODTYPES := map[string]string{
		"creation": "c",
		"delete":   "d",
		"patch":    "p",
		"move":     "m",
	}
	offline_devices := acces.GetSyncOfflineDevices()

	if offline_devices.Size() > 0 {
		var str_ids string = ""
		for i := 0; i < offline_devices.Size(); i++ {
			str_ids += offline_devices.Get(i) + ";"
		}
		// remove the last semicolon
		str_ids = str_ids[:len(str_ids)-1]

		log.Println("ADDING FOLDER TO RETARD : ", path)
		_, err := acces.db_handler.Exec("INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,\"folder\",?)", 1, path, MODTYPES["creation"], str_ids, acces.SecureId)

		if err != nil {
			log.Fatal("Error while inserting new retard in AddFolderToRetard() : ", err)
		}
	}

}

func (acces *AccesBdd) IsDeviceLinked(device_id string) bool {
	rows, err := acces.db_handler.Query("SELECT id FROM linked_devices WHERE device_id=?", device_id)

	if err != nil {
		log.Fatal("Error while querying database from IsDeviceLinked() : ", err)
	}
	defer rows.Close()

	var i int = 0
	for rows.Next() && i == 0 {
		i++
	}

	return i > 0

}

func (acces *AccesBdd) IsMyDeviceIdGenerated() bool {
	rows, err := acces.db_handler.Query("SELECT id FROM mesid")

	if err != nil {
		log.Fatal("Error while querying database from IsMyDeviceIdGenerated() : ", err)
	}
	defer rows.Close()

	var i int = 0
	for rows.Next() && i == 0 {
		i++
	}

	return i > 0
}

func (acces *AccesBdd) GenerateMyDeviceId() {

	// generating secure_id

	alphabet := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	s := make([]rune, 41)

	for i := range s {
		s[i] = alphabet[rand.Intn(len(alphabet))]
	}

	secure_id := string(s)

	// adding it to database
	acces.db_handler.Exec("INSERT INTO mesid(device_id) VALUES(?)", secure_id)

}
func (acces *AccesBdd) GetMyDeviceId() string {
	row := acces.db_handler.QueryRow("SELECT device_id FROM mesid")

	if row.Err() != nil {
		log.Fatal("Error while querying database from GetMyDeviceId() : ", row.Err())
	}

	var device_id string

	row.Scan(&device_id)

	return device_id
}

func (acces *AccesBdd) GetOfflineDevices() globals.GenArray[string] {

	rows, err := acces.db_handler.Query("SELECT device_id,is_connected FROM linked_devices")

	if err != nil {
		log.Fatal("Error while querying database from GetOfflineDevices() : ", err)
	}
	defer rows.Close()
	var offline_devices globals.GenArray[string]

	for rows.Next() {
		var device LinkDevice
		rows.Scan(&device.SecureId, &device.IsConnected)
		if !device.IsConnected {
			offline_devices.Add(device.SecureId)
		}
	}

	return offline_devices
}

func (acces *AccesBdd) GetOnlineDevices() globals.GenArray[string] {

	rows, err := acces.db_handler.Query("SELECT device_id,is_connected FROM linked_devices")

	if err != nil {
		log.Fatal("Error while querying database from GetOnlineDevices() : ", err)
	}
	defer rows.Close()

	var online_devices globals.GenArray[string]

	for rows.Next() {
		var device LinkDevice
		rows.Scan(&device.SecureId, &device.IsConnected)

		if device.IsConnected {
			online_devices.Add(device.SecureId)
		}
	}

	return online_devices
}

func (acces *AccesBdd) GetSyncOnlineDevices() globals.GenArray[string] {
	linked_devices := acces.GetSyncLinkedDevices()

	/*var str_ids string = ""
	for i := 0; i < linked_devices.Size(); i++ {
		str_ids += linked_devices.Get(i) + ","
	}
	// remove the last colon
	str_ids = str_ids[:len(str_ids)-1]

	query := "SELECT device_id,is_connected FROM linked_devices WHERE device_id IN ('"
	query += str_ids
	query += "')"

	rows, err := acces.db_handler.Query(query)

	if err != nil {
		log.Fatal("Error while querying database from GetSyncOfflineDevices() : ", err)
	}
	defer rows.Close()

	var online_devices globals.GenArray[string]

	for rows.Next() {
		var device LinkDevice
		rows.Scan(&device.SecureId, &device.IsConnected)

		if device.IsConnected {
			online_devices.Add(device.SecureId)
		}
	}*/

	var online_devices globals.GenArray[string]

	for i := 0; i < linked_devices.Size(); i++ {
		if acces.GetDeviceConnectionState(linked_devices.Get(i)) {
			log.Println(linked_devices.Get(i))
			online_devices.Add(linked_devices.Get(i))
		}
	}

	return online_devices
}
func (acces *AccesBdd) SetDeviceIP(device_id string, value string) {
	log.Println("vars : ", acces.SecureId, device_id, value)
	_, err := acces.db_handler.Exec("UPDATE linked_devices SET ip_addr=? WHERE secure_id=? AND device_id=?", value, acces.SecureId, device_id)

	if err != nil {
		log.Fatal("Error while updating database in SetDeviceIP() : ", err)
	}
}

func (acces *AccesBdd) GetDeviceIP(device_id string) string {
	var ip_addr string
	row := acces.db_handler.QueryRow("SELECT ip_addr FROM linked_devices WHERE device_id=?", device_id)

	err := row.Scan(&ip_addr)
	if err != nil {
		log.Fatal("Error while querying database in GetDeviceIP() : ", err)
	}

	return ip_addr
}

func (acces *AccesBdd) IncrementFileVersion(path string) {
	// get lastest version of file and increment it
	new_version_id := acces.GetFileLastVersionId(path) + 1

	// update version number in db
	_, err := acces.db_handler.Exec("UPDATE filesystem SET version_id=? WHERE path=? AND secure_id=?", new_version_id, path, acces.SecureId)

	if err != nil {
		log.Fatal("Error while updating file version_id in IncrementFileVersion()")
	}
}

func (acces *AccesBdd) UpdateCachedFile(relative_path string, absolute_path string) {
	// reads the current state of the given file and update it in the database

	file_content, err := os.ReadFile(absolute_path)

	var bytes_buffer bytes.Buffer

	gzip_writer := gzip.NewWriter(&bytes_buffer)

	gzip_writer.Write(file_content)

	gzip_writer.Close()

	if err != nil {
		log.Fatal("Error in UpdateCachedFile() while reading file to cache its content : ", err)
	}

	_, err = acces.db_handler.Exec("UPDATE filesystem SET content=?,size=? WHERE path=? AND secure_id=?", bytes_buffer.Bytes(), bytes_buffer.Len(), relative_path, acces.SecureId)

	if err != nil {
		log.Fatal("Error in UpdateCachedFile() while caching file content into bdd : ", err)
	}
}

func (acces *AccesBdd) ListSyncAllTasks() globals.GenArray[SyncInfos] {

	// used to get all sync task secure_id and root path listed
	// returns a custom type containing the two values as string

	rows, err := acces.db_handler.Query("SELECT secure_id,root,backup_mode FROM sync")

	if err != nil {
		log.Fatal("Error while querying database from ListSyncAllTasks() : ", err)
	}
	defer rows.Close()

	var list globals.GenArray[SyncInfos]

	for rows.Next() {
		var info SyncInfos
		rows.Scan(&info.SecureId, &info.Path, &info.BackupMode)

		if acces.IsApp(info.SecureId) {
			config := acces.GetAppConfig(info.SecureId)
			info.Name = config.AppName
			info.IsApp = true
		}
		list.Add(info)
	}

	return list

}

func (acces *AccesBdd) IsApp(secure_id string) bool {
	// used to determine if a gived secure_id is associated with a sync task used by an application

	rows, err := acces.db_handler.Query("SELECT * FROM apps WHERE secure_id=?", secure_id)
	if err != nil {
		// Handle the error properly
		log.Fatal("Error in bdd.IsApp()", err)
		return false
	}
	defer rows.Close()

	// Check if any rows were returned
	if rows.Next() {
		// At least one row was returned
		return true
	} else {
		// No rows were returned
		return false
	}
}

func (acces *AccesBdd) BuildEventQueueFromRetard(device_id string) map[string]*globals.GenArray[*globals.QEvent] {

	// as the device can be late on many tasks, we must create an hash table with all
	// the differents delta on all differents tasks he's late on
	var queue map[string]*globals.GenArray[*globals.QEvent] = make(map[string]*globals.GenArray[*globals.QEvent], 0)

	log.Println("Building missed files event queue from retard...")
	rows, err := acces.db_handler.Query("SELECT r.secure_id,d.delta,r.mod_type,r.path,r.type FROM retard AS r JOIN delta AS d ON r.path=d.path AND r.version_id=d.version_id AND r.secure_id=d.secure_id WHERE r.devices_to_patch LIKE ?", "%"+device_id+"%")

	if err != nil {
		log.Fatal("Error while querying database from BuildEventQueueFromRetard() : ", err)
	}
	defer rows.Close()

	for rows.Next() {
		var event globals.QEvent
		var delta_bytes []byte
		var delta delta_binaire.Delta
		var mod_type string
		var secure_id string
		var filepath string
		var file_type string

		MODTYPES_REVERSE := map[string]string{
			"c": "[CREATE]",
			"d": "[REMOVE]",
			"p": "[UPDATE]",
			"m": "[MOVE]",
		}

		rows.Scan(&secure_id, &delta_bytes, &mod_type, &filepath, &file_type)

		event.Flag = MODTYPES_REVERSE[mod_type]

		delta.DeSerialize(delta_bytes)

		event.SecureId = secure_id
		event.FileType = file_type
		event.FilePath = filepath
		event.Delta = delta

		log.Println("ADDING EVENT : ", event)

		if queue[secure_id] == nil {
			var genA globals.GenArray[*globals.QEvent]
			queue[secure_id] = &genA
		}
		queue[secure_id].Add(&event)
	}

	log.Println("Building missed folders creation event queue from retard...")
	rows, err = acces.db_handler.Query("SELECT r.secure_id,r.mod_type,r.path,r.type FROM retard AS r WHERE r.devices_to_patch LIKE ? AND r.type='folder'", "%"+device_id+"%")

	if err != nil {
		log.Fatal("Error while querying database from BuildEventQueueFromRetard() : ", err)
	}
	defer rows.Close()

	for rows.Next() {
		var event globals.QEvent
		var mod_type string
		var secure_id string
		var filepath string
		var file_type string

		MODTYPES_REVERSE := map[string]string{
			"c": "[CREATE]",
			"d": "[REMOVE]",
			"p": "[UPDATE]",
			"m": "[MOVE]",
		}

		rows.Scan(&secure_id, &mod_type, &filepath, &file_type)

		event.Flag = MODTYPES_REVERSE[mod_type]
		event.SecureId = secure_id
		event.FileType = file_type
		event.FilePath = filepath

		log.Println("ADDING EVENT : ", event)
		if queue[secure_id] == nil {
			var genA globals.GenArray[*globals.QEvent]
			queue[secure_id] = &genA
		}
		queue[secure_id].Add(&event)
	}

	log.Println("Retard queue : ", queue)

	return queue

}

func (acces *AccesBdd) RemoveDeviceFromRetard(device_id string) {

	var ids_str string
	var ids_list globals.GenArray[string]

	row := acces.db_handler.QueryRow("SELECT devices_to_patch FROM retard WHERE devices_to_patch LIKE ?", "%"+device_id+"%")

	err := row.Scan(&ids_str)
	if err != nil {
		log.Fatal("Error while querying database RemoveDeviceFromRetard() : ", err)
	}

	for _, val := range strings.Split(ids_str, ";") {
		ids_list.Add(val)
	}

	// same list of sync tasks secure_id but without this one
	var new_ids globals.GenArray[string]
	for i := 0; i < ids_list.Size(); i++ {
		if !(ids_list.Get(i) == device_id) {
			new_ids.Add(ids_list.Get(i))
		}
	}

	// if it was the last device to being late, we suppress the row from the table
	// if not we just rewrite without its id
	if new_ids.Size() > 0 {
		// rewrite the updated list

		var str_ids string = ""
		for i := 0; i < new_ids.Size(); i++ {
			str_ids += new_ids.Get(i) + ";"
		}
		// remove the last semicolon
		str_ids = str_ids[:len(str_ids)-1]

		_, err = acces.db_handler.Exec("UPDATE retard SET devices_to_patch= ? WHERE devices_to_patch LIKE ?", str_ids, "%"+device_id+"%")

		if err != nil {
			log.Fatal("Error while updating database in LinkDevice() : ", err)
		}

	} else {
		// rewrite the updated list
		_, err = acces.db_handler.Exec("DELETE FROM retard WHERE devices_to_patch LIKE ?", "%"+device_id+"%")

		if err != nil {
			log.Fatal("Error while updating database in LinkDevice() : ", err)
		}
	}

}

func (acces *AccesBdd) RemoveDeviceFromRetardOneFile(device_id string, relative_path string, version_id int64) {

	// to replace, this code is the one from FileSystemPatchLockState

	var ids_str string
	var ids_list globals.GenArray[string]

	row := acces.db_handler.QueryRow(
		"SELECT devices_to_patch FROM retard WHERE devices_to_patch LIKE ? AND path=? AND version_id=? AND secure_id=?",
		"%"+device_id+"%",
		relative_path,
		version_id,
		acces.SecureId,
	)

	err := row.Scan(&ids_str)
	if err != nil && err != sql.ErrNoRows {
		log.Fatal("Error while querying database RemoveDeviceFromRetard() : ", err)
	}

	// main reason : a problem led to unsiycnhronised files versionning,
	// retrying with the oldest version present in retard table
	// associated with the device id
	if err == sql.ErrNoRows {
		rows, err := acces.db_handler.Query(
			"SELECT min(version_id) FROM retard WHERE devices_to_patch LIKE ? AND path=? AND secure_id=?",
			"%"+device_id+"%",
			relative_path,
			acces.SecureId,
		)

		// no rows excepted as it can happen if all devices are online
		if err != nil {
			//log.Fatal("Error while querying database RemoveDeviceFromRetard() : ", err)
			log.Println("Nothing to remove in retard table. Nothing unusual. : ", err)
			return
		}

		if rows.Next() {
			rows.Scan(&version_id)

		}
		rows.Close()
		// update version id with the closest one to a true match
		// as the event that came first has the best chance to be patched first
		// (not assured but the others are coming very soon if there are any so not really a problem )

		// now we can retry :)
		row = acces.db_handler.QueryRow(
			"SELECT devices_to_patch FROM retard WHERE devices_to_patch LIKE ? AND path=? AND version_id=? AND secure_id=?",
			"%"+device_id+"%",
			relative_path,
			version_id,
			acces.SecureId,
		)

		err = row.Scan(&ids_str)
		// no rows excepted as it can happen if all devices are online
		if err != nil && !(err == sql.ErrNoRows) {
			log.Fatal("Error while querying database RemoveDeviceFromRetard() : ", err)

		}

	}

	for _, val := range strings.Split(ids_str, ";") {
		ids_list.Add(val)
	}

	// same list of sync tasks secure_id but without this one
	var new_ids globals.GenArray[string]
	for i := 0; i < ids_list.Size(); i++ {
		if !(ids_list.Get(i) == device_id) {
			new_ids.Add(ids_list.Get(i))
		}
	}

	// if it was the last device to being late, we suppress the row from the table
	// if not we just rewrite without its id
	if new_ids.Size() > 0 {
		// rewrite the updated list

		var str_ids string = ""
		for i := 0; i < new_ids.Size(); i++ {
			str_ids += new_ids.Get(i) + ";"
		}
		// remove the last semicolon
		str_ids = str_ids[:len(str_ids)-1]

		_, err = acces.db_handler.Exec(
			"UPDATE retard SET devices_to_patch= ? WHERE devices_to_patch LIKE ? AND path=? AND version_id=? AND secure_id=?",
			str_ids,
			"%"+device_id+"%",
			relative_path,
			version_id,
			acces.SecureId,
		)

		if err != nil {
			log.Fatal("Error while updating database in RemoveDeviceFromRetardOneFile() : ", err)
		}

	} else {
		// rewrite the updated list
		_, err = acces.db_handler.Exec("DELETE FROM retard WHERE devices_to_patch LIKE ? AND path=? AND version_id=? AND secure_id=?",
			"%"+device_id+"%",
			relative_path,
			version_id,
			acces.SecureId,
		)

		if err != nil {
			log.Fatal("Error while updating database in RemoveDeviceFromRetardOneFile() : ", err)
		}
	}

}

// this function checks if a device has some updates to catch up on
func (acces *AccesBdd) NeedsUpdate(device_id string) bool {

	var ids_str string

	row := acces.db_handler.QueryRow("SELECT devices_to_patch FROM retard WHERE devices_to_patch LIKE ?", "%"+device_id+"%")

	err := row.Scan(&ids_str)
	if (err != nil) && (err != sql.ErrNoRows) {
		log.Fatal("Error while querying database in NeedsUpdate() : ", err)

	}
	return !(err == sql.ErrNoRows)

}

// ajoute une application tout en un dans la table exprès
func (acces *AccesBdd) AddToutEnUn(data *globals.ToutEnUnConfig) {
	_, err := acces.db_handler.Exec(
		"INSERT INTO apps (name,path,version_id,type,secure_id,uninstaller_path) VALUES(?,?,?,\"toutenun\",?,?)",
		data.AppName,
		data.AppLauncherPath,
		1,
		acces.SecureId,
		data.AppUninstallerPath,
	)

	if err != nil {
		log.Fatal("Error while updating database in AddToutEnUn() : ", err)
	}
}

func (acces *AccesBdd) AddGrapin(data *globals.GrapinConfig) {
	_, err := acces.db_handler.Exec("INSERT INTO apps (name,path,version_id,type,secure_id,uninstaller_path) VALUES(?,?,?,\"grapin\",?,?)", data.AppName, "[GRAPIN]", 1, acces.SecureId, "[GRAPIN]")

	if err != nil {
		log.Fatal("Error while updating database in AddGrapin() : ", err)
	}
}

// this function list all installed apps on qsync
func (acces *AccesBdd) ListInstalledApps() globals.GenArray[*globals.MinGenConfig] {

	var configs globals.GenArray[*globals.MinGenConfig]
	var tmp globals.MinGenConfig

	rows, err := acces.db_handler.Query("SELECT name,id,path,type FROM apps")
	if err != nil {
		log.Fatal("Error while querying database in ListInstalledApps() : ", err)
	}

	for rows.Next() {
		err = rows.Scan(&tmp)
		if (err != nil) && (err != sql.ErrNoRows) {
			log.Fatal("Error while querying database in ListInstalledApps() : ", err)

		}

		configs.Add(&tmp)
	}

	return configs

}

// this function get details of a specifi app by its ID
func (acces *AccesBdd) GetAppConfig(secure_id string) globals.MinGenConfig {

	var config globals.MinGenConfig

	row := acces.db_handler.QueryRow(
		"SELECT name,id,path,type,secure_id,uninstaller_path FROM apps WHERE secure_id=?",
		secure_id,
	)

	err := row.Scan(&config.AppName, &config.AppId, &config.BinPath, &config.Type, &config.SecureId, &config.UninstallerPath)
	if (err != nil) && (err != sql.ErrNoRows) {
		log.Fatal("Error while querying database in GetAppConfig() : ", err)
	}

	return config

}

// this function get details of a specifi app by its ID
func (acces *AccesBdd) DeleteApp(secure_id string) {

	_, err := acces.db_handler.Exec("DELETE FROM apps WHERE secure_id=?", secure_id)

	if (err != nil) && (err != sql.ErrNoRows) {
		log.Fatal("Error while querying database in GetAppConfig() : ", err)
	}

}

// this function is just getting the value of wether the user wants to
// receive largage aerien or not
func (acces *AccesBdd) AreLargageAerienAllowed() bool {

	var ret bool
	row := acces.db_handler.QueryRow("SELECT accepte_largage_aerien FROM mesid")

	err := row.Scan(&ret)
	if (err != nil) && (err != sql.ErrNoRows) {
		log.Fatal("Error while querying database in AreLargageAerienAllowed() : ", err)
	}

	return ret
}

// if true set to false, if false set to true
// setting the value if the use wants or not to receive largage aerien
func (acces *AccesBdd) SwitchLargageAerienAllowingState() bool {

	var ret bool
	_, err := acces.db_handler.Exec("UPDATE mesid SET accepte_largage_aerien=NOT accepte_largage_aerien")

	if (err != nil) && (err != sql.ErrNoRows) {
		log.Fatal("Error while querying database in SwitchLargageAerienAllowingState() : ", err)
	}

	return ret
}

func (acces *AccesBdd) RemoveDeviceFromNetworkMap(device_id, ip_addr string) {
	_, err := acces.db_handler.Exec("DELETE FROM reseau WHERE device_id=? AND ip_addr=?", device_id, ip_addr)
	if err != nil {
		log.Fatal("Error while executing query in RemoveDeviceFromNetworkMap(): ", err)
	}
}

func (acces *AccesBdd) AddDeviceToNetworkMap(device_id, ip_addr, hostname string) {
	_, err := acces.db_handler.Exec("INSERT INTO reseau(device_id, ip_addr, hostname) VALUES(?, ?, ?)", device_id, ip_addr, hostname)
	if err != nil {
		log.Fatal("Error while executing query in AddDeviceToNetworkMap(): ", err)
	}
}

func (acces *AccesBdd) GetNetworkMap() globals.GenArray[map[string]string] {
	rows, err := acces.db_handler.Query("SELECT device_id, ip_addr, hostname FROM reseau")
	if err != nil {
		log.Fatal("Error while executing query in GetNetworkMap(): ", err)
	}
	defer rows.Close()

	var ret globals.GenArray[map[string]string]
	for rows.Next() {
		var deviceID, ipAddr, hostname string
		err := rows.Scan(&deviceID, &ipAddr, &hostname)
		if err != nil {
			log.Fatal("Error while scanning rows in GetNetworkMap(): ", err)
		}

		device := map[string]string{
			"device_id": deviceID,
			"ip_addr":   ipAddr,
			"hostname":  hostname,
		}
		ret.Add(device)
	}

	return ret
}

func (acces *AccesBdd) IsDeviceOnNetworkMap(ipAddr string) bool {
	row := acces.db_handler.QueryRow("SELECT * FROM reseau WHERE ip_addr=?", ipAddr)
	var deviceID, ipAddrDB, hostname string
	err := row.Scan(&deviceID, &ipAddrDB, &hostname)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}

		log.Fatal("Error while executing query in IsDeviceOnNetworkMap(): ", err)
	}

	return true
}

func (acces *AccesBdd) CleanNetworkMap() {
	_, err := acces.db_handler.Exec("DELETE FROM reseau")
	if err != nil {
		log.Fatal("Error while executing query in CleanNetworkMap(): ", err)
	}
}

func (acces *AccesBdd) IsSyncInBackupMode() bool {
	row := acces.db_handler.QueryRow("SELECT backup_mode FROM sync WHERE secure_id=?", acces.SecureId)
	var bcp_mode bool = false
	err := row.Scan(&bcp_mode)
	if err != nil {
		log.Fatal("Error while executing query in IsSyncInBackupMode(): ", err)
	}

	return bcp_mode

}
func (acces *AccesBdd) ToggleBackupMode() {
	_, err := acces.db_handler.Exec("UPDATE sync SET backup_mode = NOT backup_mode WHERE secure_id=?", acces.SecureId)

	if err != nil {
		log.Fatal("Error while executing query in ToggleBackupMode(): ", err)
	}

}

func (acces *AccesBdd) UpdateSyncId(root_path string, secure_id string) {

	// get old sync id
	var old_sync_id string
	row := acces.db_handler.QueryRow("SELECT secure_id FROM sync WHERE root LIKE ?", root_path+"%")

	err := row.Scan(&old_sync_id)

	if err == sql.ErrNoRows {
		log.Fatal("The provided path does not match any existing sync rootpath")
	}

	if err != nil {
		log.Fatal("Error while querying database GetSecureIdFromRootPath() : ", err)
	}

	// update the sync row
	_, err = acces.db_handler.Exec("UPDATE sync SET secure_id = ? WHERE secure_id=?", secure_id, old_sync_id)

	if err != nil {
		log.Fatal("Error while executing query in UpdateSyncId(): ", err)
	}

	// update the app row
	_, err = acces.db_handler.Exec("UPDATE apps SET secure_id = ? WHERE secure_id=?", secure_id, old_sync_id)

	if err != nil {
		log.Fatal("Error while executing query in UpdateSyncId(): ", err)
	}

	// change all registered files to have this secure id
	_, err = acces.db_handler.Exec("UPDATE filesystem SET secure_id = ? WHERE secure_id=?", secure_id, old_sync_id)

	if err != nil {
		log.Fatal("Error while executing query in UpdateSyncId(): ", err)
	}

}

func (acces *AccesBdd) GetSyncCreationDate() int64 {
	var timestamp int64
	row := acces.db_handler.QueryRow("SELECT creation_date FROM sync WHERE secure_id=?", acces.SecureId)

	err := row.Scan(&timestamp)

	if err != nil {
		log.Fatal("Error while querying database GetSyncCreationDate() : ", err)
	}

	return timestamp
}
func (acces *AccesBdd) GetSyncCreationDateFromPathMatch(root_path string) int64 {
	var timestamp int64
	row := acces.db_handler.QueryRow("SELECT creation_date FROM sync WHERE root LIKE ?", root_path+"%")

	err := row.Scan(&timestamp)

	if err != nil {
		log.Fatal("Error while querying database GetSyncCreationDate() : ", err)
	}

	return timestamp
}

func (acces *AccesBdd) SyncStillExists() bool {
	var count int = 0
	err := acces.db_handler.QueryRow(
		"SELECT COUNT(*) FROM sync WHERE secure_id=?",
		acces.SecureId,
	).Scan(&count)

	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}

	return count > 0

}
