package delta_binaire

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
)

type delta_instruction struct {
	InstructionType string
	Data            []byte
	ByteIndex       int64
}

type Delta struct {
	Instructions []delta_instruction
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

	var needs_truncature bool = false
	if old_file_size > new_file_size {
		needs_truncature = true
	}

	// TODO: IMPLEMENT TRUNCATURE IN PATCH AND IN DELTA FORGE

	new_file_reader := io.ByteReader(bufio.NewReader(new_file_handler))

	var new_file_buff, old_file_buff byte

	/* replace this with a real database reading */
	old_file_buff = 10

	file_delta := make([]delta_instruction, 0)

	var byte_index int64 = 0
	var blocking_byte_index int64 = 0

	// errors related to files manipulations
	var new_err error

	var i int64

	for (i < old_file_size) && (new_err != io.EOF) {
		new_file_buff, new_err = new_file_reader.ReadByte()

		if new_err != nil {
			log.Fatal("Erreur dans la lecture du fichier : ", err)
		}

		// replace this line with a database read byte
		old_file_buff = old_file_content[i]

		// new delta instruction

		var delta_index int = 0
		var byte_index_cond bool = true
		if len(file_delta) > 0 {
			delta_index = len(file_delta) - 1
			byte_index_cond = (file_delta[delta_index].ByteIndex != blocking_byte_index)
		}

		if (new_file_buff != old_file_buff) && (byte_index_cond) {
			inst := delta_instruction{
				Data:            []byte{new_file_buff},
				InstructionType: "ab",
				ByteIndex:       byte_index,
			}

			file_delta = append(file_delta, inst)

			byte_index = byte_index + 1

		} else {
			// continue to fill bytes to delta instruction
			if new_file_buff != old_file_buff {

				// add the byte we've just read to the data of the delta chunk
				file_delta[len(file_delta)-1].Data = append(file_delta[len(file_delta)-1].Data, new_file_buff)

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
				delta_instruction{
					Data:            []byte{0},
					InstructionType: "t",
					ByteIndex:       new_file_size,
				})

		} else {
			file_delta[delta_index] = delta_instruction{
				Data:            []byte{0},
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
			_, err = file_writer.Write(delta.Instructions[i].Data)
			if err != nil {
				log.Fatal(err)
			}
		case "t":
			file_handler.Truncate(delta.Instructions[i].ByteIndex)
		}

	}

}
