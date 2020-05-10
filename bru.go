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
	simulate bool     // simulate or not ?
	clocked  bool     // clocked or not
}

var bruData string              // contents of hdl file.
var finalGo string              // final go code -> hdl translation
var mainFuncCode string = "$\n" // string to store name of chip being simulated
var scNumIns int                // simulated chip's number of inputs
var scNumOuts int               // simulated chip's number of outputs
var scInArgsBits string         // single bit inputs to simulation chip
var scInArgsBufs []string       // multi bit inputs to simulation chip
var scOArgsBits []string        // single bit outputs of simulation chip
var oArgBitsAll []string        // Clean this mess !
var oBufDec string              // multi bit outputs of simulation chip ('s declaration)
var clkOBufDec string           // multi bit outputs of simulation chip ('s declaration)
var chipsInFile []string        // names of chips in the hdl file
var sim bool                    // I have forgotten what this variable does
var numSim int                  // number of chips registered for simulation
var globalClocked bool          // is the sim circuit clocked?
var outFileName string          // name of file to store all the outputs in
var writeblank bool             // on error, write blank -> true
var loopCommand string          // contains the code to be added if any output is connected as an input

//	stores an intermediate mostly-go code. Does not contain the runtime/ main function.
//	it is initialized with the 3 basic gates available to us- and, or and not.
var goEquivOutput string = `
var outputs string

func printArrays(arr []string) string {
	out := "[ "
	for _, v := range arr {
		out += v + " "
	}
	out += "] "
	return out
}

func writeString(filename string, data string) error {
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
			newChip.name = strings.TrimSpace(v[1:])
		} else if strings.HasPrefix(v, "IN") {
			//	line contains declaration of chip inputs
			newChip.args = strings.TrimSpace(v[3:])
			newChip.numIns = strings.Count(newChip.args, " ") + 1
		} else if strings.HasPrefix(v, "OUT") {
			//	line contains declaration of chip outputs
			newChip.outputs = strings.Split(v[4:], " ")
			newChip.numOuts = len(newChip.outputs)
		} else if strings.TrimSpace(v) == "CON" {
			//	line indicates that the description for the chip's
			//	functioning is going to start and adds the lines
			//	that follow (containing the instructions on how the
			//	chip is constructed) to a variable. Process stops
			//	when a line containing 'end' is encountered.
			for i := k + 1; i < len(lines); i++ {
				s := strings.TrimSpace(lines[i])
				if s != "END" {
					newChip.commands += s + "\n"
				}
			}
			break
		} else if strings.TrimSpace(v) == "SIM" {
			//	line indicates that the current chip being evaluated
			//	is also scheduled to be simumlated.
			if numSim == 0 {
				numSim++
			} else {
				fmt.Println("ERROR: More than one chip scheduled for simulation.")
				// WRITEBLANK = true   -->   not really required here
				os.Exit(2)
			}
			newChip.simulate = true
			mainFuncCode += "    RUN_FUNC: [" + newChip.name + "]\n"
			sim = true
		} else if strings.TrimSpace(v) == "CLOCKED" {
			globalClocked = true
			newChip.clocked = true
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
	// fmt.Println(inps)
	outs := chip.outputs
	var bufOuts []string
	var bitOuts []string
	for _, v := range outs {
		if strings.Contains(v, "[") {
			bufOuts = append(bufOuts, v)
			if chip.simulate && chip.clocked {
				oArgBitsAll = append(oArgBitsAll, v[:strings.Index(v, "[")+1])
			}
		} else {
			bitOuts = append(bitOuts, v)
			if chip.simulate && chip.clocked {
				oArgBitsAll = append(oArgBitsAll, v)
			}
		}
	}
	for _, v := range inps {
		if strings.Index(v, "[") == -1 {
			inArgsBits += v + " "
		} else {
			inArgsBufs = append(inArgsBufs, v)
		}
	}
	inArgsBits = strings.TrimSpace(inArgsBits)
	if inArgsBits != "" {
		fun += strings.ReplaceAll(inArgsBits, " ", ", ") + " string"
	}
	if len(inArgsBufs) > 0 {
		if inArgsBits != "" {
			fun += ", "
		}
		for k, v := range inArgsBufs {
			v = strings.TrimSpace(v)
			fun += v[:strings.Index(v, "[")]
			fun += " " + v[strings.Index(v, "["):] + "string"
			if k != len(inArgsBufs)-1 {
				fun += ", "
			}
		}
	}
	fun += ")("
	for k := range bitOuts {
		fun += "string"
		if k != len(bitOuts)-1 {
			fun += ", "
		}
	}
	if len(bufOuts) > 0 {
		for k, v := range bufOuts {
			fun += v[strings.Index(v, "["):] + "string"
			if k != len(bufOuts)-1 {
				fun += ", "
			}
		}
	}
	fun += ") {\n"
	for _, v := range bufOuts {
		fun += "var " + v[:strings.Index(v, "[")] + " " + v[strings.Index(v, "["):strings.Index(v, "]")+1] + "string\n"
	}
	if chip.simulate == true {
		template :=
			`for k := range I {
v = "X"
}`
		for _, v := range bufOuts {
			n := v[:strings.Index(v, "[")]
			oBufDec += "\nvar " + n + " " + v[strings.Index(v, "["):strings.Index(v, "]")+1] + "string"
			oBufDec += "\n" + strings.Replace(strings.Replace(template, "v", n+"[k]", 1), "I", n, 1)
			if globalClocked {
				oBufDec += "\nvar l" + n + " " + v[strings.Index(v, "["):strings.Index(v, "]")+1] + "string"
				oBufDec += "\n" + strings.Replace(strings.Replace(template, "v", "l"+n+"[k]", 1), "I", n, 1)
			}
		}
	}
	c := ""
	funCom := ""
	for _, v := range returnLines(chip.commands) {
		c = ""
		if strings.Contains(v, "=") {
			n := strings.TrimSpace(v[:strings.Index(v, "=")])
			if strings.Contains(n, ",") && strings.Contains(n, "[") {
				vars := strings.Split(n, ", ")
				for _, w := range vars {
					if !strings.Contains(w, "[") {
						fun += "var " + w + " string\n"
					}
				}
			} else if !strings.Contains(n, "[") {
				v = strings.Replace(v, "=", ":=", 1)
			}
			c = v
		}
		funCom += c
		funCom += "\n"
	}
	fun += funCom
	fun += "return "
	for k, v := range chip.outputs {
		if strings.Contains(v, "[") {
			fun += v[:strings.Index(v, "[")]
		} else {
			fun += v
		}
		if k != chip.numOuts-1 {
			fun += ", "
		}
	}
	fun += "\n}"
	if chip.simulate == true {
		scOArgsBits = bitOuts
	}
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
			chipInTF = tempFile[strings.Index(tempFile, "#") : strings.Index(tempFile, "END")+3]
			tempFile = tempFile[strings.Index(tempFile, "END")+3:]
			for _, v := range chipsInFile {
				if retNames(chipInTF)[0] == v {
					fmt.Println("WARNING : preventing double loading of --> " + retNames(chipInTF)[0])
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
		if strings.Contains(bruData, "LOAD") {
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
		temp = bruData[strings.Index(bruData, "#") : strings.Index(bruData, "END")+3]
		bruData = bruData[strings.Index(bruData, "END")+3:]
		lines := returnLines(temp)
		chips = append(chips, parseLines(lines))
		if strings.Contains(mainFuncCode, chips[i].name) {
			scNumOuts = chips[i].numOuts
			scNumIns = chips[i].numIns
			inps := strings.Split(chips[i].args, " ")
			for _, v := range inps {
				if strings.Contains(v, "|") {
					// support for looped back outputs
					l := chips[i].args
					in := v[strings.Index(v, "(")+1 : strings.Index(v, "|")]
					out := v[strings.Index(v, "|")+1 : strings.Index(v, ")")]
					loopCommand += in + " = " + out + "\n"
					v = v[strings.Index(v, "(")+1 : strings.Index(v, "|")]
					p := "(" + in + "|" + out + ")"
					l = l[:strings.Index(l, p)] + in + l[strings.Index(l, p)+len(p):]
					chips[i].args = l
				}
				if strings.Index(v, "[") == -1 {
					scInArgsBits += v + " "
				} else {
					scInArgsBufs = append(scInArgsBufs, v)
				}
			}
		}
		//fmt.Println(chips[i].name)
		goEquivOutput += constructFunction(chips[i])
		goEquivOutput += "\n\n"
	}
}

func prepareOutput(vars []string) string {
	code := ""
	for _, v := range vars {
		if strings.Contains(v, "[") {
			code += "outputs += printArrays(" + v[:strings.Index(v, "[")] + ")"
		} else {
			code += "outputs += " + v + " + \" \""
		}
		code += "\n"
	}
	return code
}

//	interpretScript interprets the information provided by the script file and
//	writes the corresponding syntactically correct go equvalent code for the script
//	contents to the finalGo variable. It also adds code to declare and initialize
//	variables for the inputs and outputs to be provided and obtained to and from the
//	chip that is being simulated.
func interpretScript(simFunc, scriptData string) {
	if globalClocked {
		if strings.Contains(scriptData, "call") {
			finalGo = ""
			fmt.Println("ERROR: CLOCKED chip not compatible with \"call\" command")
			//os.Exit(2)
			return
		}
		if sim {
			lines := returnLines(scriptData)
			mainFuncStuff := ""
			firstIF := true
			moreDurs := false
			varList, varDec := assembleVlistVdec()
			mainFuncStuff += varDec
			for _, v := range lines {
				if strings.Contains(v, "=") {
					if moreDurs && strings.Contains(v, "dur") {
						fmt.Println("ERROR: 'dur' declared more than once")
						writeblank = true
						finalGo = ""
						return
					}
					if strings.Contains(v, "dur") && moreDurs == false {
						moreDurs = true
						mainFuncStuff += "\n" + strings.ReplaceAll(strings.TrimSpace(v), "=", ":=")
						mainFuncStuff += "\noutputs := \"\""

						mainFuncStuff += "\nfor t := 0; t < dur; t++ {"

						inpvars := strings.Split(varList, ",")
						mainFuncStuff += "\nif "
						for k, s := range inpvars {
							s = strings.TrimSpace(s)
							mainFuncStuff += "l" + s + " != " + s
							if k != len(inpvars)-1 {
								mainFuncStuff += " || "
							}
						}
						mainFuncStuff += " {\n"
						for k, g := range oArgBitsAll {
							mainFuncStuff += g
							if k != len(oArgBitsAll)-1 {
								mainFuncStuff += ", "
							}
						}
						mainFuncStuff += " = " + assembleFuncCall(simFunc, varList) + "\n}"
						mainFuncStuff += " else{\n"
						for k, g := range oArgBitsAll {
							mainFuncStuff += g
							if k != len(oArgBitsAll)-1 {
								mainFuncStuff += ", "
							}
						}
						mainFuncStuff += " = "
						for k, g := range oArgBitsAll {
							mainFuncStuff += "l" + g
							if k != len(oArgBitsAll)-1 {
								mainFuncStuff += ", "
							}
						}
						mainFuncStuff += "\n}\n"

						for k, g := range oArgBitsAll {
							mainFuncStuff += "l" + g
							if k != len(oArgBitsAll)-1 {
								mainFuncStuff += ", "
							}
						}
						mainFuncStuff += " = "
						for k, g := range oArgBitsAll {
							mainFuncStuff += g
							if k != len(oArgBitsAll)-1 {
								mainFuncStuff += ", "
							}
						}
						mainFuncStuff += "\n"
						mainFuncStuff += loopCommand
						for k, g := range inpvars {
							mainFuncStuff += "l" + strings.TrimSpace(g)
							if k != len(inpvars)-1 {
								mainFuncStuff += ", "
							}
						}
						mainFuncStuff += " = "
						for k, g := range inpvars {
							mainFuncStuff += strings.TrimSpace(g)
							if k != len(inpvars)-1 {
								mainFuncStuff += ", "
							}
						}

					} else if strings.Contains(v, "t") {
						if firstIF {
							mainFuncStuff += "\nif " + strings.ReplaceAll(v, "=", "==")
							firstIF = false
						} else {
							mainFuncStuff += " else if " + strings.ReplaceAll(v, "=", "==")
						}
					} else {
						literal := v[strings.Index(v, "="):]
						switch {
						case strings.Contains(literal, "0"):
							literal = strings.ReplaceAll(literal, "0", "\"0\"")
						case strings.Contains(literal, "1"):
							literal = strings.ReplaceAll(literal, "1", "\"1\"")
						case strings.Contains(literal, "X"):
							literal = strings.ReplaceAll(literal, "X", "\"X\"")
						}
						v = v[:strings.Index(v, "=")]
						mainFuncStuff += "\n" + v + literal
					}
				} else {
					if strings.Contains(v, "}") {
						mainFuncStuff += "\n" + strings.TrimSpace(v)
					}
				}
			}
			mainFuncStuff += "\n" + prepareOutput(oArgBitsAll)
			mainFuncStuff += "outputs += \"\\n\"\n"
			mainFuncStuff += "writeString(\"" + outFileName + "\", outputs)\n"
			finalGo += mainFuncStuff
			finalGo += "}"
		} else {
			finalGo = goEquivOutput[:strings.Index(goEquivOutput, "$")]
		}
	} else if sim {
		lines := returnLines(scriptData)
		var mainGo string
		var temp string
		var varList string
		decalred := false
		for _, v := range lines {
			if strings.Contains(v, "//") {
				continue
			}
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
		finalGo = goEquivOutput[:strings.Index(goEquivOutput, "$")]
	}
}

var outVarList string

//	assembleVlistVdecOlist function prepares the list of inputs and outputs that are required for the chip's
//	execution.
func assembleVlistVdec() (string, string) {
	var varDec string
	varList := ""
	clockedVarList := ""
	vals := ""
	scInArgsBits = strings.TrimSpace(scInArgsBits)
	inArgsBits := strings.Split(scInArgsBits, " ")
	if scInArgsBits != "" {
		for k, v := range inArgsBits {
			varList += v
			if globalClocked {
				clockedVarList += "l" + v
			}
			vals += "\"X\""
			if k != len(inArgsBits)-1 {
				vals += ", "
				varList += ", "
				if globalClocked {
					clockedVarList += ", "
				}
			}
		}
		varDec += "var " + varList + " string = " + vals + "\n"
		if globalClocked {
			varDec += "var " + clockedVarList + " string = " + vals + "\n"
		}
	}
	template :=
		`for k := range I {
		v = "X"
	}`
	if len(scInArgsBufs) > 0 {
		if scInArgsBits != "" {
			varList += ", "
		}
		for k, v := range scInArgsBufs {
			n := v[:strings.Index(v, "[")]
			varDec += "\nvar " + n + " " + v[strings.Index(v, "["):strings.Index(v, "]")+1] + "string"
			varDec += "\n" + strings.Replace(strings.Replace(template, "v", n+"[k]", 1), "I", n, 1)
			if globalClocked {
				varDec += "\nvar l" + n + " " + v[strings.Index(v, "["):strings.Index(v, "]")+1] + "string"
				varDec += "\n" + strings.Replace(strings.Replace(template, "v", "l"+n+"[k]", 1), "I", n, 1)
			}
			varList += n
			if k != len(scInArgsBufs)-1 {
				varList += ", "
			}
		}
	}
	varDec += "\n"
	clockedVarList = ""
	vals2 := ""
	vlO := ""
	if len(scOArgsBits) > 0 {
		varDec += "var "
		clockedVarList += "var "
		for l, v := range scOArgsBits {
			vlO += v
			vals2 += "\"X\""
			if globalClocked {
				clockedVarList += "l" + v
			}
			if l != len(scOArgsBits)-1 {
				vlO += ", "
				vals2 += ", "
				if globalClocked {
					clockedVarList += ", "
				}
			}
		}
		varDec += vlO + " string = " + vals2 + "\n"
		if globalClocked {
			clockedVarList += " string = " + vals2 + "\n"
		}
	}
	if globalClocked {
		varDec += clockedVarList
	}
	varDec += oBufDec + "\n"
	if globalClocked {
		varDec += clkOBufDec + "\n"
	}
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
		if len(os.Args) <= 3 {
			fmt.Println("ERROR : script not found. \n\tTry using '-s scriptName' to provide a script file.")
			finalGo = ""
			// WRITEBLANK = true   -->   not really required here
			return
		}
		simFunc := goEquivOutput[strings.Index(goEquivOutput, "$")+1 : strings.LastIndex(goEquivOutput, "$")]
		simFunc = simFunc[strings.Index(simFunc, "[")+1 : strings.Index(simFunc, "]")]
		simFunc = strings.TrimSpace(simFunc)
		var runMode string
		var scriptData string
		var indexStart int
		finalGo = "package main\n\nimport (\n\"io\"\n\"os\"\n)\n"
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
				if strings.TrimSpace(scriptData) == "" {
					fmt.Println("ERROR: script file is empty.")
					// WRITEBLANK = true   -->   not really required here
					finalGo = ""
					return
					//os.Exit(1)
				}
				interpretScript(simFunc, scriptData)
				finalGo += "\n}"
			}
		}
		if globalClocked {
			if len(os.Args) < 6 {
				fmt.Println("ERROR: Filename to store outputs not specified.")
				// WRITEBLANK = true   -->   not really required here
				finalGo = ""
				return
				//os.Exit(2)
			}
		}
	} else if sim == false {
		if len(os.Args) > 2 {
			fmt.Println("WARNING : script given but nothing to simulate")
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
	writeblank = false
	if len(os.Args) > 4 {
		fileMode := os.Args[4][strings.Index(os.Args[4], "-")+1:]
		if fileMode == "o" {
			if len(os.Args) == 5 {
				fmt.Println("ERROR: Filename to store outputs not specified.")
				os.Exit(2)
			} else {
				outFileName = os.Args[5]
				fmt.Println(os.Args[5])
			}
		}
	}
	bruData = loadFile(os.Args[1])
	chipsInFile = retNames(bruData)
	preproc()
	makeChip()
	mainFuncCode += "$\n"
	goEquivOutput += mainFuncCode
	ui()
	writeToFile("main.go", finalGo)
}
