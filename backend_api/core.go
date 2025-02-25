/*
 * @file            backend_api/core.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2024-03-02 19:14:18
 * @lastModified    2024-07-25 22:21:17
 * Copyright ©Théo Mougnibas All rights reserved
 */

package backend_api

import (
	"ecosys/globals"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/ncruces/zenity"
)

func NotifyDesktop(msg string) {
	err := beeep.Alert("ecosys", msg, "assets/warning.png")
	if err != nil {
		panic(err)
	}
}

// IF THE BACKEND IS MULTITHREADED, DO NOT CLOSE APP BEFORE THE USER INPUT HAS BEEN
// PROCESSED BY BACKEND, THIS FUNCTION DOES NOT MAKES SURE OF IT
func AskInput(flag string, context string) string {

	f, err := os.Create(filepath.Join(globals.EcosysWriteableDirectory, flag+".btf"))
	defer os.Remove(filepath.Join(globals.EcosysWriteableDirectory, flag+".btf"))

	if err != nil {
		log.Fatal("Unable to Create input file in AskInput() : ", err)
	}

	f.WriteString(context)

	og_fstat, err_2 := os.Stat(filepath.Join(globals.EcosysWriteableDirectory, flag+".btf"))

	if err_2 != nil {
		log.Fatal("Unable to read stats of input file in AskInput() : ", err)
	}

	// wait for front-end to provide user input
	var nw_fstat os.FileInfo
	nw_fstat, err_2 = os.Stat(filepath.Join(globals.EcosysWriteableDirectory, flag+".btf"))

	if err_2 != nil {
		log.Fatal("Unable to read stats of input file in AskInput() : ", err)
	}

	for nw_fstat.Size() == og_fstat.Size() {
		time.Sleep(2 * time.Second)

		nw_fstat, err_2 = os.Stat(filepath.Join(globals.EcosysWriteableDirectory, flag+".btf"))

		if err_2 != nil {
			log.Fatal("Unable to read stats of input file in AskInput() : ", err)
		}

	}

	// now that we have the user input in the file, we can read it

	ret, err := os.ReadFile(filepath.Join(globals.EcosysWriteableDirectory, flag+".btf"))
	if err != nil {
		log.Fatal("Unable to Read input file in AskInput() : ", err)
	}
	return string(ret[len([]byte(context)):])
}

// use this function to get the message from the backend
// that is riding with the ask of user input
// must be used before providing the user's input
func ReadInputContext(flag string) string {

	buff, err := os.ReadFile(filepath.Join(globals.EcosysWriteableDirectory, flag+".btf"))
	if err != nil {
		log.Fatal("Unable to Read input file in ReadInputContext() : ", err)
	}

	return string(buff)
}

func GiveInput(flag string, data string) {
	f, err := os.OpenFile(filepath.Join(globals.EcosysWriteableDirectory, flag+".btf"), os.O_RDWR|os.O_APPEND, os.ModeAppend)
	if err != nil {
		log.Fatal("Unable to Create input file in AskInput() : ", err)
	}

	defer f.Close()

	f.WriteString(data)

}

func WaitEventLoop(callbacks map[string]func(context string)) {

	for {
		// Read the contents of the root directory
		files, err := os.ReadDir(globals.EcosysWriteableDirectory)
		if err != nil {
			log.Fatal("Error while reading directory in WaitEventLoop() : ", err)
		}

		// Check each file to see if it has a .btf extension
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if file.Name()[len(file.Name())-4:] == ".btf" {

				event_flag := file.Name()[:len(file.Name())-4]
				context_buff, err := os.ReadFile(file.Name())
				if err != nil {
					log.Fatal("Error while reading event fie in WaitEventLoop() : ", err)
				}

				callbacks[event_flag](string(context_buff))

			}
		}

		// Sleep for 1 second before checking again
		time.Sleep(1 * time.Second)
	}

}

