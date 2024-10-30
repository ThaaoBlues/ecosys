package globals

var CurrentLang = "en" // Default language
var availableLangagues = []string{"en", "fr"}

func SetCurrentLangIfAvailable(lang string) {
	for _, supportedLanguage := range availableLangagues {
		if supportedLanguage == lang {
			CurrentLang = lang
		}
	}
}

var Translations = map[string]map[string]string{
	"en": {
		"devicesTitle":                  "Devices on your network",
		"tasksTitle":                    "Active tasks on your device",
		"taskActionMenuTitle":           "What do you wanna do about this task ?",
		"deviceActions":                 "Device Actions",
		"sendFile":                      "Send File",
		"sendFolder":                    "Send Folder",
		"sendText":                      "Send Text",
		"removeTask":                    "Remove Task",
		"openApp":                       "Open App",
		"syncAnotherDevice":             "Sync Another Device",
		"enableBackupMode":              "Enable Backup Mode",
		"disableBackupMode":             "Disable Backup Mode",
		"alertTaskCreated":              "Task Created at ",
		"navigationHelp":                "To navigate in Ecosys you can use the mouse.\n But if you are someone cool (⌐■_■), use the tab (\u2B7E) key to change section and the arrow up/down keys to select an option in the section. The enter key (\u23CE) is used to validate your selection.",
		"loading":                       "Loading...",
		"createSyncTask":                "Create a new sync task",
		"openMagasin":                   "Open the app marketplace",
		"toggleLargageAerien":           "Allow/Refuse to receive Largages Aeriens",
		"openLargagesFolder":            "Open the folder where are stored received Largages Aeriens",
		"qToQuit":                       "Press 'q' to get out of this section.",
		"largageDesktopNotification":    "Incoming Largage Aérien !!  \n File name : ",
		"largageAcceptationPrompt":      "Accept the largage aérien ?  \n File name : ",
		"multiLargageAcceptationPrompt": "Accept the MULTI largage aérien ?  \n It will be stored at : ",
		"appLinkingSuccess":             "Applications successfully linked to each others ! Please restart ecosys before changing anything",
		"selectSyncFolderPrompt":        "Choose a path where new sync files will be stored.",
		"openSettings":                  "Open Ecosys settings",
		"success":                       "Operation successfully done !",
		"toggleAutostart":               "Allow Ecosys to start automatically",
	},
	"fr": {
		"devicesTitle":                  "Appareils sur votre réseau",
		"tasksTitle":                    "Tâches actives sur votre appareil",
		"taskActionMenuTitle":           "Que veux-tu faire en rapprot avec cette tache ?",
		"deviceActions":                 "Actions de l'appareil",
		"sendFile":                      "Envoyer un fichier",
		"sendFolder":                    "Envoyer un dossier",
		"sendText":                      "Envoyer un texte",
		"removeTask":                    "Supprimer la tâche",
		"openApp":                       "Ouvrir l'application",
		"syncAnotherDevice":             "Synchroniser un autre appareil",
		"enableBackupMode":              "Activer le mode de sauvegarde",
		"disableBackupMode":             "Désactiver le mode de sauvegarde",
		"alertTaskCreated":              "Tâche créée à ",
		"navigationHelp":                "La navigation dans Ecosys se fait via la souris.\n Mais si vous êtes quelqu'un de cool (⌐■_■), vous pouvez utiliser la touche tab (\u2B7E) pour changer de section et les flèches haut/bas pour selectionner une option de la section. La touche entrée (\u23CE) est là pour valider.",
		"loading":                       "Chargement...",
		"createSyncTask":                "Créer une tâche de synchronisation",
		"openMagasin":                   "Ouvrir le magasin d'applications",
		"toggleLargageAerien":           "Autoriser/Refuser les largages aerien",
		"openLargagesFolder":            "Ouvrir le dossier contenant les largages aerien",
		"qToQuit":                       "Appuyez sur la touche 'q' pour sortir d'ici.",
		"largageDesktopNotification":    "Largage Aérien en approche !!  \n Nom du fichier : ",
		"largageAcceptationPrompt":      "Accepter le largage aérien ?  \n Nom du fichier : ",
		"multiLargageAcceptationPrompt": "Accepter le MULTI largage aérien ?  \n Il sera téléchargé dans le dossier suivant : ",
		"appLinkingSuccess":             "Applications liées avec succés ! Il maintenant est préférable de relancer ecosys avant de modifier des choses dessus.",
		"selectSyncFolderPrompt":        "Selectionnez le dossier vers lequel vous voulez établir la synchronisation.",
		"openSettings":                  "Ouvrir les paramètres d'Ecosys",
		"success":                       "Opération effectuée avec success !",
		"toggleAutostart":               "Autoriser Ecosys à démarrer automatiquement",
	},
}
