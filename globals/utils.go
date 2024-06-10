package globals

import (
	"archive/tar"
	"archive/zip"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"qsync/delta_binaire"
	"strconv"
	"strings"
)

// exists returns whether the given file or directory exists
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func SerializeQevent(event QEvent) string {
	instructions := make([]string, len(event.Delta.Instructions))
	for i, instruction := range event.Delta.Instructions {
		dataStr := make([]string, len(instruction.Data))
		for j, data := range instruction.Data {
			dataStr[j] = strconv.Itoa(int(data))
		}
		instructions[i] = instruction.InstructionType + "," + strings.Join(dataStr, ",") + "," + strconv.FormatInt(instruction.ByteIndex, 10)
	}
	return fmt.Sprintf("%s;%s;%s;%s;%s;%s;%s",
		event.Flag,
		event.FileType,
		strings.Join(instructions, "|"),
		event.Delta.FilePath,
		event.FilePath,
		event.NewFilePath,
		event.SecureId,
	)
}

func DeSerializeQevent(data string) QEvent {
	parts := strings.Split(data, ";")

	// check if instructions are present, as some requests does not needs it
	if len(parts[2]) > 1 {
		instructionParts := strings.Split(parts[2], "|")
		instructions := make([]delta_binaire.Delta_instruction, len(instructionParts))
		for i, instructionStr := range instructionParts {
			instructionData := strings.Split(instructionStr, ",")
			dataInts := make([]int8, len(instructionData)-2)
			for j := 1; j < len(instructionData)-1; j++ {
				val, _ := strconv.Atoi(instructionData[j])
				dataInts[j-1] = int8(val)
			}
			byteIndex, _ := strconv.ParseInt(instructionData[len(instructionData)-1], 10, 64)
			instructions[i] = delta_binaire.Delta_instruction{
				InstructionType: instructionData[0],
				Data:            dataInts,
				ByteIndex:       byteIndex,
			}
		}
		return QEvent{
			Flag:        parts[0],
			FileType:    parts[1],
			Delta:       delta_binaire.Delta{Instructions: instructions, FilePath: parts[3]},
			FilePath:    parts[4],
			NewFilePath: parts[5],
			SecureId:    parts[6],
		}
	} else {
		return QEvent{
			Flag:        parts[0],
			FileType:    parts[1],
			FilePath:    parts[4],
			NewFilePath: parts[5],
			SecureId:    parts[6],
		}

	}

}

func ZipFolder(source, target string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)

	err = filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Create the correct zip file header
		info, err := d.Info()

		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Set the header name to be the relative path
		header.Name, err = filepath.Rel(source, path)
		if err != nil {
			return err
		}

		// If it's a directory, add a trailing slash to the header name
		if d.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// If it's not a directory, write the file content to the zip file
		if !d.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}
		return nil
	})

	// Ensure the zipWriter is closed only after filepath.WalkDir is done
	if err != nil {
		zipWriter.Close()
		return err
	}

	log.Println("Closing ZipWriter")

	return zipWriter.Close()
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
