package webui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"qsync/bdd"
	"qsync/filesystem"
	"qsync/globals"
	"qsync/magasin"
	"qsync/networking"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/sqweek/dialog"
)

func StartWebUI() {
	router := mux.NewRouter()

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
	router.HandleFunc("/ask-file-path", askFilePath).Methods("GET")
	router.HandleFunc("/check-internet", checkInternetConnection).Methods("GET")

	http.Handle("/", router)

	fmt.Println("Server started at :8275")
	log.Fatal(http.ListenAndServe(":8275", nil))
}

func InstallAppHandler(w http.ResponseWriter, r *http.Request) {

	err := magasin.InstallApp(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func InstallGrapinHandler(w http.ResponseWriter, r *http.Request) {
	err := magasin.InstallGrapin(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
func InstalledAppsHandler(w http.ResponseWriter, r *http.Request) {

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

func LaunchAppHandler(w http.ResponseWriter, r *http.Request) {

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

func DeleteAppHandler(w http.ResponseWriter, r *http.Request) {

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

func TestFileHandler(w http.ResponseWriter, r *http.Request) {
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
		FilePath    string `json:"filepath"`
		DeviceIndex int    `json:"device"`
		IsFolder    bool   `json:"is_folder"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	devices := acces.GetNetworkMap()
	device := devices.Get(requestData.DeviceIndex)

	if requestData.IsFolder {
		filepath := "multilargage.tar"
		err := globals.TarFolder(requestData.FilePath, filepath)
		if err != nil {
			log.Fatal("Error while taring folder ", err)
		}
	}

	networking.SendLargageAerien(requestData.FilePath, device["ip_addr"], false)
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
