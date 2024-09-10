package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var currentLang = "en" // Default language
var translations = map[string]map[string]string{
	"en": {
		"devicesTitle":      "Devices on your network",
		"tasksTitle":        "Active tasks on your device",
		"taskActions":       "Task Actions",
		"deviceActions":     "Device Actions",
		"sendFile":          "Send File",
		"sendFolder":        "Send Folder",
		"sendText":          "Send Text",
		"removeTask":        "Remove Task",
		"openApp":           "Open App",
		"syncAnotherDevice": "Sync Another Device",
		"enableBackupMode":  "Enable Backup Mode",
		"disableBackupMode": "Disable Backup Mode",
		"alertTaskCreated":  "Task Created at ",
	},
	"fr": {
		"devicesTitle":      "Appareils sur votre réseau",
		"tasksTitle":        "Tâches actives sur votre appareil",
		"taskActions":       "Actions de tâche",
		"deviceActions":     "Actions de l'appareil",
		"sendFile":          "Envoyer un fichier",
		"sendFolder":        "Envoyer un dossier",
		"sendText":          "Envoyer un texte",
		"removeTask":        "Supprimer la tâche",
		"openApp":           "Ouvrir l'application",
		"syncAnotherDevice": "Synchroniser un autre appareil",
		"enableBackupMode":  "Activer le mode de sauvegarde",
		"disableBackupMode": "Désactiver le mode de sauvegarde",
		"alertTaskCreated":  "Tâche créée à ",
	},
}

type Config struct {
	AppName            string   `json:"AppName"`
	AppDescription     string   `json:"AppDescription"`
	AppIconURL         string   `json:"AppIconURL"`
	SupportedPlatforms []string `json:"SupportedPlatforms"`
}

type Data struct {
	ToutEnUnConfigs []Config `json:"tout_en_un_configs"`
	GrapinConfigs   []Config `json:"grapin_configs"`
}

func updateLanguage(lang string) {
	currentLang = lang
}

func fetchTasks() []map[string]string {
	resp, err := http.Get("http://127.0.0.1:8275/list-tasks")
	if err != nil {
		log.Println("Error fetching tasks:", err)
		return []map[string]string{}
	}
	defer resp.Body.Close()

	var tasks []map[string]string
	json.NewDecoder(resp.Body).Decode(&tasks)
	return tasks
}

func fetchDevices() []map[string]string {
	resp, err := http.Get("http://127.0.0.1:8275/list-devices")
	if err != nil {
		log.Println("Error fetching devices:", err)
		return []map[string]string{}
	}
	defer resp.Body.Close()

	var devices []map[string]string
	json.NewDecoder(resp.Body).Decode(&devices)
	return devices
}

