/*
 * @file            globals/utils.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2024-04-28 16:50:11
 * @lastModified    2024-08-26 17:15:48
 * Copyright ©Théo Mougnibas All rights reserved
 */

package globals

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"ecosys/delta_binaire"
	"ecosys/separators"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// exists returns whether the given file or directory exists
func ExistsInFilesystem(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func SerializeQevent(event QEvent) string {

	instructions := make([]strings.Builder, len(event.Delta.Instructions))
	var instructions_joiner strings.Builder

	for i, instruction := range event.Delta.Instructions {

		instructions[i].WriteString(instruction.InstructionType)
		instructions[i].WriteString(separators.BytesToHex(separators.VALUE_SEPARATOR))

		// attention, we write bytes bc it is not always text
		for _, octet := range instruction.Data {
			instructions[i].WriteByte(byte(octet))
		}

		instructions[i].WriteString(separators.BytesToHex(separators.VALUE_SEPARATOR))
		instructions[i].WriteString(strconv.FormatInt(instruction.ByteIndex, 10))

		instructions_joiner.WriteString(instructions[i].String())

		// so it does not append a commas at the end of the string
		if i < (len(event.Delta.Instructions) - 1) {
			instructions_joiner.WriteString(separators.BytesToHex(separators.INSTRUCTION_SEPARATOR))
		}
	}
	sep := separators.BytesToHex(separators.FIELD_SEPARATOR)
	return fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s%s%s%d%s%s",
		event.Flag,
		sep,
		event.FileType,
		sep,
		instructions_joiner.String(),
		sep,
		event.Delta.FilePath,
		sep,
		event.FilePath,
		sep,
		event.NewFilePath,
		sep,
		event.VersionToPatch,
		sep,
		event.SecureId,
	)
}

func DeSerializeQevent(data string, secure_id string) QEvent {
	log.Println("splitting serialized event")
	parts := strings.Split(data, separators.BytesToHex(separators.FIELD_SEPARATOR))
	log.Println("Event split")

	file_version, err := strconv.ParseInt(string(parts[6]), 10, 64)
	if err != nil {
		log.Println("Error while parsing event file version")
		file_version = 0
	}
	//log.Println(parts)
	// check if instructions are present, as some requests does not needs it
	if len(parts[2]) > 1 {

		var delta delta_binaire.Delta
		delta.DeSerialize(parts[2])
		delta.FilePath = string(parts[3])
		return QEvent{
			Flag:           string(parts[0]),
			FileType:       string(parts[1]),
			Delta:          delta,
			FilePath:       string(parts[4]),
			NewFilePath:    string(parts[5]),
			SecureId:       secure_id,
			VersionToPatch: file_version,
		}
	} else {

		return QEvent{
			Flag:           string(parts[0]),
			FileType:       string(parts[1]),
			FilePath:       string(parts[4]),
			NewFilePath:    string(parts[5]),
			SecureId:       secure_id,
			VersionToPatch: file_version,
		}

	}

}

func TarFolder(source, target string) error {
	tarFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	tarWriter := tar.NewWriter(tarFile)
	defer tarWriter.Close()

	err = filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Create the correct tar file header
		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Set the header name to be the relative path
		header.Name, err = filepath.Rel(source, path)
		if err != nil {
			return err
		}

		// Write the header to the tar file
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If it's not a directory, write the file content to the tar file
		if !d.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println("Tar file created successfully")
	return nil
}

func UntarFolder(tarFile, destDir string) error {
	// Open the tar file
	file, err := os.Open(tarFile)
	if err != nil {
		return fmt.Errorf("could not open tar file: %w", err)
	}
	defer file.Close()

	// Create a new tar reader
	tarReader := tar.NewReader(file)

	// Iterate through the tar archive
	for {
		// Get the next header
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("could not read tar header: %w", err)
		}

		// Determine the full path for the file or directory
		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directories as needed
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("could not create directory: %w", err)
			}
		case tar.TypeReg:
			// Create a file and write its contents
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("could not create directories for file: %w", err)
			}
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("could not create file: %w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("could not write file: %w", err)
			}
			outFile.Close()
		default:
			return fmt.Errorf("unsupported tar entry type: %c", header.Typeflag)
		}
	}
	return nil
}

// generateRandomString generates a random string of a given length
func GenerateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		randomByte, err := randByte()
		if err != nil {
			return "", err
		}
		b[i] = charset[randomByte%byte(len(charset))]
	}
	return string(b), nil
}

// randByte generates a random byte
func randByte() (byte, error) {
	var b [1]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

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

// Known executable file signatures
var signatures = map[string][]byte{
	"ELF":     {0x7F, 'E', 'L', 'F'},    // Linux/Unix ELF
	"PE":      {'M', 'Z'},               // Windows PE (Portable Executable)
	"Mach-O":  {0xFE, 0xED, 0xFA, 0xCE}, // macOS Mach-O (32-bit)
	"Mach-O2": {0xFE, 0xED, 0xFA, 0xCF}, // macOS Mach-O (64-bit)
	"Mach-O3": {0xCA, 0xFE, 0xBA, 0xBE}, // macOS universal binary
	"Mach-O4": {0xCE, 0xFA, 0xED, 0xFE}, // macOS 32-bit, opposite endian
	"Mach-O5": {0xCF, 0xFA, 0xED, 0xFE}, // macOS 64-bit, opposite endian
}

func IsExecutable(filePath string) bool {
	file, err := os.Open(filePath)

	// don't take risks, assume executable first
	ret := true
	if err != nil {
		return ret
	}
	defer file.Close()

	header := make([]byte, 4) // Read the first 4 bytes
	_, err = io.ReadFull(file, header)
	if err != nil {
		return ret
	}

	ret = false
	for _, signature := range signatures {
		if bytes.HasPrefix(header, signature) {
			ret = true
		}
	}

	return ret
}
