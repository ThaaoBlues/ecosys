/*
 * @file            webui/core.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2024-06-24 18:47:41
 * @lastModified    2024-09-02 22:13:54
 * Copyright ©Théo Mougnibas All rights reserved
 */

package webui

import (
	"ecosys/backend_api"
	"ecosys/bdd"
	"ecosys/filesystem"
	"ecosys/globals"
	"ecosys/magasin"
	"ecosys/networking"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	//router.HandleFunc("/start", startecosys).Methods("GET")
	router.HandleFunc("/create-task", createSyncTask).Methods("GET")
	router.HandleFunc("/link", linkDevice).Methods("POST")
	router.HandleFunc("/list-tasks", listTasks).Methods("GET")
	router.HandleFunc("/list-devices", listDevices).Methods("GET")
	router.HandleFunc("/toggle-largage", toggleLargageAerien).Methods("GET")
	router.HandleFunc("/send-largage", sendLargage).Methods("POST")
	router.HandleFunc("/send-text", sendText).Methods("POST")
	router.HandleFunc("/ask-file-path", askFilePath).Methods("GET")
	router.HandleFunc("/check-internet", checkInternetConnection).Methods("GET")
	router.HandleFunc("/open-largages-folder", openLargagesFolder).Methods("GET")
	router.HandleFunc("/remove-task", removeTask).Methods("GET")
	router.HandleFunc("/toggle-backup-mode", toggleBackupMode).Methods("GET")
	router.HandleFunc("/js/{file_name}", serveJsFile).Methods("GET")
	router.HandleFunc("/css/{file_name}", serveCssFile).Methods("GET")
	router.HandleFunc("/img/{file_name}", serveImgFile).Methods("GET")
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
		//broadcast <- []byte("[CHOOSELINKPATH]|" + context)

		data := backend_api.ShowConfirmationPrompt("Accept the link request ?")

		if data {
			data = backend_api.ShowConfirmationPrompt(context)

			if data {

				path, err := dialog.Directory().Title("Select Folder").Browse()
				if err != nil {
					fmt.Println("Folder selection cancelled.")
					return
				}
				backend_api.GiveInput("[CHOOSELINKPATH]", path)

				// to avoid an import cycle in networking, we have to put the start of filesystem watcher here
				// start watching so user don't have to restart Ecosys
				go filesystem.StartWatcher(path)
			}

		} else {
			backend_api.GiveInput("[CHOOSELINKPATH]", "[CANCELLED]")
		}

		// give back success message to front-end
		//broadcast <- []byte("success")

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	// air dropping something
	callbacks["[OTDL]"] = func(context string) {
		// send context throught websocket
		//broadcast <- []byte("[OTDL]|" + context)

		// wait user input from web gui

		//data := <-user_response

		data := backend_api.ShowConfirmationPrompt(context)
		backend_api.GiveInput("[OTDL]", strconv.FormatBool(data))

		// give back success message to front-end
		//broadcast <- []byte("success")

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	callbacks["[MOTDL]"] = func(context string) {
		// send context throught websocket

		//broadcast <- []byte("[OTDL]|" + context)

		// wait user input from web gui

		//data := <-user_response

		data := backend_api.ShowConfirmationPrompt(context)
		backend_api.GiveInput("[MOTDL]", strconv.FormatBool(data))

		// give back success message to front-end
		//broadcast <- []byte("success")

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	callbacks["[ALERT_USER]"] = func(context string) {
		// send context throught websocket

		//broadcast <- []byte("[ALERT_USER]|" + context)

		backend_api.ShowAlert(context)
		data := "prout"

		backend_api.GiveInput("[ALERT_USER]", string(data))

		// give back success message to front-end
		//broadcast <- []byte("success")

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	callbacks["[CHOOSE_APP_TO_LINK]"] = func(context string) {
		broadcast <- []byte("[CHOOSE_APP_TO_LINK]|" + context)

		data := <-user_response
		var response struct {
			Path       string
			SecureId   string
			IsApp      bool
			Name       string
			BackupMode bool
			Flag       string
		}
		err := json.Unmarshal(data, response)

		if err != nil {
			log.Fatal("Unable to unmarshall app to link response from websocket : ", err)
		}

		if response.Flag == "[APP_TO_LINK_CHOOSEN]" {
			backend_api.GiveInput("[CHOOSE_APP_TO_LINK]", response.SecureId)
		}

	}

	go backend_api.WaitEventLoop(callbacks)

	fmt.Println("Server started at :8275")
	log.Fatal(http.ListenAndServe(":8275", nil))

}

func broadcastMessagesLoop(conn *websocket.Conn) {

	for {
		msg := <-broadcast
		log.Println("Broadcasting websocket message to front : ", msg)
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

	json.NewEncoder(w).Encode(MenuResponse{Message: "success"})

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

	var config globals.MinGenConfig
	config = acces.GetAppConfig(app_id)
	acces.SecureId = config.SecureId
	log.Println(config.BinPath)
	cmd := exec.Command(config.BinPath)

	// set working directory at the sync root for the application
	// so it can recover the path easily
	cmd.Dir = acces.GetRootSyncPath()

	err := cmd.Run()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(MenuResponse{Message: "success"})

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

func startecosys(w http.ResponseWriter, r *http.Request) {
	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	tasks := acces.ListSyncAllTasks()
	for i := 0; i < tasks.Size(); i++ {
		filesystem.StartWatcher(tasks.Get(i).Path)
	}
	json.NewEncoder(w).Encode(MenuResponse{Message: "ecosys started"})
}

func createSyncTask(w http.ResponseWriter, r *http.Request) {

	path, err := dialog.Directory().Title("Select Folder").Browse()
	if err != nil {
		fmt.Println("Folder selection cancelled.")
		return
	}

	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	acces.CreateSync(path)

	go filesystem.StartWatcher(path)
	json.NewEncoder(w).Encode(MenuResponse{Message: "Sync task created"})
}

func linkDevice(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
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

	acces.LinkDevice(requestData.DeviceId, requestData.IpAddr)

	var event globals.QEvent
	event.Flag = "[LINK_DEVICE]"
	event.SecureId = acces.SecureId
	event.NewFilePath = ""

	if acces.IsApp(acces.SecureId) {
		event.FileType = "[APPLICATION]"
		event.VersionToPatch = acces.GetSyncCreationDate()
		event.FilePath = acces.GetAppConfig(requestData.SecureId).AppName
	} else {
		event.FileType = "[CLASSIC]"
	}

	var queue globals.GenArray[globals.QEvent]
	queue.Add(event)
	var device_ids globals.GenArray[string]
	device_ids.Add(requestData.DeviceId)

	networking.SendDeviceEventQueueOverNetwork(device_ids, acces.SecureId, queue, requestData.IpAddr)

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
		FilePath string `json:"filepath"`
		IpAddr   string `json:"ip_addr"`
		IsFolder bool   `json:"is_folder"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println(requestData)

	if requestData.IsFolder {
		tarfile_path := filepath.Join(globals.EcosysWriteableDirectory, "multilargage.tar")

		err := globals.TarFolder(requestData.FilePath, tarfile_path)
		if err != nil {
			log.Fatal("Error while taring folder ", err)
		}
		networking.SendLargageAerien(tarfile_path, requestData.IpAddr, requestData.IsFolder)
	} else {
		networking.SendLargageAerien(requestData.FilePath, requestData.IpAddr, requestData.IsFolder)

	}

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

	f_path := filepath.Join(globals.EcosysWriteableDirectory, globals.GetCurrentTimestampString()+"_text.txt")

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
	open.Run(filepath.Join(globals.EcosysWriteableDirectory, "largage_aerien"))
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
	path := filepath.Join(globals.EcosysWriteableDirectory, "webui/js/"+mux.Vars(r)["file_name"])

	http.ServeFile(w, r, path)
}
func serveCssFile(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(globals.EcosysWriteableDirectory, "webui/css/"+mux.Vars(r)["file_name"])

	http.ServeFile(w, r, path)
}
func serveImgFile(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(globals.EcosysWriteableDirectory, "webui/img/"+mux.Vars(r)["file_name"])

	http.ServeFile(w, r, path)
}
