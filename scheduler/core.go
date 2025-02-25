package scheduler

import (
	"log"
	"os"
)

// this packages aims to provide an interface to the web ui
// so one can schedule a job on a computer and get a notification on another device
// when the job is done ( optionnaly with output with )

func RegisterJob(command string, ip_addr string) {
	f, err := os.OpenFile("jobs.db", os.O_CREATE, 0755)

	if err != nil {
		log.Fatal("Une erreur est survenue dans la fonction RegisterJob() en tentant d'ouvrir la base de données des jobs.")
	}

	f.Write([]byte(ip_addr + "\n"))
	f.Close()
}
func StartJob(command string) {
	//os.StartProcess()
}

func SendJobResuls() {

}
