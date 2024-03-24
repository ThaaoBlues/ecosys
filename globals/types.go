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
