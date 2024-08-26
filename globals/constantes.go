/*
 * @file            globals/constantes.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2024-04-19 14:18:54
 * @lastModified    2024-08-26 12:59:17
 * Copyright ©Théo Mougnibas All rights reserved
 */

package globals

var EcosysWriteableDirectory string = ""

// this function is used to make ecosys write its internal files
// into another directory than its root
// For example, it is useful for android because
// most of the filesystem is read-only or event not accessible
func SetecosysWriteableDirectory(path string) {
	EcosysWriteableDirectory = path
}