// showConfirmationPrompt displays a native confirmation prompt with the specified message.
// It returns true if the user confirms, and false otherwise.
func ShowConfirmationPrompt(message string) bool {
	err := zenity.Question(
		message,
		zenity.Title("Confirmation"),
		zenity.CancelLabel("No"),
	)

	return err == nil
}

func ShowAlert(message string) {
	_ = zenity.Warning(message, zenity.Title("Information"))
}

func RunDPKGAsRoot(deb_path string) {
	// Use zenity to show a graphical prompt for sudo
	cmd := exec.Command("zenity", "--password", "--title=Authentication Required", "--text=ecosys needs to be root to run dpkg and install your app.")
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Get the password from the zenity prompt
	passwordBytes, err := cmd.Output()
	if err != nil {
		fmt.Println("Failed to get password:", err)
		os.Exit(1)
	}

	// Convert password bytes to string
	password := string(passwordBytes)

	// Use the password to run the command with sudo
	log.Println("Running sudo command")
	sudoCmd := exec.Command("sudo", "-S", "dpkg", "-i", deb_path)
	sudoCmd.Stderr = os.Stderr
	sudoCmd.Stdout = os.Stdout

	// Create a pipe to send the password to sudo's stdin
	stdin, err := sudoCmd.StdinPipe()
	log.Println("Providing password")
	if err != nil {
		log.Fatal("Failed to create stdin pipe:", err)
		//os.Exit(1)
	}

	// Start the sudo command
	if err := sudoCmd.Start(); err != nil {
		log.Fatal("Failed to start sudo:", err)
		//os.Exit(1)
	}

	// Write the password to sudo's stdin
	if _, err := stdin.Write([]byte(password)); err != nil {
		log.Fatal("Failed to write password to stdin:", err)
		//os.Exit(1)
	}
	stdin.Close()

	// Wait for the sudo command to complete
	if err := sudoCmd.Wait(); err != nil {
		log.Fatal("Failed to run with sudo:", err)
		//os.Exit(1)
	}

}

/*
returns (string) the absolute path of the selected file/directory,
the "[CANCELLED]" flag is returned if anything happened
like the user closing the window, an error etc...
*/
func AskFilePath() string {
	var ret string

	file, err := zenity.SelectFile(zenity.Title("Select a file !"))

	if err != nil {
		log.Println("File selection cancelled.")
		ret = "[CANCELLED]"
	} else {
		ret = file
	}

	return ret
}

/*
returns (string) the absolute path of the selected file/directory,
the "[CANCELLED]" flag is returned if anything happened
like the user closing the window, an error etc...
*/
func AskDirectoryPath() string {

	var ret string

	dir, err := zenity.SelectFile(
		zenity.Directory(),
		zenity.Title("Select a directory !"),
	)

	if err != nil {
		log.Println("directory selection cancelled.")
		ret = "[CANCELLED"
	} else {
		ret = dir
	}

	return ret
}

/*
returns ([]string) the absolute path of the selected files/directories,
the "[CANCELLED]" flag is returned if anything happened
like the user closing the window, an error etc...
*/
func AskMultipleFilePath() []string {
	var ret []string

	files, err := zenity.SelectFileMultiple(
		zenity.Directory(),
		zenity.Title("Select multiple files !"),
	)

	if err != nil {
		log.Println("directory selection cancelled.")
		ret = make([]string, 1)
		ret[0] = "[CANCELLED]"
	} else {
		ret = files
	}

	return ret
}

/*
returns ([]string) the absolute path of the selected files/directories,
the "[CANCELLED]" flag is returned if anything happened
like the user closing the window, an error etc...
*/
func AskMultipleDirectoryPath() []string {
	var ret []string

	dirs, err := zenity.SelectFileMultiple(
		zenity.Directory(),
		zenity.Title("Select multiple files !"),
	)

	if err != nil {
		log.Println("directory selection cancelled.")
		ret = make([]string, 1)
		ret[0] = "[CANCELLED]"
	} else {
		ret = dirs
	}

	return ret
}
