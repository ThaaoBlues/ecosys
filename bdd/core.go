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
	dtbin "qsync/delta_binaire"
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
	Path     string
	SecureId string
}

type LinkDevice struct {
	SecureId    string
	IsConnected bool
}

// InitConnection initializes the database connection and creates necessary tables if they don't exist.
// This function is used everytime we create an AccesBdd object
func (bdd *AccesBdd) InitConnection() {
	var err error
	bdd.db_handler, err = sql.Open("sqlite3", "qsync.db")

	if err != nil {
		log.Fatal("An error occured while connecting to the qsync database.", err)
	}

	_, err = bdd.db_handler.Exec(`CREATE TABLE IF NOT EXISTS retard(
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

	_, err = bdd.db_handler.Exec(`CREATE TABLE IF NOT EXISTS delta(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT,
		version_id INTEGER,
		delta TEXT,
		secure_id TEXT
	)`)
	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}

	_, err = bdd.db_handler.Exec(`CREATE TABLE IF NOT EXISTS filesystem(
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

	_, err = bdd.db_handler.Exec(`CREATE TABLE IF NOT EXISTS sync(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		secure_id TEXT,
		linked_devices_id TEXT DEFAULT "",
		root TEXT
	)`)
	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}

	_, err = bdd.db_handler.Exec(`CREATE TABLE IF NOT EXISTS linked_devices(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id TEXT,
		is_connected BOOLEAN,
		receiving_update TEXT,
		ip_addr TEXT
	)`)
	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}
	_, err = bdd.db_handler.Exec(`CREATE TABLE IF NOT EXISTS mesid(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id TEXT
	)`)
	if err != nil {
		log.Fatal("Error while creating table : ", err)
	}

	if !bdd.IsMyDeviceIdGenerated() {
		bdd.GenerateMyDeviceId()
	}
}

// CloseConnection closes the database connection.
func (bdd *AccesBdd) CloseConnection() {
	bdd.db_handler.Close()
}

// CheckFileExists checks if a file with a given path exists in the database.
func (bdd *AccesBdd) CheckFileExists(path string) bool {
	rows, err := bdd.db_handler.Query("SELECT id FROM filesystem WHERE path=? AND secure_id=? VALUES=(?,?)", path, bdd.SecureId)

	if err != nil {
		log.Fatal("Error while querying database from CheckFileExists() : ", err)
	}
	defer rows.Close()

	var i int = 0

	for rows.Next() && i == 0 {
		/*var id int

		if err := rows.Scan(&id); err != nil {
			log.Fatal("Error while querying database ", err)
		}*/
		i++

	}

	return i > 0

}

