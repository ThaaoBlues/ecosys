/*
 * @file            delta_binaire/core.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2024-04-19 14:18:54
 * @lastModified    2024-07-21 20:37:09
 * Copyright ©Théo Mougnibas All rights reserved
 */

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

func CalculateBufferSize(file_size int64) int {
	// do not make chunk of more than 100Mo
	// we stop when we have a chunk size
	// that is the maximum one
	// that can still fit 2 times in the file

	var c int = 100
	if file_size > 100<<10 {
		c = 100 << 10
	} else {
		for (c <= 100<<10) && (c < (int(file_size) >> 2)) {
			c = c << 1
		}
	}

	return c
}

func byteBufferToInt8Slice(buff []byte) []int8 {

	size := len(buff)
	ret := make([]int8, size)

	for i := 0; i < size; i++ {
		ret[i] = int8(buff[i])
	}

	return ret
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

	new_file_reader := bufio.NewReader(new_file_handler)

	var BUFF_SIZE int = CalculateBufferSize(new_file_size)
	log.Println("Calculated adapted buffer size : ", BUFF_SIZE, " bytes")

	var new_file_buff = make([]byte, BUFF_SIZE)

	var file_delta []Delta_instruction

	var byte_index int64 = 0
	// blocking byte index is used to concatenate
	// multiples consecutives bytes change
	// into a single delta instruction
	var blocking_byte_index int64 = 0

	// errors related to files manipulations
	var new_err error

	var global_index int64 = 0
	log.Println("old file size : ", old_file_size)
	//log.Println("old file content : ", old_file_content)

	var instruction_buffer bytes.Buffer

	if new_file_size > 0 {
		for (global_index < old_file_size || byte_index < new_file_size) && (new_err != io.EOF) {

			new_buff_fill_size, new_err := new_file_reader.Read(new_file_buff)

			if new_err != nil {
				log.Fatal("Erreur dans la lecture du fichier : ", err)

			}

			//log.Println(new_file_buff, int8(new_file_buff))

			// new delta instruction
			// we are looping throught the newly read block

			// var instructions_buffer bytes.Buffer;
			for new_buff_index := 0; new_buff_index < new_buff_fill_size; new_buff_index++ {

				var delta_index int = 0
				var byte_index_cond bool = true

				if len(file_delta) > 0 {
					delta_index = len(file_delta) - 1
					byte_index_cond = (file_delta[delta_index].ByteIndex != blocking_byte_index)
				}

				// initialize to a non-zero value as some files actually need zeros in them
				// and comparing it to non-initialized buffer would act as if the old file already
				// had zeros

				// the comparison does not work if old file is at EOF
				// because old_file_buff will always be 0xff
				// so it would skip all the 255 byte
				// so to counter that we must double check

				var old_file_byte byte = 0xff
				if global_index < old_file_size {
					old_file_byte = old_file_content[global_index]
				}

				if ((new_file_buff[new_buff_index] != old_file_byte) || (global_index >= old_file_size)) && (byte_index_cond) {

					//instruction_buffer.WriteByte(new_file_buff[new_buff_index])

					inst := Delta_instruction{
						Data:            []int8{int8(new_file_buff[new_buff_index])},
						InstructionType: "ab",
						ByteIndex:       global_index,
					}
					//log.Println("append : ", inst)

					instruction_buffer.WriteByte(new_file_buff[new_buff_index])
					file_delta = append(file_delta, inst)

				} else {
					// continue to fill bytes to delta instruction
					if (new_file_buff[new_buff_index] != old_file_byte) || global_index >= old_file_size {

						// add the byte we've just read to the data of the delta chunk
						instruction_buffer.WriteByte(new_file_buff[new_buff_index])

						// check if we are at end of read buffer block and flush the write buffer
						// this operation removes the need to clone a byte array to extend it at each new byte in a block
						// This block is necessary as if we weren't using this, the last differences chunk being
						// processed ad each read buffer block would not be written to its delta Instruction
						if instruction_buffer.Len() > 0 && (new_buff_index == new_buff_fill_size-1) {

							file_delta[len(file_delta)-1].Data = byteBufferToInt8Slice(instruction_buffer.Bytes())
							instruction_buffer.Reset()

							// prepare a new block for the next buffer, with the good index

							blocking_byte_index = global_index + 1

						}

						//file_delta[len(file_delta)-1].Data = append(file_delta[len(file_delta)-1].Data, int8(new_file_buff[new_buff_index]))

					} else {
						// same bytes, regular case we just increment counters

						// check if we are at the end of a block change ( i.e the data stream has bytes in it )
						// as if this is just a random byte in a chunk of unchanged bytes the buffer would be empty
						// this operation removes the need to clone a byte array to extend it at each new byte in a block

						if instruction_buffer.Len() > 0 {
							copy(file_delta[len(file_delta)-1].Data, byteBufferToInt8Slice(instruction_buffer.Bytes()))
							instruction_buffer.Reset()
						}

						blocking_byte_index = global_index + 1
					}
				}
				// don't forget to increment byte index
				global_index++
			}

			byte_index = byte_index + int64(new_buff_fill_size)

		}

	}

	if needs_truncature {

		if len(file_delta) > 0 {
			file_delta = append(file_delta,
				Delta_instruction{
					Data:            []int8{0},
					InstructionType: "t",
					ByteIndex:       new_file_size,
				})

		} else {
			file_delta = append(file_delta,
				Delta_instruction{
					Data:            []int8{0},
					InstructionType: "t",
					ByteIndex:       new_file_size,
				})
		}

	}

	log.Println("built this delta : ", file_delta)

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

func (delta *Delta) DeSerialize(instructions_string []byte) {
	delta.Instructions = make([]Delta_instruction, 0)
	var block_index int = 0
	var i int = 0
	for i < len(instructions_string) {

		var block_builder strings.Builder

		for block_index < len(instructions_string) && instructions_string[block_index] != '|' {
			block_builder.WriteByte(instructions_string[block_index])
			block_index += 1
		}
		if block_builder.Len() == 0 {
			break
		}
		// to skip the final "|" for the next block
		block_index += 1

		instructionData := bytes.Split(bytes.NewBufferString(block_builder.String()).Bytes(), []byte(","))
		//instructionData := block_builder.String()

		dataInts := make([]int8, len(instructionData)-2)

		for j := 1; j < len(instructionData)-1; j++ {

			tmp, _ := strconv.Atoi(string(instructionData[j]))
			dataInts[j-1] = int8(tmp)
		}

		byteIndex, _ := strconv.ParseInt(string(instructionData[len(instructionData)-1]), 10, 64)

		delta.Instructions = append(delta.Instructions,
			Delta_instruction{
				InstructionType: string(instructionData[0]),
				Data:            dataInts,
				ByteIndex:       byteIndex,
			})

		i += 1
	}

}