func CreateUI(app *tview.Application) tview.Primitive {

	updateLanguage("fr")

	// Header and Titles
	header := tview.NewTextView().
		SetText("Ecosys Terminal GUI").
		SetTextColor(tcell.ColorGreen).
		SetTextAlign(tview.AlignCenter)

	devicesTitle := tview.NewTextView().
		SetText(translations[currentLang]["devicesTitle"]).
		SetTextColor(tcell.ColorYellow).
		SetDynamicColors(true)

	tasksTitle := tview.NewTextView().
		SetText(translations[currentLang]["tasksTitle"]).
		SetTextColor(tcell.ColorYellow).
		SetDynamicColors(true)

	// Menu buttons (navigable)
	createTaskBtn := tview.NewButton("Create a Sync Task").SetSelectedFunc(func() {
		createSyncTask()
	})
	openMagasinBtn := tview.NewButton("Open Magasin").SetSelectedFunc(func() {
		openMagasin(app)
	})
	toggleLargageBtn := tview.NewButton("Toggle Largage Aerien").SetSelectedFunc(func() {
		toggleLargageAerien()
	})
	openLargagesFolderBtn := tview.NewButton("Open Largage Aerien Folder").SetSelectedFunc(func() {
		openLargagesFolder()
	})

	// Button list
	buttons := []*tview.Button{createTaskBtn, openMagasinBtn, toggleLargageBtn, openLargagesFolderBtn}
	menu := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(createTaskBtn, 2, 1, true).
		AddItem(openMagasinBtn, 2, 1, true).
		AddItem(toggleLargageBtn, 2, 1, true).
		AddItem(openLargagesFolderBtn, 2, 1, true)

	// Device List
	devicesList := tview.NewList()

	// Task List
	tasksList := tview.NewList()

	// Set up focus navigation between buttons
	for i, btn := range buttons {
		buttonIndex := i
		btn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyDown:
				app.SetFocus(buttons[(buttonIndex+1)%len(buttons)])
				return nil
			case tcell.KeyUp:
				app.SetFocus(buttons[(buttonIndex+len(buttons)-1)%len(buttons)])
				return nil
			}
			return event
		})
	}

	// Layout for devices and tasks
	devicesListLayout := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(devicesTitle, 1, 1, false).
			AddItem(devicesList, 0, 1, true), 0, 1, true)

	tasksListLayout := tview.NewFlex().AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tasksTitle, 1, 1, false).
		AddItem(tasksList, 0, 1, true), 0, 1, true)

	terminalParts := []*tview.Flex{menu, devicesListLayout, tasksListLayout}

	// Main layout
	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 3, 1, false)

	// Add keyboard navigation between menu and content
	for i, layout := range terminalParts {
		layoutIndex := i
		layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			log.Println("Setting focus to another terminal part")

			switch event.Key() {
			case tcell.KeyTab:
				app.SetFocus(terminalParts[(layoutIndex+1)%len(terminalParts)])
				// remove highlight on previous focus zone
				terminalParts[layoutIndex].
					SetBorder(false)

				// highlight focus zone
				terminalParts[(layoutIndex+1)%len(terminalParts)].
					SetBorder(true).
					SetBorderStyle(tcell.StyleDefault).
					SetBorderColor(tcell.ColorGhostWhite)
			}
			return event
		})

		mainLayout = mainLayout.AddItem(layout, 0, 1, true)
	}

	app.SetFocus(menu)

	// filling up lists

	go func() {
		for {
			app.QueueUpdateDraw(func() {
				devicesList.Clear()
				for _, device := range fetchDevices() {
					devicesList.AddItem(device["hostname"], device["ip_addr"], 0, func() {
						openDeviceActionsMenu(app, device, mainLayout)
					})
				}
			})
			time.Sleep(5 * time.Second)
		}
	}()

	go func() {
		for {
			app.QueueUpdateDraw(func() {
				tasksList.Clear()
				for _, task := range fetchTasks() {
					label := task["Path"]
					if task["IsApp"] == "true" {
						label = "( application ) " + task["Name"]
					}
					tasksList.AddItem(label, "", 0, func() {
						openTaskActionsMenu(app, task, mainLayout)
					})
				}
			})
			time.Sleep(5 * time.Second)
		}
	}()

	return mainLayout
}

// Popup menus for task and device actions
func openTaskActionsMenu(app *tview.Application, task map[string]string, appRoot *tview.Flex) {
	var backupModeText string
	if task["BackupMode"] == "true" {
		backupModeText = translations[currentLang]["disableBackupMode"]
	} else {
		backupModeText = translations[currentLang]["enableBackupMode"]

	}

	modal := tview.NewModal().
		SetText(translations[currentLang]["taskActions"]).
		AddButtons([]string{
			translations[currentLang]["openApp"],
			translations[currentLang]["syncAnotherDevice"],
			translations[currentLang]["removeTask"],
			backupModeText,
		}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case translations[currentLang]["openApp"]:
				openApp(task)
			case translations[currentLang]["syncAnotherDevice"]:
				chooseDeviceAndLinkIt(app, task, appRoot)
			case translations[currentLang]["removeTask"]:
				removeTask(task)
			case translations[currentLang]["enableBackupMode"], translations[currentLang]["disableBackupMode"]:
				toggleBackupMode(task)
			}
			app.SetRoot(CreateUI(app), true)
		})
	app.SetRoot(modal, true).SetFocus(modal)
}

func openDeviceActionsMenu(app *tview.Application, device map[string]string, appRoot *tview.Flex) {
	modal := tview.NewModal().
		SetText(translations[currentLang]["deviceActions"]).
		AddButtons([]string{
			translations[currentLang]["sendFile"],
			translations[currentLang]["sendFolder"],
			translations[currentLang]["sendText"],
		}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case translations[currentLang]["sendFile"]:
				sendLargage(device, false)
				app.SetRoot(appRoot, true)

			case translations[currentLang]["sendFolder"]:
				sendLargage(device, true)
				app.SetRoot(appRoot, true)
			case translations[currentLang]["sendText"]:
				log.Println("IN SEND TEXT")
				sendText(app, device, appRoot)
				// not setting root as sendText needs another form
			}
		})
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		log.Println("Setting focus to another terminal part")

		switch event.Key() {
		case tcell.KeyEscape:
			log.Println("Hiding modal")
			app.SetRoot(appRoot, false)
		}
		return event
	})

	app.SetRoot(modal, true).SetFocus(modal)
}

func createSyncTask() {
	_, err := http.Get("http://127.0.0.1:8275/create-task")
	if err != nil {
		fmt.Println("Error creating sync task:", err)
	}
}

func openMagasin(app *tview.Application) {
	app.SetRoot(prepareMagasin(app), false)
}

func toggleLargageAerien() {
	_, err := http.Get("http://127.0.0.1:8275/toggle-largage")
	if err != nil {
		fmt.Println("Error toggling Largage Aerien:", err)
	}
}