// WasFile checks if a file with a given path was present in the past.
// This is made by checking the database as the filesystem may have been alterated
func (bdd *AccesBdd) WasFile(path string) bool {

	row := bdd.db_handler.QueryRow("SELECT type FROM filesystem WHERE path=? AND secure_id=?", path, bdd.SecureId)

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
func (bdd *AccesBdd) IsFile(path string) bool {

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

// GetSecureId retrieves the secure ID associated with a given root path.
func (bdd *AccesBdd) GetSecureId(rootpath string) {
	row := bdd.db_handler.QueryRow("SELECT secure_id FROM sync WHERE root=?", rootpath)

	err := row.Scan(&bdd.SecureId)

	if err == sql.ErrNoRows {
		log.Fatal("The provided path does not match any existing sync rootpath")
	}

	if err != nil {
		log.Fatal("Error while querying database ", err)
	}

}

// CreateFile adds a file to the database.
func (bdd *AccesBdd) CreateFile(path string) {

	file_handler, err := os.Open(path)

	if err != nil {
		log.Fatal("Error while opening the file from real filesystem to seek changes.", err)
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

	_, err = bdd.db_handler.Exec("INSERT INTO filesystem (path, version_id, type, size, secure_id,content) VALUES (?, 0, 'file', ?, ?,?)", path, stat.Size(), bdd.SecureId, bytes_buffer.Bytes())

	if err != nil {
		log.Fatal("Error while inserting into database ", err)
	}
}

// GetFileLastVersionId retrieves the last version ID of a file.
func (bdd *AccesBdd) GetFileLastVersionId(path string) int64 {
	row := bdd.db_handler.QueryRow("SELECT version_id FROM filesystem WHERE path=? AND secure_id=?", path, bdd.SecureId)

	var version_id int64
	err := row.Scan(&version_id)
	if err != nil {
		log.Fatal("Error while querying database ", err)
	}

	return version_id
}

// UpdateFile updates a file in the database with a new version.
// For that, a binary delta object is used.
func (bdd *AccesBdd) UpdateFile(path string, delta dtbin.Delta) {

	new_version_id := bdd.GetFileLastVersionId(path) + 1

	bdd.IncrementFileVersion(path)

	// convert detla to json
	json_data, err := json.Marshal(delta)

	if err != nil {
		log.Fatal("Error while creating json object from delta type : ", err)
	}

	// add line in delta table

	_, err = bdd.db_handler.Exec("INSERT INTO delta (path,version_id,delta,secure_id) VALUES(?,?,?,?)", path, new_version_id, json_data, bdd.SecureId)

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

	offline_linked_devices := bdd.GetOfflineDevices()

	_, err = bdd.db_handler.Exec("INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,\"file\",?)", new_version_id, path, MODTYPES["patch"], strings.Join(offline_linked_devices, ";"), bdd.SecureId)

	if err != nil {
		log.Fatal("Error while inserting new retard : ", err)
	}

	// update the cached file content to build the next delta
	bdd.UpdateCachedFile(path)
}

func (bdd *AccesBdd) NotifyDeviceUpdate(path string, device_id string) {
	// remove all mentions of the given device_id in the retard table for a specific file

	var devices_to_patch string
	row := bdd.db_handler.QueryRow("SELECT devices_to_patch FROM retard WHERE path=? AND secure_id=?", path, bdd.SecureId)

	row.Scan(&devices_to_patch)

	devices_split := strings.Split(devices_to_patch, ";")

	var list_builder []string

	for _, dev := range devices_split {
		if !(dev == device_id) {
			list_builder = append(list_builder, dev)
		}
	}

	_, err := bdd.db_handler.Exec("UPDATE retard SET WHERE path=? AND secure_id=?", list_builder, path)

	if err != nil {
		log.Fatal("Error while inserting new retard : ", err)
	}

}

// GetFileContent retrieves the content of a file from the database.
// returned as byte array
func (bdd *AccesBdd) GetFileContent(path string) []byte {
	row := bdd.db_handler.QueryRow("SELECT content FROM filesystem WHERE path=? AND secure_id=?", path, bdd.SecureId)

	var compressed_content []byte
	err := row.Scan(&compressed_content)
	if err != nil {
		log.Fatal("Error while querying database ", err)
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
func (bdd *AccesBdd) RmFile(path string) {
	_, err := bdd.db_handler.Exec("DELETE FROM filesystem WHERE path=? AND secure_id=?", path, bdd.SecureId)

	if err != nil {
		log.Fatal("Error while deleting from database ", err)
	}

	// now, purge all data involving this file from retard table
	_, err = bdd.db_handler.Exec("DELETE FROM retard WHERE path=? AND secure_id=?", path, bdd.SecureId)

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
	linked_devices := bdd.GetOfflineDevices()

	_, err = bdd.db_handler.Exec("INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,\"file\",?)", 0, path, MODTYPES["delete"], strings.Join(linked_devices, ";"), bdd.SecureId)

	if err != nil {
		log.Fatal("Error while inserting new retard : ", err)
	}

}

// CreateFolder adds a folder to the database.
func (bdd *AccesBdd) CreateFolder(path string) {
	_, err := bdd.db_handler.Exec("INSERT INTO filesystem (path, version_id, type, size, secure_id) VALUES (?, 0, 'folder', 0, ?)", path, bdd.SecureId)

	if err != nil {
		log.Fatal("Error while inserting into database ", err)
	}
}

// RmFolder deletes a folder from the database and adds it in delete mode to the retard table.
func (bdd *AccesBdd) RmFolder(path string) {

	_, err := bdd.db_handler.Exec("DELETE FROM filesystem WHERE path LIKE ? AND secure_id=?", path+"%", bdd.SecureId)

	if err != nil {
		log.Fatal("Error while deleting from database ", err)
	}

	// now, purge all data involving this folder from retard table
	_, err = bdd.db_handler.Exec("DELETE FROM retard WHERE path LIKE ? AND secure_id=?", path+"%", bdd.SecureId)

	if err != nil {
		log.Fatal("Error while deleting from database ", err)
	}

	// purge all data from delta table involving this folder
	_, err = bdd.db_handler.Exec("DELETE FROM delta WHERE path LIKE ? AND secure_id=?", path+"%", bdd.SecureId)

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
	linked_devices := bdd.GetOfflineDevices()

	_, err = bdd.db_handler.Exec("INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,\"folder\",?)", 0, path, MODTYPES["delete"], strings.Join(linked_devices, ";"), bdd.SecureId)

	if err != nil {
		log.Fatal("Error while inserting new retard : ", err)
	}

}

// Move updates the path of a file or folder in the database and adds a move event to the retard table.
func (bdd *AccesBdd) Move(path string, new_path string, file_type string) {
	_, err := bdd.db_handler.Exec("UPDATE filesystem SET path=? WHERE path=? AND secure_id=?", new_path, path, bdd.SecureId)
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

	linked_devices := bdd.GetOfflineDevices()

	_, err = bdd.db_handler.Exec("INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,?,?)", 0, path, MODTYPES["move"], strings.Join(linked_devices, ";"), file_type, bdd.SecureId)

	if err != nil {
		log.Fatal("Error while inserting new retard : ", err)
	}

}

// CreateSync initializes a new synchronization entry in the database.
func (bdd *AccesBdd) CreateSync(rootPath string) {

	// generating secure_id

	alphabet := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	s := make([]rune, 41)

	for i := range s {
		s[i] = alphabet[rand.Intn(len(alphabet))]
	}

	secure_id := string(s)

	bdd.SecureId = secure_id

	// adding sync to database
	_, err := bdd.db_handler.Exec("INSERT INTO sync (secure_id, root) VALUES(?,?)", secure_id, rootPath)

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
		if info.IsDir() {
			bdd.CreateFolder(path)
		} else {
			bdd.CreateFile(path)
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
func (bdd *AccesBdd) CreateSyncFromOtherEnd(rootPath string, secure_id string) {
	bdd.SecureId = secure_id

	// adding sync to database
	_, err := bdd.db_handler.Exec("INSERT INTO sync (secure_id, root) VALUES(?,?)", secure_id, rootPath)

	if err != nil {
		log.Fatal("Error while inserting into database ", err)
	}
}

// RmSync removes a synchronization entry from the database.

func (bdd *AccesBdd) RmSync() {
	_, err := bdd.db_handler.Exec("DELETE FROM sync WHERE secure_id=?", bdd.SecureId)

	if err != nil {
		log.Fatal("Error while deleting from database ", err)
	}
}

// LinkDevice links a device to the synchronization entry.
func (bdd *AccesBdd) LinkDevice(device_id string, ip_addr string) {

	// link this device to an existing task
	_, err := bdd.db_handler.Exec("UPDATE sync SET linked_devices_id=IFNULL(linked_devices_id, '') || ? WHERE secure_id=?", device_id+";", bdd.SecureId)

	if err != nil {
		log.Fatal("Error while updating database in LinkDevice() : ", err)
	}

	// if the device is not registered as a target (from previous tasks), register it
	if !bdd.IsDeviceLinked(device_id) {
		_, err = bdd.db_handler.Exec("INSERT INTO linked_devices (device_id,is_connected,ip_addr) VALUES(?,TRUE,?)", device_id, ip_addr)

		if err != nil {
			log.Fatal("Error while updating database in LinkDevice() : ", err)
		}
	}

}

// UnlinkDevice unlinks a device from a synchronization entry.
func (bdd *AccesBdd) UnlinkDevice(device_id string) {

	// remove id from the list
	_, err := bdd.db_handler.Exec("DELETE FROM linked_devices WHERE device_id=?", device_id)

	if err != nil {
		log.Fatal("Error while updating database in UnlinkDevice() : ", err)
	}

	// remove id from the list in sync table
	_, err = bdd.db_handler.Exec("UPDATE sync SET linked_devices_id=REPLACE(linked_devices_id,?,'')", device_id+";", bdd.SecureId)

	if err != nil {
		log.Fatal("Error while updating database in UnlinkDevice() : ", err)
	}
}

// GetRootSyncPath retrieves the root path associated with the synchronization entry.
func (bdd *AccesBdd) GetRootSyncPath() string {
	var rootPath string
	row := bdd.db_handler.QueryRow("SELECT root FROM sync WHERE secure_id=?", bdd.SecureId)

	err := row.Scan(&rootPath)
	if err != nil {
		log.Fatal("Error while querying database ", err)
	}

	return rootPath
}

// SetDeviceConnectionState updates the connection state of a linked device.
func (bdd *AccesBdd) SetDeviceConnectionState(device_id string, value bool, ip_addr ...string) {

	if len(ip_addr) == 0 {
		_, err := bdd.db_handler.Exec("UPDATE linked_devices SET is_connected=? WHERE device_id=?", value, device_id)

		if err != nil {
			log.Fatal("Error while updating database in SetDeviceConnectionState() : ", err)
		}
	} else {
		_, err := bdd.db_handler.Exec("UPDATE linked_devices SET is_connected=?,ip_addr=? WHERE device_id=?", value, ip_addr[0], device_id)

		if err != nil {
			log.Fatal("Error while updating database in SetDeviceConnectionState() : ", err)
		}
	}

}

func (bdd *AccesBdd) GetDeviceConnectionState(device_id string) bool {
	var connection_state bool
	row := bdd.db_handler.QueryRow("SELECT is_connected FROM linked_devices WHERE device_id=?", device_id)

	err := row.Scan(&connection_state)
	if err != nil {
		log.Fatal("Error while querying database in GetDeviceConnectionState() : ", err)
	}

	return connection_state
}

// search in the list of secure_id if the given one is receiving updates from another device
func (bdd *AccesBdd) IsThisFileSystemBeingPatched() bool {

	var ids_str string
	var ids_list []string

	row := bdd.db_handler.QueryRow("SELECT receiving_update FROM linked_devices")

	err := row.Scan(&ids_str)
	if err != nil {
		log.Fatal("Error while querying database ", err)
	}

	ids_list = strings.Split(ids_str, ";")

	for _, id := range ids_list {
		if id == bdd.SecureId {
			return true
		}
	}

	return false

}

func (bdd *AccesBdd) SetFileSystemPatchLockState(device_id string, value bool) {

	// lock the filesystem (simply add the secure_id of the sync task to the list)
	if value {

		_, err := bdd.db_handler.Exec("UPDATE linked_devices SET receiving_update=IFNULL(receiving_update, '') || ?", bdd.SecureId+";")

		if err != nil {
			log.Fatal("Error while updating database in LinkDevice() : ", err)
		}

		// unlock the filesystem
	} else {
		var ids_str string
		var ids_list []string

		row := bdd.db_handler.QueryRow("SELECT receiving_update FROM linked_devices")

		err := row.Scan(&ids_str)
		if err != nil {
			log.Fatal("Error while querying database ", err)
		}

		ids_list = strings.Split(ids_str, ";")

		// same list of sync tasks secure_id but without this one
		var new_ids []string
		for _, id := range ids_list {
			if !(id == bdd.SecureId) {
				new_ids = append(new_ids, id)
			}
		}

		// rewrite the updated list
		_, err = bdd.db_handler.Exec("UPDATE linked_devices SET receiving_update= ?", strings.Join(new_ids, ";"))

		if err != nil {
			log.Fatal("Error while updating database in LinkDevice() : ", err)
		}

	}
}

func (bdd *AccesBdd) GetFileSizeFromBdd(path string) int64 {
	var size int64
	row := bdd.db_handler.QueryRow("SELECT size FROM filesystem WHERE path=? AND secure_id=?", path, bdd.SecureId)

	err := row.Scan(&size)
	if err != nil {
		log.Fatal("Error while querying database ", err)
	}

	return size
}

func (bdd *AccesBdd) GetSyncLinkedDevices() []string {
	var devices_str string
	var devices_list []string

	row := bdd.db_handler.QueryRow("SELECT linked_devices_id FROM sync WHERE secure_id=?", bdd.SecureId)

	err := row.Scan(&devices_str)
	if err != nil {
		log.Fatal("Error while querying database ", err)
	}

	devices_list = strings.Split(devices_str, ";")

	// remove the last slot (empty space) in the array
	// caused by the last semicolon of the string representation of the list
	devices_list = devices_list[:len(devices_list)-1]

	return devices_list
}

func (bdd *AccesBdd) GetFileDelta(version int64, path string) dtbin.Delta {

	var delta dtbin.Delta
	var json_data string

	row := bdd.db_handler.QueryRow("SELECT delta FROM delta WHERE path=? AND version_id=?", path, version)

	err := row.Scan(&json_data)
	if err != nil {
		log.Fatal("Error while querying database ", err)
	}

	err = json.Unmarshal([]byte(json_data), &delta)

	if err != nil {
		log.Fatal("Error while decoding json data for binary delta : ", err)
	}

	return delta
}

func (bdd *AccesBdd) AddFolderToRetard(path string) {

	// get lastest version of file and increment it
	new_version_id := bdd.GetFileLastVersionId(path) + 1

	// add a line in retard table with all devices linked and the version number

	MODTYPES := map[string]string{
		"creation": "c",
		"delete":   "d",
		"patch":    "p",
		"move":     "m",
	}
	linked_devices := bdd.GetOfflineDevices()

	_, err := bdd.db_handler.Exec("INSERT INTO retard (version_id,path,mod_type,devices_to_patch,type,secure_id) VALUES(?,?,?,?,\"folder\",?)", new_version_id, path, MODTYPES["creation"], strings.Join(linked_devices, ";"), bdd.SecureId)

	if err != nil {
		log.Fatal("Error while inserting new retard in AddFolderToRetard() : ", err)
	}

}

func (bdd *AccesBdd) IsDeviceLinked(device_id string) bool {
	rows, err := bdd.db_handler.Query("SELECT id FROM linked_devices WHERE device_id=?", device_id)

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

func (bdd *AccesBdd) IsMyDeviceIdGenerated() bool {
	rows, err := bdd.db_handler.Query("SELECT id FROM mesid")

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

func (bdd *AccesBdd) GenerateMyDeviceId() {

	// generating secure_id

	alphabet := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	s := make([]rune, 41)

	for i := range s {
		s[i] = alphabet[rand.Intn(len(alphabet))]
	}

	secure_id := string(s)

	// adding it to database
	bdd.db_handler.Exec("INSERT INTO mesid(device_id) VALUES(?)", secure_id)

}
func (bdd *AccesBdd) GetMyDeviceId() string {
	row := bdd.db_handler.QueryRow("SELECT device_id FROM mesid")

	if row.Err() != nil {
		log.Fatal("Error while querying database from IsMyDeviceIdGenerated() : ", row.Err())
	}

	var device_id string

	row.Scan(&device_id)

	return device_id
}

func (bdd *AccesBdd) GetOfflineDevices() []string {

	rows, err := bdd.db_handler.Query("SELECT device_id,is_connected FROM linked_devices")

	if err != nil {
		log.Fatal("Error while querying database from IsMyDeviceIdGenerated() : ", err)
	}
	defer rows.Close()
	var offline_devices []string

	for rows.Next() {
		var device LinkDevice
		rows.Scan(&device.SecureId, &device.IsConnected)
		if !device.IsConnected {
			offline_devices = append(offline_devices, device.SecureId)
		}
	}

	return offline_devices
}

func (bdd *AccesBdd) GetOnlineDevices() []string {

	rows, err := bdd.db_handler.Query("SELECT device_id,is_connected FROM linked_devices")

	if err != nil {
		log.Fatal("Error while querying database from IsMyDeviceIdGenerated() : ", err)
	}
	defer rows.Close()

	var online_devices []string

	for rows.Next() {
		var device LinkDevice
		rows.Scan(&device.SecureId, &device.IsConnected)

		if device.IsConnected {
			online_devices = append(online_devices, device.SecureId)
		}
	}

	return online_devices
}
func (bdd *AccesBdd) SetDeviceIP(device_id string, value string) {
	log.Println("vars : ", bdd.SecureId, device_id, value)
	_, err := bdd.db_handler.Exec("UPDATE linked_devices SET ip_addr=? WHERE secure_id=? AND device_id=?", value, bdd.SecureId, device_id)

	if err != nil {
		log.Fatal("Error while updating database in SetDeviceIP() : ", err)
	}
}

func (bdd *AccesBdd) GetDeviceIP(device_id string) string {
	var ip_addr string
	row := bdd.db_handler.QueryRow("SELECT ip_addr FROM linked_devices WHERE device_id=?", device_id)

	err := row.Scan(&ip_addr)
	if err != nil {
		log.Fatal("Error while querying database in GetDeviceIP() : ", err)
	}

	return ip_addr
}

func (bdd *AccesBdd) IncrementFileVersion(path string) {
	// get lastest version of file and increment it
	new_version_id := bdd.GetFileLastVersionId(path) + 1

	// update version number in db
	_, err := bdd.db_handler.Exec("UPDATE filesystem SET version_id=? WHERE path=? AND secure_id=?", new_version_id, path, bdd.SecureId)

	if err != nil {
		log.Fatal("Error while updating file version_id in IncrementFileVersion()")
	}
}

func (bdd *AccesBdd) UpdateCachedFile(path string) {
	// reads the current state of the given file and update it in the database

	file_content, err := os.ReadFile(path)

	if err != nil {
		log.Fatal("Error in UpdateCachedFile() while reading file to cache its content : ", err)
	}

	_, err = bdd.db_handler.Exec("UPDATE filesystem SET content=? WHERE path=? AND secure_id=?", file_content, path, bdd.SecureId)

	if err != nil {
		log.Fatal("Error in UpdateCachedFile() while caching file content into bdd : ", err)
	}
}

func (bdd *AccesBdd) ListSyncAllTasks() []SyncInfos {

	// used to get all sync task secure_id and root path listed
	// returns a custom type containing the two values as string

	rows, err := bdd.db_handler.Query("SELECT secure_id,root FROM sync")

	if err != nil {
		log.Fatal("Error while querying database from IsMyDeviceIdGenerated() : ", err)
	}
	defer rows.Close()

	var list []SyncInfos

	for rows.Next() {
		var info SyncInfos
		rows.Scan(&info.SecureId, &info.Path)
		list = append(list, info)
	}

	return list

}
