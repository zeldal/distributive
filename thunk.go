// thunks.go provides functions that construct functions in the format that
// Distributive expects, namely the Thunk type, that can be used as health checks.
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Thunk is the type of function that runs without parameters and returns
// an error code and an exit message to be printed to stdout.
// Generally, if exitCode == 0, exitMessage == "".
type Thunk func() (exitCode int, exitMessage string)

// Command runs a shell command, and collapses its error code to 0 or 1.
// It outputs stderr and stdout if the command has error code != 0.
// TODO handle executable in PATH
func Command(toExec string) Thunk {
	return func() (exitCode int, exitMessage string) {
		params := strings.Split(toExec, " ")
		cmd := exec.Command(params[0], params[1:]...)
		// capture outputs
		stdout, err := cmd.StdoutPipe()
		fatal(err)
		stderr, err := cmd.StderrPipe()
		fatal(err)
		// run the command
		err = cmd.Start()
		fatal(err)
		err = cmd.Wait()
		exitCode = 0
		if err != nil {
			exitCode = 1
		}
		stdoutBytes, err := ioutil.ReadAll(stdout)
		stderrBytes, err := ioutil.ReadAll(stderr)
		// Create output message
		exitMessage = ""
		if exitCode != 0 {
			exitMessage = "Command " + toExec + " executed "
			exitMessage += "with exit code " + fmt.Sprint(exitCode)
			exitMessage += "\n\n"
			exitMessage += "stdout: \n" + fmt.Sprint(stdoutBytes)
			exitMessage += "\n\n"
			exitMessage += "stderr: \n" + fmt.Sprint(stderrBytes)
		}
		return exitCode, exitMessage
	}
}

// Running checks if a process is running using `ps aux`, and searching for the
// process name, excluding this process (in case the process name is in the JSON
// file name)
func Running(proc string) Thunk {
	return func() (exitCode int, exitMessage string) {
		cmd := exec.Command("ps", "aux")
		stdoutBytes, err := cmd.Output()
		fatal(err)
		// this regex matches: flag, space, quote, path, filename.json, quote
		re, e := regexp.Compile("-f\\s+\"*?.*?(health-checks/)*?[^/]*.json\"*")
		fatal(e)
		// remove this process from consideration
		filtered := re.ReplaceAllString(string(stdoutBytes), "")
		if strings.Contains(filtered, proc) {
			return 0, ""
		} else {
			return 1, proc + " is not running"
		}
	}
}

// Installed detects whether the OS is using dpkg, rpm, or pacman, queries
// a package accoringly, and returns an error if it is not installed.
func Installed(pkg string) Thunk {
	// getManager returns the program to use for the query
	getManager := func(managers []string) string {
		for _, program := range managers {
			cmd := exec.Command(program, "--version")
			err := cmd.Start()
			// as long as the command was found, return that manager
			message := ""
			if err != nil {
				message = err.Error()
			}
			if strings.Contains(message, "not found") == false {
				return program
			}
		}
		log.Fatal("No package manager found")
		return "No package manager found"
	}
	// getQuery returns the command that should be used to query the pkg
	getQuery := func(program string) (name string, options string) {
		switch program {
		case "dpkg":
			return "dpkg", "-s"
		case "rpm":
			return "rpm", "-q"
		case "pacman":
			return "pacman", "-Qs"
		default:
			log.Fatal("Unsupported package manager")
			return "echo " + program + " is not supported. ", ""
		}
	}

	managers := []string{"dpkg", "rpm", "pacman"}

	return func() (exitCode int, exitMessage string) {
		name, options := getQuery(getManager(managers))
		out, _ := exec.Command(name, options, pkg).Output()
		if strings.Contains(string(out), pkg) == false {
			return 1, "Package " + pkg + " was not found with " + name + "\n"
		}
		return 0, ""
	}
}

// Temp parses the output of lm_sensors and determines if Core 0 (all cores) are
// over a certain threshold as specified in the JSON.
func Temp(max int) Thunk {
	// getCoreTemp returns an integer temperature for a certain core
	getCoreTemp := func(core int) (temp int) {
		out, err := exec.Command("sensors").Output()
		fatal(err)
		// get all-core line up to paren
		lineRegex, err := regexp.Compile("Core " + fmt.Sprint(core) + ":?(.*)\\(")
		fatal(err)
		line := lineRegex.Find(out)
		// get temp from that line
		tempRegex, err := regexp.Compile("\\d+\\.\\d*")
		fatal(err)
		tempString := string(tempRegex.Find(line))
		tempFloat, err := strconv.ParseFloat(tempString, 64)
		fatal(err)
		return int(tempFloat)

	}
	return func() (exitCode int, exitMessage string) {
		temp := getCoreTemp(0)
		if temp < max {
			return 0, ""
		}
		return 1, "Core temp " + fmt.Sprint(temp) + " exceeds defined max of " + fmt.Sprint(max) + "\n"
	}
}