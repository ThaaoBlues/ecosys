package magasin

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
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
func HelloHandler(w http.ResponseWriter, r *http.Request) {
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

func StartServer() {
	//get the value of the ADDR environment variable
	addr := os.Getenv("ADDR")

	//if it's blank, default to ":80", which means
	//listen port 80 for requests addressed to any host
	if len(addr) == 0 {
		addr = "127.0.0.1:8275"
	}

	//create a new mux (router)
	//the mux calls different functions for
	//different resource paths
	mux := http.NewServeMux()

	//tell it to call the HelloHandler() function
	//when someone requests the resource path `/hello`
	mux.HandleFunc("/", HelloHandler)
	mux.HandleFunc("/InstallerAjout", InstallGrapinHandler)
	mux.HandleFunc("/InstallerToutEnUn", InstallAppHandler)

	//start the web server using the mux as the root handler,
	//and report any errors that occur.
	//the ListenAndServe() function will block so
	//this program will continue to run until killed
	log.Printf("server is listening at %s...", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
