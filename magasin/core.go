package magasin

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"qsync/bdd"
	"runtime"
	"strconv"
	"text/template"
)

// open magasin in a new web browser tab
// open opens the specified URL in the default browser of the user.
func OpenUrlInWebBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// HelloHandler handles requests for the `/hello` resource
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./magasin/html/index.html")
}

func InstallAppHandler(w http.ResponseWriter, r *http.Request) {

	err := InstallApp(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func InstallGrapinHandler(w http.ResponseWriter, r *http.Request) {
	err := InstallGrapin(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
func InstalledAppsHandler(w http.ResponseWriter, r *http.Request) {

	html, err := os.ReadFile("./magasin/html/installed.html")

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

	app_id, err := strconv.Atoi(r.URL.Query().Get("AppId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	config := acces.GetAppConfig(app_id)

	cmd := exec.Command(config.BinPath)

	err = cmd.Run()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func DeleteAppHandler(w http.ResponseWriter, r *http.Request) {

	app_id, err := strconv.Atoi(r.URL.Query().Get("AppId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	config := acces.GetAppConfig(app_id)

	if config.Type == "toutenun" {
		err = UninstallToutenun(config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	acces.DeleteApp(app_id)

}

func TestFileHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./magasin/test_configs.json")
}
func StartServer() {

	addr := "127.0.0.1:8275"

	//create a new mux (router)
	//the mux calls different functions for
	//different resource paths
	mux := http.NewServeMux()

	mux.HandleFunc("/", HomeHandler)
	mux.HandleFunc("/InstallerAjout", InstallGrapinHandler)
	mux.HandleFunc("/InstallerToutEnUn", InstallAppHandler)
	mux.HandleFunc("/InstalledApps", InstalledAppsHandler)
	mux.HandleFunc("/LaunchApp", LaunchAppHandler)
	mux.HandleFunc("/DeleteApp", DeleteAppHandler)
	mux.HandleFunc("/test_configs.json", TestFileHandler)
	//start the web server using the mux as the root handler,
	//and report any errors that occur.
	//the ListenAndServe() function will block so
	//this program will continue to run until killed
	log.Printf("server is listening at %s...", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
