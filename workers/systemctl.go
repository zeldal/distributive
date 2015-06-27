package workers

import (
	"github.com/CiscoCloud/distributive/tabular"
	"github.com/CiscoCloud/distributive/wrkutils"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

// RegisterSystemctl registers these checks so they can be used.
func RegisterSystemctl() {
	wrkutils.RegisterCheck("systemctlloaded", systemctlLoaded, 1)
	wrkutils.RegisterCheck("systemctlactive", systemctlActive, 1)
	wrkutils.RegisterCheck("systemctlsockpath", systemctlSockPath, 1)
	wrkutils.RegisterCheck("systemctlsockunit", systemctlSockUnit, 1)
	wrkutils.RegisterCheck("systemctltimer", systemctlTimer, 1)
	wrkutils.RegisterCheck("systemctltimerloaded", systemctlTimerLoaded, 1)
	wrkutils.RegisterCheck("systemctlunitfilestatus", systemctlUnitFileStatus, 2)
}

// systemctlExists returns whether or not systemctl is available ona given
// machine
func systemctlExists() bool {
	_, err := exec.Command("systemctl", "--version").CombinedOutput()
	if err != nil && strings.Contains(err.Error(), "not found in $PATH") {
		return false
	}
	return true
}

// systemctlShouldExist logs and quits if it doesn't.
func systemctlShouldExist() {
	if !systemctlExists() {
		log.Fatal("Couldn't execute systemctl")
	}
}

func systemctlService(service string, activeOrLoaded string) (exitCode int, exitMessage string) {
	// cmd depends on whether we're checking active or loaded
	cmd := exec.Command("systemctl", "show", "-p", "ActiveState", service)
	if activeOrLoaded == "loaded" {
		cmd = exec.Command("systemctl", "show", "-p", "LoadState", service)
	}

	out, err := cmd.CombinedOutput()
	outstr := string(out)
	if err != nil {
		wrkutils.ExecError(cmd, outstr, err)
	}
	contained := "ActiveState=active"
	if activeOrLoaded == "loaded" {
		contained = "LoadState=loaded"
	}
	if strings.Contains(outstr, contained) {
		return 0, ""
	}
	msg := "Service not " + activeOrLoaded
	return wrkutils.GenericError(msg, service, []string{outstr})
}

// systemctlLoaded checks to see whether or not a given service is loaded
func systemctlLoaded(parameters []string) (exitCode int, exitMessage string) {
	return systemctlService(parameters[0], "loaded")
}

// systemctlActive checks to see whether or not a given service is active
func systemctlActive(parameters []string) (exitCode int, exitMessage string) {
	return systemctlService(parameters[0], "active")
}

// systemctlSock is an abstraction of systemctlSockPath and systemctlSockUnit,
// it reads from `systemctl list-sockets` and sees if the value is in the
// appropriate column.
func systemctlSock(value string, path bool) (exitCode int, exitMessage string) {
	systemctlShouldExist() // log.Fatal if it doesn't
	column := 1
	if path {
		column = 0
	}
	cmd := exec.Command("systemctl", "list-sockets")
	values := wrkutils.CommandColumnNoHeader(column, cmd)
	if tabular.StrIn(value, values) {
		return 0, ""
	}
	return wrkutils.GenericError("Socket not found", value, values)
}

// systemctlSock checks to see whether the sock at the given path is registered
// within systemd using the sock's filesystem path.
func systemctlSockPath(parameters []string) (exitCode int, exitMessage string) {
	return systemctlSock(parameters[0], true)
}

// systemctlSock checks to see whether the sock at the given path is registered
// within systemd using the sock's unit name.
func systemctlSockUnit(parameters []string) (exitCode int, exitMessage string) {
	return systemctlSock(parameters[0], false)
}

func getTimers(all bool) []string {
	cmd := exec.Command("systemctl", "list-timers")
	if all {
		cmd = exec.Command("systemctl", "list-timers", "--all")
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := "Couldn't execute `systemctl list-timers`:\n\t" + err.Error()
		log.Fatal(msg)
	}
	// matches anything with hyphens or letters, then a ".timer"
	re := regexp.MustCompile("(\\w|\\-)+\\.timer")
	return re.FindAllString(string(out), -1)
}

// timers(exitCode int, exitMessage string) is pure DRY for systemctlTimer and systemctlTimerLoaded
func timersWorker(unit string, all bool) (exitCode int, exitMessage string) {
	timers := getTimers(all)
	if tabular.StrIn(unit, timers) {
		return 0, ""
	}
	return wrkutils.GenericError("Timer not found", unit, timers)
}

// systemctlTimer reports whether a given timer is running (by unit).
func systemctlTimer(parameters []string) (exitCode int, exitMessage string) {
	return timersWorker(parameters[0], false)
}

// systemctlTimerLoaded checks to see if a timer is loaded, even if it might
// not be active
func systemctlTimerLoaded(parameters []string) (exitCode int, exitMessage string) {
	return timersWorker(parameters[0], true)
}

// systemctlUnitFileStatus checks whether or not the given unit file has the
// given status: static | enabled | disabled
func systemctlUnitFileStatus(parameters []string) (exitCode int, exitMessage string) {
	// getUnitFilesWithStatuses returns a pair of string slices that hold
	// the name of unit files with their current statuses.
	getUnitFilesWithStatuses := func() (units []string, statuses []string) {
		cmd := exec.Command("systemctl", "--no-pager", "list-unit-files")
		units = wrkutils.CommandColumnNoHeader(0, cmd)
		cmd = exec.Command("systemctl", "--no-pager", "list-unit-files")
		statuses = wrkutils.CommandColumnNoHeader(1, cmd)
		// last two are empty line and junk statistics we don't care about
		return units[:len(units)-2], statuses[:len(statuses)-2]
	}
	unit := parameters[0]
	status := parameters[1]
	units, statuses := getUnitFilesWithStatuses()
	var actualStatus string
	for i, un := range units {
		if un == unit {
			actualStatus = statuses[i]
			if actualStatus == status {
				return 0, ""
			}
		}
	}
	msg := "Unit didn't have status"
	return wrkutils.GenericError(msg, status, []string{actualStatus})
}