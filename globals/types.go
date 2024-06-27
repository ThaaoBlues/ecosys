/*
 * @file            globals/types.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2024-03-24 15:15:33
 * @lastModified    2024-06-27 19:25:33
 * Copyright ©Théo Mougnibas All rights reserved
 */

package globals

import "qsync/delta_binaire"

type QEvent struct {
	Flag        string
	FileType    string
	Delta       delta_binaire.Delta
	FilePath    string
	NewFilePath string
	SecureId    string
}

type ToutEnUnConfig struct {
	AppName               string // well... the app's name ?
	AppDownloadUrl        string // the url where to download the app
	NeedsInstaller        bool   // if we need to run the binary installer
	AppLauncherPath       string // the path to the main executable of your app
	AppInstallerPath      string // the installer path
	AppUninstallerPath    string // the uninstaller path
	AppSyncDataFolderPath string // the folder where the data to synchronize is stored
	AppDescription        string // well that's the app's descriptions
	AppIconURL            string
	SupportedPlatforms    []string
}

type GrapinConfig struct {
	AppName               string
	AppSyncDataFolderPath string
	NeedsFormat           bool
	SupportedPlatforms    []string
	AppDescription        string // well that's the app's descriptions
	AppIconURL            string
}

type MinGenConfig struct {
	AppName         string
	AppId           int
	BinPath         string
	Type            string
	SecureId        string
	UninstallerPath string
}

type GenArrayInterface[T any] interface {
	Add(val T) GenArray[T]
	Get(i int) T
	Size() int
	PopLast() GenArray[T]
}

// TODO solve this with generics
type GenArray[T any] struct {
	items []T
}

func (array *GenArray[T]) Add(val T) {
	array.items = append(array.items, val)
}

func (array *GenArray[T]) Get(i int) T {
	return array.items[i]
}
func (array *GenArray[T]) PopLast() {
	array.items = array.items[:len(array.items)-1]
}

func (array *GenArray[T]) Size() int {
	return len(array.items)
}

func (array *GenArray[T]) ToSlice() []T {
	return array.items
}