func openLargagesFolder() {
	_, err := http.Get("http://127.0.0.1:8275/open-largages-folder")
	if err != nil {
		fmt.Println("Error opening Largages Folder:", err)
	}
}

func sendLargage(device map[string]string, folder bool) {

	// Request the file path from the user
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:8275/ask-file-path?is_folder=%t", folder))
	if err != nil {
		log.Println("Error fetching file path:", err)
		return
	}
	defer resp.Body.Close()

	var response map[string]string
	json.NewDecoder(resp.Body).Decode(&response)

	if response["Path"] != "[CANCELLED]" {

		data := map[string]interface{}{
			"filepath":  response["Path"],
			"device_id": device["device_id"],
			"ip_addr":   device["ip_addr"],
			"is_folder": folder,
		}

		log.Println("Sending request to trigger largage aerien.", data)

		// Send the largage
		jsonData, _ := json.Marshal(data)
		resp, err := http.Post("http://127.0.0.1:8275/send-largage", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Println("Error sending largage:", err)
		}

		//json.NewDecoder(resp.Body).Decode(&response)
		log.Println(resp.Body)
		defer resp.Body.Close()
	}
}

func sendText(app *tview.Application, device map[string]string, appRoot *tview.Flex) {
	form := tview.NewForm().
		AddTextArea("Text", "", 0, 20, 3000, nil)

	form.
		AddButton("Send", func() {
			text := form.GetFormItemByLabel("Text").(*tview.TextArea).GetText()
			// Send text
			data := map[string]interface{}{
				"device": device,
				"text":   text,
			}
			jsonData, _ := json.Marshal(data)
			resp, err := http.Post("http://127.0.0.1:8275/send-text", "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Println("Error sending text:", err)
			}
			defer resp.Body.Close()

			// Return to main layout after sending
			app.SetRoot(appRoot, true)
		}).
		AddButton("Cancel", func() {
			app.SetRoot(appRoot, true)
		})

	app.SetRoot(form, true).SetFocus(form)

}

func chooseDeviceAndLinkIt(app *tview.Application, task map[string]string, appRoot *tview.Flex) {
	// Fetch devices to display
	resp, err := http.Get("http://127.0.0.1:8275/list-devices")
	if err != nil {
		log.Println("Error fetching devices:", err)
		return
	}
	defer resp.Body.Close()

	var devices []map[string]string
	json.NewDecoder(resp.Body).Decode(&devices)

	list := tview.NewList()
	for _, device := range devices {
		d := device // Capture loop variable
		list.AddItem(device["hostname"], "", 0, func() {
			// Link the device with the task
			linkDevice(task, d)
			app.SetRoot(appRoot, true)
		})
	}

	// Set up modal for choosing device
	modal := tview.NewModal().
		SetText("Choose a device to synchronize").
		AddButtons(
			[]string{"Cancel"},
		).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(appRoot, true)
		})

	// Create a layout with the device list and the modal
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(list, 0, 1, true).
		AddItem(modal, 3, 1, false)

	app.SetRoot(layout, true).SetFocus(list)
}

func linkDevice(task map[string]string, device map[string]string) {
	data := map[string]string{
		"SecureId": task["SecureId"],
		"IpAddr":   device["ip_addr"],
		"DeviceId": device["device_id"],
	}

	// Send the request to link the device
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post("http://127.0.0.1:8275/link", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error linking device:", err)
	}
	defer resp.Body.Close()
}

func removeTask(task map[string]string) {
	_, err := http.Get(fmt.Sprintf("http://127.0.0.1:8275/remove-task?secure_id=%s", task["SecureId"]))
	if err != nil {
		log.Println("Error removing task:", err)
	}
}

func toggleBackupMode(task map[string]string) {
	_, err := http.Get(fmt.Sprintf("http://127.0.0.1:8275/toggle-backup-mode?secure_id=%s", task["SecureId"]))
	if err != nil {
		log.Println("Error toggling backup mode:", err)
	}
}

func openApp(task map[string]string) {
	_, err := http.Get(fmt.Sprintf("http://127.0.0.1:8275/launch-app?AppId=%s", task["SecureId"]))
	if err != nil {
		log.Println("Error launching app:", err)
	}
}

