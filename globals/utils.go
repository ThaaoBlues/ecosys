package globals

import (
	"fmt"
	"os"
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
