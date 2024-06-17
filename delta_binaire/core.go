package delta_binaire

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

type Delta_instruction struct {
	InstructionType string
	Data            []int8
	ByteIndex       int64
}

type Delta struct {
	Instructions []Delta_instruction
	FilePath     string
}

func BuilDelta(relative_path string, absolute_path string, old_file_size int64, old_file_content []byte) Delta {

	new_file_handler, err := os.Open(absolute_path)

	if err != nil {
		log.Fatal("Error while opening the file from real filesystem to seek changes. : ", err)
	}

	defer new_file_handler.Close()

	// first determine wich file is longer
	// if old file is longer, we will need to truncate file
	// (trucature is made by delta instruction of type t)

	// else, new file is longer so we don't need to do anything as it will be a regular "ab"
	// type instruction

	new_file_stat, err := new_file_handler.Stat()
	if err != nil {
		log.Fatal("Error while obtenaining old file statistics.")
	}

	new_file_size := new_file_stat.Size()
	log.Println("New file size : ", new_file_size)

	var needs_truncature bool = false
	if old_file_size > new_file_size {
		needs_truncature = true
	}
	log.Println("needs truncature : ", needs_truncature)

	old_file_reader := bufio.NewReader(bytes.NewReader(old_file_content))
	new_file_reader := bufio.NewReader(new_file_handler)

	var new_file_buff byte

	var file_delta []Delta_instruction

	var byte_index int64 = 0
	// blocking byte index is used to concatenate
	// multiples consecutives bytes change
	// into a single delta instruction
	var blocking_byte_index int64 = 0

	// errors related to files manipulations
	var new_err error

	var i int64 = 0
	log.Println("old file size : ", old_file_size)
	log.Println("old file content : ", old_file_content)
	for (i < old_file_size || byte_index < new_file_size) && (new_err != io.EOF) {

		new_file_buff, new_err = new_file_reader.ReadByte()
		//log.Println("byte read : ", new_file_buff)

		// zip files contains 0 so we don't need this
		// NOT USED ANYMORE
		/*if new_file_buff == 0 {

			new_err = io.EOF
			break
		}*/

		if new_err != nil {
			log.Fatal("Erreur dans la lecture du fichier : ", err)

		}

		//log.Println(new_file_buff, int8(new_file_buff))

		// initialize to a non-zero value as some files actually need zeros in them
		// and comparing it to non-initialized buffer would act as if the old file already
		// had zeros
		var old_file_buff byte = 0xff

		if i < old_file_size {
			old_file_buff, err = old_file_reader.ReadByte()
			if err != nil {
				log.Fatal("Erreur dans la lecture du fichier : ", err)
			}
		}

		// new delta instruction

		var delta_index int = 0
		var byte_index_cond bool = true
		if len(file_delta) > 0 {
			delta_index = len(file_delta) - 1
			byte_index_cond = (file_delta[delta_index].ByteIndex != blocking_byte_index)
		}

		// the comparison does not work if old file is at EOF
		// because old_file_buff will always be 0xff
		// so it would skip all the 255 byte
		// so to counter that we must double check

		if ((new_file_buff != old_file_buff) || (i >= old_file_size)) && (byte_index_cond) {
			/*if new_file_buff == 0xff {
				log.Println("255 !!! :: cast = ", int8(new_file_buff))
			}*/
			inst := Delta_instruction{
				Data:            []int8{int8(new_file_buff)},
				InstructionType: "ab",
				ByteIndex:       byte_index,
			}
			//log.Println("append : ", inst)
			file_delta = append(file_delta, inst)

			byte_index = byte_index + 1

		} else {
			// continue to fill bytes to delta instruction
			if (new_file_buff != old_file_buff) || (i >= old_file_size) {

				// add the byte we've just read to the data of the delta chunk

				file_delta[len(file_delta)-1].Data = append(file_delta[len(file_delta)-1].Data, int8(new_file_buff))

				// don't forget to increment byte index
				byte_index = byte_index + 1

			} else {
				// same bytes, regular case we just increment counters
				byte_index = byte_index + 1
				blocking_byte_index = byte_index
			}
		}

		i++

	}

	if needs_truncature {

		var delta_index int = 0
		if len(file_delta) > 0 {
			delta_index = len(file_delta) - 1
			file_delta = append(file_delta,
				Delta_instruction{
					Data:            []int8{0},
					InstructionType: "t",
					ByteIndex:       new_file_size,
				})

		} else {
			file_delta[delta_index] = Delta_instruction{
				Data:            []int8{0},
				InstructionType: "t",
				ByteIndex:       new_file_size,
			}
		}

	}

	log.Println(file_delta)

	return Delta{Instructions: file_delta, FilePath: relative_path}
}

func (delta Delta) PatchFile() {

	file_handler, err := os.OpenFile(delta.FilePath, os.O_WRONLY, os.ModeAppend)

	if errors.Is(err, os.ErrNotExist) {
		os.Create(delta.FilePath)
		file_handler, err = os.OpenFile(delta.FilePath, os.O_WRONLY, os.ModeAppend)
	}

	if err != nil {
		log.Fatal("Unable to open file to apply patch.", err)
	}

	defer file_handler.Close()

	file_writer := io.WriteSeeker(file_handler)

	for i := 0; i < len(delta.Instructions); i++ {
		switch delta.Instructions[i].InstructionType {
		case "ab":
			file_writer.Seek(delta.Instructions[i].ByteIndex, io.SeekStart)

			// convert int8 to byte
			// all this shit is needed as a byte is unsigned in go
			// and signed in java

			for _, v := range delta.Instructions[i].Data {
				//log.Println(v, byte(v))
				_, err = file_writer.Write([]byte{byte(v)})
				if err != nil {
					log.Fatal(err)
				}
			}

		case "t":
			file_handler.Truncate(delta.Instructions[i].ByteIndex)
		}

	}

}

func (delta Delta) Serialize() string {
	instructions := make([]strings.Builder, len(delta.Instructions))
	var instructions_joiner strings.Builder

	for i, instruction := range delta.Instructions {

		instructions[i].WriteString(instruction.InstructionType)
		instructions[i].WriteString(",")
		for _, data := range instruction.Data {
			instructions[i].WriteString(strconv.Itoa(int(data)))
			instructions[i].WriteString(",")
		}
		instructions[i].WriteString(strconv.FormatInt(instruction.ByteIndex, 10))

		instructions_joiner.WriteString(instructions[i].String())

		// so it does not append a commas at the end of the string
		if i < (len(delta.Instructions) - 1) {
			instructions_joiner.WriteString("|")
		}
	}

	return instructions_joiner.String()
}

func (delta Delta) DeSerialize(instructions_string []byte) {
	instructionParts := bytes.Split(instructions_string, []byte("|"))

	delta.Instructions = make([]Delta_instruction, len(instructionParts))

	for i, instructionStr := range instructionParts {

		instructionData := bytes.Split(instructionStr, []byte(","))

		dataInts := make([]int8, len(instructionData)-2)

		for j := 1; j < len(instructionData)-1; j++ {

			tmp, _ := strconv.Atoi(string(instructionData[j]))
			dataInts[j-1] = int8(tmp)
		}

		byteIndex, _ := strconv.ParseInt(string(instructionData[len(instructionData)-1]), 10, 64)

		delta.Instructions[i] = Delta_instruction{
			InstructionType: string(instructionData[0]),
			Data:            dataInts,
			ByteIndex:       byteIndex,
		}
	}

}
