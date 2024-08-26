package separators

import "fmt"

// Separate main fields of the request
var FIELD_SEPARATOR = []byte{
	0x00, 0xFF, 0x00, 0xFF,
	'-', '-', 'C', 'H', 'A', 'M', 'P', '-', '-', 'C', 'H', 'A', 'M', 'P', '-',
	0xFF, 0x00, 0xFF, 0x00,
}

// Separate specific values of a field
var VALUE_SEPARATOR = []byte{
	0x00, 0xFF, 0x00, 0xFF,
	'-', '-', 'V', 'A', 'L', 'U', 'E', '-', '-', 'V', 'A', 'L', 'U', 'E', '-',
	0xFF, 0x00, 0xFF, 0x00,
}

// Separate delta instructions
var INSTRUCTION_SEPARATOR = []byte{
	0x00, 0xFF, 0x00, 0xFF,
	'-', '-', 'I', 'N', 'S', 'T', 'R', 'U', 'C', 'T', 'I', 'O', 'N', '-', '-', 'I', 'N', 'S', 'T', 'R', 'U', 'C', 'T', 'I', 'O', 'N', '-',
	0xFF, 0x00, 0xFF, 0x00,
}

// Convert bytes to hex string
func BytesToHex(bytes []byte) string {
	hexString := ""
	for _, b := range bytes {
		hexString += fmt.Sprintf("%02X", b)
	}
	return hexString
}