func prepareMagasin(app *tview.Application) *tview.Pages {
	pages := tview.NewPages()

	// Sections (ToutEnUn and Grapins)
	toutEnUnSection := tview.NewFlex().SetDirection(tview.FlexRow)
	grapinsSection := tview.NewFlex().SetDirection(tview.FlexRow)

	// Loading popup
	loadingPopup := tview.NewModal().
		SetText("Loading...").
		AddButtons([]string{"Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.SwitchToPage("Main")
		})

	// Add sections to Pages
	pages.AddPage("ToutEnUn", toutEnUnSection, true, true)
	pages.AddPage("Grapins", grapinsSection, true, false)
	pages.AddPage("Loading", loadingPopup, false, false)

	// Fetch and process data
	go fetchMagasinData(app, pages, toutEnUnSection, grapinsSection)

	// Main menu to switch sections
	menu := tview.NewList().
		AddItem("Tout en un", "View Tout en un apps", 't', func() {
			pages.SwitchToPage("ToutEnUn")
		}).
		AddItem("Grapins", "View Grapins", 'g', func() {
			pages.SwitchToPage("Grapins")
		}).
		AddItem("Quit", "Quit the application", 'q', func() {
			app.Stop()
		})

	// Set menu as root of the pages
	pages.AddPage("Main", menu, true, true)

	return pages

}

// fetchData fetches the app configurations from the provided URL
func fetchMagasinData(app *tview.Application, pages *tview.Pages, toutEnUnSection *tview.Flex, grapinsSection *tview.Flex) {

	os := findOsName()

	// Show loading popup
	app.QueueUpdateDraw(func() {
		pages.SwitchToPage("Loading")
	})

	// Fetch data
	url := "https://raw.githubusercontent.com/ThaaoBlues/ecosys/master/magasin_database.json"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching config: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	var data Data
	if err := json.Unmarshal(body, &data); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	// Process Tout en un apps
	for _, config := range data.ToutEnUnConfigs {
		if contains(config.SupportedPlatforms, os) {
			app.QueueUpdateDraw(func() {
				toutEnUnSection.AddItem(generateCard(config, "Install Tout en un", func() {
					showLoading(app, pages)
					go installApp(config, "/install-tout-en-un", app, pages)
				}), 0, 1, true)
			})
		}
	}

	// Process Grapin apps
	for _, config := range data.GrapinConfigs {
		if contains(config.SupportedPlatforms, os) {
			app.QueueUpdateDraw(func() {
				grapinsSection.AddItem(generateCard(config, "Install Grapin", func() {
					showLoading(app, pages)
					go installApp(config, "/install-grapin", app, pages)
				}), 0, 1, true)
			})
		}
	}

	// Hide loading popup after processing
	app.QueueUpdateDraw(func() {
		pages.SwitchToPage("Main")
	})
}

// generateCard creates a card UI component for the app configuration
func generateCard(config Config, buttonText string, onClick func()) *tview.Flex {
	// Card layout
	card := tview.NewFlex().SetDirection(tview.FlexRow)

	// App title
	title := tview.NewTextView().SetText(config.AppName).SetDynamicColors(true)

	// App description
	description := tview.NewTextView().SetText(config.AppDescription).SetDynamicColors(true)

	// Install button
	button := tview.NewButton(buttonText).SetSelectedFunc(onClick)

	// Add components to the card
	card.AddItem(title, 1, 0, false).
		AddItem(description, 1, 0, false).
		AddItem(button, 1, 0, false)

	return card
}

// installApp performs an HTTP POST request to install the app
func installApp(config Config, endpoint string, app *tview.Application, pages *tview.Pages) {
	url := fmt.Sprintf("http://localhost%s", endpoint) // Adjust the URL as necessary

	// Marshal the app config into JSON
	jsonData, err := json.Marshal(config)
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}

	// Make the POST request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error installing app %s: %v", config.AppName, err)
		showError(app, pages, fmt.Sprintf("Error installing %s: %v", config.AppName, err))
		return
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response for app %s: %v", config.AppName, err)
		showError(app, pages, fmt.Sprintf("Error reading response for %s: %v", config.AppName, err))
		return
	}

	// Handle success
	log.Printf("Successfully installed app %s: %s", config.AppName, string(body))
	showSuccess(app, pages, fmt.Sprintf("Successfully installed %s!", config.AppName))
}

// showLoading shows the loading modal
func showLoading(app *tview.Application, pages *tview.Pages) {
	app.QueueUpdateDraw(func() {
		pages.SwitchToPage("Loading")
	})
}

// showError shows an error message in a modal
func showError(app *tview.Application, pages *tview.Pages, message string) {
	errorModal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Ok"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.SwitchToPage("Main")
		})

	app.QueueUpdateDraw(func() {
		pages.AddPage("Error", errorModal, true, true)
		pages.SwitchToPage("Error")
	})
}

// showSuccess shows a success message in a modal
func showSuccess(app *tview.Application, pages *tview.Pages, message string) {
	successModal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Ok"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.SwitchToPage("Main")
		})

	app.QueueUpdateDraw(func() {
		pages.AddPage("Success", successModal, true, true)
		pages.SwitchToPage("Success")
	})
}

// contains checks if a slice contains a specific item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.ToLower(s) == strings.ToLower(item) {
			return true
		}
	}
	return false
}

func findOsName() string {
	return "Linux"
}
