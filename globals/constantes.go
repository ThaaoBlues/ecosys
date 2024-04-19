package globals

var QSyncWriteableDirectory string = ""

// this function is used to make qsync write its internal files
// into another directory than its root
// For example, it is useful for android because
// most of the filesystem is read-only or event not accessible
func SetQsyncWriteableDirectory(path string) {
	QSyncWriteableDirectory = path
}
