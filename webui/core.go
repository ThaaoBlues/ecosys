/*
 * @file            webui/core.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2024-06-24 18:47:41
 * @lastModified    2024-06-27 17:24:42
 * Copyright ©Théo Mougnibas All rights reserved
 */

package webui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"qsync/backend_api"
	"qsync/bdd"
	"qsync/filesystem"
	"qsync/globals"
	"qsync/magasin"
	"qsync/networking"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/skratchdot/open-golang/open"
	"github.com/sqweek/dialog"
)

var PROCESSING_EVENT bool
var CURRENT_EVENT_FLAG string

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var broadcast = make(chan []byte)
var user_response = make(chan []byte)

func StartWebUI() {
	router := mux.NewRouter()
	upgrader.CheckOrigin = func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://127.0.0.1:8275"
	}

	router.HandleFunc("/", serveIndex).Methods("GET")
	router.HandleFunc("/magasin", magasinHandler).Methods("GET")
	router.HandleFunc("/start", startQSync).Methods("GET")
	router.HandleFunc("/create", createSyncTask).Methods("GET")
	router.HandleFunc("/link", linkDevice).Methods("POST")
	router.HandleFunc("/list-tasks", listTasks).Methods("GET")
	router.HandleFunc("/list-devices", listDevices).Methods("GET")
	router.HandleFunc("/open-magasin", openMagasin).Methods("GET")
	router.HandleFunc("/toggle-largage", toggleLargageAerien).Methods("GET")
	router.HandleFunc("/send-largage", sendLargage).Methods("POST")
	router.HandleFunc("/send-text", sendText).Methods("POST")
	router.HandleFunc("/ask-file-path", askFilePath).Methods("GET")
	router.HandleFunc("/check-internet", checkInternetConnection).Methods("GET")
	router.HandleFunc("/open-largages-folder", openLargagesFolder).Methods("GET")
	router.HandleFunc("/remove-task", removeTask).Methods("GET")
	router.HandleFunc("/toggle-backup-mode", toggleBackupMode).Methods("GET")
	router.HandleFunc("/js/translations.js", serveJsFile).Methods("GET")
	router.HandleFunc("/install-tout-en-un", installAppHandler).Methods("POST")
	router.HandleFunc("/install-grapin", installGrapinHandler).Methods("POST")
	router.HandleFunc("/launch-app", launchAppHandler).Methods("GET")
	router.HandleFunc("/delete-app", deleteAppHandler).Methods("GET")
	router.HandleFunc("/test_configs.json", testFileHandler).Methods("GET")

	router.HandleFunc("/ws", websocketMsgHandler)
	http.Handle("/", router)

	// interactive events callbacks
	callbacks := make(map[string]func(string))

	callbacks["[CHOOSELINKPATH]"] = func(context string) {
		// send context throught websocket
		broadcast <- []byte("[CHOOSELINKPATH]|" + context)

		path, err := dialog.Directory().Title("Select Folder").Browse()
		if err != nil {
			fmt.Println("Folder selection cancelled.")
			return
		}
		backend_api.GiveInput("[CHOOSELINKPATH]", path)

		// give back success message to front-end
		broadcast <- []byte("success")

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	// air dropping something
	callbacks["[OTDL]"] = func(context string) {
		// send context throught websocket
		broadcast <- []byte("[OTDL]|" + context)

		// wait user input from web gui

		data := <-user_response

		backend_api.GiveInput("[OTDL]", string(data))

		// give back success message to front-end
		broadcast <- []byte("success")

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	callbacks["[MOTDL]"] = func(context string) {
		// send context throught websocket

		broadcast <- []byte("[OTDL]|" + context)

		// wait user input from web gui

		data := <-user_response

		backend_api.GiveInput("[MOTDL]", string(data))

		// give back success message to front-end
		broadcast <- []byte("success")

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	callbacks["[ALERT_USER]"] = func(context string) {
		// send context throught websocket

		broadcast <- []byte("[ALERT_USER]|" + context)

		// wait user input from web gui

		data := "prout"

		backend_api.GiveInput("[ALERT_USER]", string(data))

		// give back success message to front-end
		broadcast <- []byte("success")

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	go backend_api.WaitEventLoop(callbacks)

	fmt.Println("Server started at :8275")
	log.Fatal(http.ListenAndServe(":8275", nil))

}

func broadcastMessagesLoop(conn *websocket.Conn) {

	for {
		msg := <-broadcast
		conn.WriteMessage(websocket.TextMessage, msg)
	}

}

func websocketMsgHandler(w http.ResponseWriter, r *http.Request) {

	// always used to handle user response to an event
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	go broadcastMessagesLoop(conn)

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("Received: %s", p)

		// give response to event callback function
		user_response <- p
	}

}

func installAppHandler(w http.ResponseWriter, r *http.Request) {

	err := magasin.InstallApp(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func installGrapinHandler(w http.ResponseWriter, r *http.Request) {
	err := magasin.InstallGrapin(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
func installedAppsHandler(w http.ResponseWriter, r *http.Request) {

	html, err := os.ReadFile("./webui/html/installed.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	// Parse the HTML template
	tmpl := template.Must(template.New("index").Parse(string(html)))

	// Execute the template with the data
	if err := tmpl.Execute(w, acces.ListInstalledApps()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func launchAppHandler(w http.ResponseWriter, r *http.Request) {

	app_id := r.URL.Query().Get("AppId")

	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	config := acces.GetAppConfig(app_id)

	cmd := exec.Command(config.BinPath)

	err := cmd.Run()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func deleteAppHandler(w http.ResponseWriter, r *http.Request) {

	app_id := r.URL.Query().Get("AppId")

	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	config := acces.GetAppConfig(app_id)

	if config.Type == "toutenun" {
		err := magasin.UninstallToutenun(config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	acces.DeleteApp(app_id)

}

func testFileHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./webui/test_configs.json")
}

func magasinHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./webui/html/magasin.html")
}

type MenuResponse struct {
	Message string `json:"message"`
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./webui/html/index.html")
}

func startQSync(w http.ResponseWriter, r *http.Request) {
	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	tasks := acces.ListSyncAllTasks()
	for i := 0; i < tasks.Size(); i++ {
		filesystem.StartWatcher(tasks.Get(i).Path)
	}
	json.NewEncoder(w).Encode(MenuResponse{Message: "QSync started"})
}

func createSyncTask(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	acces.CreateSync(path)
	json.NewEncoder(w).Encode(MenuResponse{Message: "Sync task created"})
}

func linkDevice(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		Path     string `json:"Path"`
		SecureId string `json:"SecureId"`
		DeviceId string `json:"DeviceId"`
		IpAddr   string `json:"IpAddr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println(requestData)

	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	acces.SecureId = requestData.SecureId

	var event globals.QEvent
	event.Flag = "[LINK_DEVICE]"
	event.SecureId = acces.SecureId
	event.FilePath = ""

	var queue globals.GenArray[globals.QEvent]
	queue.Add(event)
	var device_ids globals.GenArray[string]
	device_ids.Add(requestData.DeviceId)

	networking.SendDeviceEventQueueOverNetwork(device_ids, acces.SecureId, queue, requestData.IpAddr)
	acces.LinkDevice(requestData.SecureId, requestData.IpAddr)

	json.NewEncoder(w).Encode(MenuResponse{Message: "Device linked"})
}

func listTasks(w http.ResponseWriter, r *http.Request) {
	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	tasks := acces.ListSyncAllTasks()
	json.NewEncoder(w).Encode(tasks.ToSlice())
}

func listDevices(w http.ResponseWriter, r *http.Request) {
	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	devices := acces.GetNetworkMap()

	json.NewEncoder(w).Encode(devices.ToSlice())
}

func openMagasin(w http.ResponseWriter, r *http.Request) {
	globals.OpenUrlInWebBrowser("http://127.0.0.1:8275/magasin")
	json.NewEncoder(w).Encode(MenuResponse{Message: "Magasin opened"})
}

func toggleLargageAerien(w http.ResponseWriter, r *http.Request) {
	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	acces.SwitchLargageAerienAllowingState()
	isAllowed := acces.AreLargageAerienAllowed()
	if isAllowed {
		json.NewEncoder(w).Encode(MenuResponse{Message: "Largage Aerien allowed"})
	} else {
		json.NewEncoder(w).Encode(MenuResponse{Message: "Largage Aerien disallowed"})
	}
}

func sendLargage(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		FilePath string            `json:"filepath"`
		Device   map[string]string `json:"device"`
		IsFolder bool              `json:"is_folder"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println(requestData)
	tarfile_path := filepath.Join(globals.QSyncWriteableDirectory, "multilargage.tar")

	if requestData.IsFolder {
		err := globals.TarFolder(requestData.FilePath, tarfile_path)
		if err != nil {
			log.Fatal("Error while taring folder ", err)
		}
	}

	networking.SendLargageAerien(tarfile_path, requestData.Device["ip_addr"], requestData.IsFolder)
	json.NewEncoder(w).Encode(MenuResponse{Message: "File sent"})

}

func sendText(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		Device map[string]string `json:"device"`
		Text   string            `json:"text"`
	}

	log.Println(requestData)

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f_path := filepath.Join(globals.QSyncWriteableDirectory, "text.txt")

	f, err := os.OpenFile(f_path, os.O_CREATE|os.O_RDWR, 0750)

	if err != nil {
		log.Fatal("Error while opening temporary text largage file", err)
	}

	f.WriteString(requestData.Text)
	f.Close()

	networking.SendLargageAerien(
		f_path,
		requestData.Device["ip_addr"],
		false,
	)
	log.Println(f_path)

	// remove the temporary file some time after
	go func() {
		time.Sleep(10 * time.Second)
		os.Remove(f_path)
	}()
	json.NewEncoder(w).Encode(MenuResponse{Message: "File sent"})

}

func askFilePath(w http.ResponseWriter, r *http.Request) {
	var pathResponse struct {
		Path string
	}

	is_folder := r.URL.Query().Get("is_folder") == "true"
	var err error
	if is_folder {
		pathResponse.Path, err = dialog.Directory().Title("Select Folder").Browse()

	} else {
		pathResponse.Path, err = dialog.File().Title("Select Folder").Load()
	}

	if err != nil {
		log.Println("File selection cancelled.")
		pathResponse.Path = "[CANCELLED]"
	}

	json.NewEncoder(w).Encode(pathResponse)
}

func checkInternetConnection(w http.ResponseWriter, r *http.Request) {
	var boolResponse struct {
		ConnectionState bool
	}

	boolResponse.ConnectionState = networking.IsNetworkAvailable()

	json.NewEncoder(w).Encode(boolResponse)

}

func openLargagesFolder(w http.ResponseWriter, r *http.Request) {
	open.Run(filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien"))
	json.NewEncoder(w).Encode(MenuResponse{Message: "success"})
}

func removeTask(w http.ResponseWriter, r *http.Request) {
	secure_id := r.URL.Query().Get("secure_id")
	var acces bdd.AccesBdd
	acces.InitConnection()
	acces.SecureId = secure_id
	acces.RmSync()
	json.NewEncoder(w).Encode(MenuResponse{Message: "success"})

}
func toggleBackupMode(w http.ResponseWriter, r *http.Request) {
	secure_id := r.URL.Query().Get("secure_id")
	var acces bdd.AccesBdd
	acces.InitConnection()
	acces.SecureId = secure_id
	acces.ToggleBackupMode()
	json.NewEncoder(w).Encode(MenuResponse{Message: "success"})
}

func serveJsFile(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(globals.QSyncWriteableDirectory, "webui/js/translations.js")

	http.ServeFile(w, r, path)
}
