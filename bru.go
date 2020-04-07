/*
	:: issues ::
		ADD DEBUG MODE TO INTERMEDIATE VALUES.
*/

/*
   Copyright (C) 2020 Ashwin Godbole

   This file is part of Bru.

   Bru is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   Bru is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with Bru. If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

//	properties of every chip
type chip struct {
	name     string   // chip name
	args     string   // inputs taken by the chip
	numOuts  int      // number of outputs taken by the chip
	numIns   int      // number of inputs taken by the chip
	commands string   // instructions on how the chip works
	outputs  []string // list of outputs obtained after chip has been evaluated
}

var bruData string              // contents of hdl file.
var finalGo string              // final go code -> hdl translation
var mainFuncCode string = "$\n" // string to store name of chip being simulated
var scNumIns int                // simulated chip's number of inputs
var scNumOuts int               // simulated chip's number of outputs
var chipsInFile []string        // names of chips in the hdl file
var sim bool

//	stores an intermediate mostly-go code. Does not contain the runtime/ main function.
//	it is initialized with the 3 basic gates available to us- and, or and not.
var goEquivOutput string = `
func not(i string) string {	
	switch i {
	case "1":
		return "0"
	case "0":
		return "1"
	}
	return "X"
}

func and(a, b string) string {
	if a == "0" || b == "0" {
		return "0"
	} else if a == "1" && b == "1" {
		return "1"
	}
	return "X"
}


func or(a, b string) string {
	if a == "1" || b == "1" {
		return "1"
	} else if a == "0" && b == "0" {
		return "0"
	}
	return "X"
}

`

//	general function to search for an element in a string slice
func strSliceHas(slice []string, str string) bool {
	if len(slice) > 0 {
		for _, v := range slice {
			fmt.Println(v, "||", str)
			if v == str {
				return true
			}
		}
	}
	return false
}

//	loadFile loads a file, ie. returns the context of a file as a string.
func loadFile(filename string) string {
	var content string = ""
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
	}
	for _, v := range file {
		content += string(v)
	}
	return content
}

//	writeToFile writes a string to a file.
func writeToFile(filename string, data string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, data)
	if err != nil {
		return err
	}
	return file.Sync()
}

//	returnLines splits a string at the newline character, ie. it splits
// 	the given string into its individual lines and returns the lines as
//	an array of strings
func returnLines(str string) []string {
	lines := strings.Split(str, "\n")
	for k, v := range lines {
		lines[k] = strings.TrimSpace(v)
	}
	return lines
}

//	parseLines looks for keywords in the form of words or characters
//	and calls the corresponding function that deals with a line that
//	contains the keyword found. (this function delegates actions to
//	other functions [in most cases] )
func parseLines(lines []string) chip {
	var newChip chip
	for k, v := range lines {
		if strings.HasPrefix(v, "#") {
			//	line contains declaration of chip name
			newChip.name = v[1:]
		} else if strings.HasPrefix(v, "in") {
			//	line contains declaration of chip inputs
			newChip.args = strings.TrimSpace(v[3:])
			newChip.numIns = strings.Count(newChip.args, " ") + 1
		} else if strings.HasPrefix(v, "out") {
			//	line contains declaration of chip outputs
			newChip.outputs = strings.Split(v[4:], " ")
			newChip.numOuts = len(newChip.outputs)
		} else if strings.TrimSpace(v) == "con" {
			//	line indicates that the description for the chip's
			//	functioning is going to start and adds the lines
			//	that follow (containing the instructions on how the
			//	chip is constructed) to a variable. Process stops
			//	when a line containing 'end' is encountered.
			for i := k + 1; i < len(lines); i++ {
				s := strings.TrimSpace(lines[i])
				if s != "end" {
					newChip.commands += strings.Replace(s, "=", ":=", 1) + "\n"
				}
			}
			break
		} else if strings.TrimSpace(v) == "--sim" {
			//	line indicates that the current chip being evaluated
			//	is also scheduled to be simumlated.
			mainFuncCode += "    RUN_FUNC: [" + newChip.name + "]\n"
			sim = true
		}
	}
	return newChip
}

//	constructFuntion assembles/ generates a syntactically correct go function
//	which is equivalent to the hdl version of the chip. It takes a chip struct
//	'object' which contains all the information about a chip and uses it to
//	generate the go function
func constructFunction(chip chip) string {
	var inArgsBits string
	var inArgsBufs []string
	fun := "func "
	fun += chip.name + "("
	inps := strings.Split(chip.args, " ")
	for _, v := range inps {
		if strings.Index(v, "[") == -1 {
			inArgsBits += v + " "
		} else {
			inArgsBufs = append(inArgsBufs, v)
		}
	}
	inArgsBits = strings.TrimSpace(inArgsBits)
	fun += strings.ReplaceAll(inArgsBits, " ", ", ") + " string)("
	for k, v := range inArgsBufs {
		v = strings.TrimSpace(v)
		fun += v[:strings.Index(v, "[")]
		fun += " " + v[strings.Index(v, "["):] + "string"
		if k != len(inArgsBufs)-1 {
			fun += ", "
		} else {
			fun += ")("
		}
	}
	for i := 0; i < chip.numOuts; i++ {
		fun += "string"
		if i != chip.numOuts-1 {
			fun += ", "
		} else {
			fun += ") {\n"
		}
	}
	fun += chip.commands
	fun += "return "
	for k, v := range chip.outputs {
		if k != chip.numOuts-1 {
			fun += v + ", "
		} else {
			fun += v
		}
	}
	fun += "\n}"
	return fun
}

//	preproc is a preprocessor that identifies the "load" block in the file and replaces
//	the block with the contents of the hdl files whose names are listed within this load
//	block. If the file after one replacement still contains a "load" block, it calls
//	itself recursively till there are no more "load" blocks remaining
func preproc() {
	if strings.Index(bruData, "[") == -1 {
		return
	}

	loads := strings.Split(strings.TrimSpace(bruData[strings.Index(bruData, "[")+1:strings.Index(bruData, "]")]), "\n")
	bruData = bruData[strings.Index(bruData, "]")+1:]

	for _, v := range loads {
		tempFile := loadFile(v)
		var chipInTF string
		flag := false
		for {
			chipInTF = tempFile[strings.Index(tempFile, "#") : strings.Index(tempFile, "end")+3]
			tempFile = tempFile[strings.Index(tempFile, "end")+3:]
			for _, v := range chipsInFile {
				if retNames(chipInTF)[0] == v {
					fmt.Println("preventing double loading of --> " + retNames(chipInTF)[0])
					flag = true
					break
				}
			}
			if flag == false {
				bruData = chipInTF + "\n" + bruData
			} else {
				flag = false
			}
			if !strings.Contains(tempFile, "#") {
				break
			}
		}
		if strings.Contains(bruData, "load") {
			preproc()
		}
	}
}

//	makeChip calls other functions to interpret the contents of the hdl file, and
//	for each chip declared in the hdl file, it generates a chip object and also
//	adds the go equivalent code for that chip to the goEquivOutput variable.
func makeChip() {
	var chips []chip
	var temp string

	numChips := strings.Count(bruData, "#")

	for i := 0; i < numChips; i++ {
		temp = bruData[strings.Index(bruData, "#") : strings.Index(bruData, "end")+3]
		// fmt.Println(temp)
		bruData = bruData[strings.Index(bruData, "end")+3:]
		lines := returnLines(temp)
		chips = append(chips, parseLines(lines))
		if strings.Contains(mainFuncCode, chips[i].name) {
			scNumOuts = chips[i].numOuts
			scNumIns = chips[i].numIns
		}
		//fmt.Println(chips[i].name)
		goEquivOutput += constructFunction(chips[i])
		goEquivOutput += "\n\n"
	}
}

//	interpretScript interprets the information provided by the script file and
//	writes the corresponding syntactically correct go equvalent code for the script
//	contents to the finalGo variable. It also adds code to declare and initialize
//	variables for the inputs and outputs to be provided and obtained to and from the
//	chip that is being simulated.
func interpretScript(simFunc, scriptData string) {
	if sim == true {
		lines := returnLines(scriptData)
		var mainGo string
		var temp string
		var varList string
		decalred := false
		for _, v := range lines {
			if strings.Contains(v, "=") {
				literal := v[strings.Index(v, "="):]
				switch {
				case strings.Contains(literal, "0"):
					literal = strings.ReplaceAll(literal, "0", "\"0\"")
				case strings.Contains(literal, "1"):
					literal = strings.ReplaceAll(literal, "1", "\"1\"")
				case strings.Contains(literal, "X"):
					literal = strings.ReplaceAll(literal, "X", "\"X\"")
				}
				if decalred == false {
					v = v[:strings.Index(v, "=")]
					mainGo += v + literal + "\n"
				} else {
					v = v[:strings.Index(v, "=")]
					mainGo += v + literal + "\n"
				}
			} else if command := strings.TrimSpace(v); command == "call" {
				if decalred == false {
					varList, temp = assembleVlistVdec()
					finalGo += temp
				}
				decalred = true
				finalGo += mainGo + "\n\nfmt.Println(" + assembleFuncCall(simFunc, varList) + ")\n"
				mainGo = ""
			} else if strings.Contains(v, "in") {
				varList = strings.ReplaceAll(strings.TrimSpace(v[strings.Index(v, "in")+2:]), " ", ", ")
			}
		}
	} else {
		//fmt.Println(goEquivOutput)
		finalGo = goEquivOutput[:strings.Index(goEquivOutput, "$")]
	}
}

//	assembleVlistVdec function prepares the list of inputs and outputs that are required for the chip's
//	execution.
func assembleVlistVdec() (string, string) {
	varDec := "var "
	varList := ""
	vals := ""
	for k := 0; k < scNumIns; k++ {
		varList += "i" + strconv.Itoa(k+1)
		vals += "\"X\""
		if k != scNumIns-1 {
			vals += ", "
			varList += ", "
		}
	}
	varDec += varList + " string = " + vals + "\n"
	varDec += "var "
	for l := 0; l < scNumOuts; l++ {
		varDec += "o" + strconv.Itoa(l+1)
	}
	varDec += " string\n"
	return varList, varDec
}

//	assembleFuncCall prepares the function call for the chip being simulated
//	with the right number of inputs
func assembleFuncCall(funcName string, varList string) string {
	var functionCall string
	functionCall += funcName + "(" + varList + ")"
	return functionCall
}

//	ui adds finishing touches to the finalGo variable and also looks at the command line arguments provided
//	to check the mode of execution and loads the script files content if the run-mode provided is the script
//	mode.
func ui() {
	if sim == true {
		simFunc := goEquivOutput[strings.Index(goEquivOutput, "$")+1 : strings.LastIndex(goEquivOutput, "$")]
		simFunc = simFunc[strings.Index(simFunc, "[")+1 : strings.Index(simFunc, "]")]
		simFunc = strings.TrimSpace(simFunc)
		var runMode string
		var scriptData string
		var indexStart int
		finalGo = "package main\n\nimport \"fmt\"\n\n"
		finalGo += goEquivOutput[:strings.Index(goEquivOutput, "$")]
		finalGo += "\nfunc main() {\n"
		if len(os.Args) > 2 {
			indexStart = strings.Count(os.Args[2], "-")
			runMode = os.Args[2][indexStart:]
			switch {
			case runMode == "i" || runMode == "stdin":
				fmt.Println("feature not ready yet")
			case runMode == "s" || runMode == "script":
				scriptData = loadFile(os.Args[3])
				interpretScript(simFunc, scriptData)
				finalGo += "\n}"
			}
		}
	} else if sim == false {
		if len(os.Args) > 2 {
			fmt.Println("script given but nothing to simulate")
		}
		finalGo = goEquivOutput[:strings.Index(goEquivOutput, "$")]
	}
}

//	retrieve names of all chips in the hdl file
func retNames(from string) []string {
	t := from
	var names []string
	var n string
	l := returnLines(t)
	for _, v := range l {
		if strings.HasPrefix(v, "#") {
			n = v[1:]
			names = append(names, n)
		}
	}
	return names
}

func main() {
	// to run the program >>  bru.exe [source HDL] -s[--script] [script File]
	bruData = loadFile(os.Args[1])
	chipsInFile = retNames(bruData)
	preproc()
	makeChip()
	mainFuncCode += "$\n"
	goEquivOutput += mainFuncCode
	ui()
	writeToFile("main.go", finalGo)
}
