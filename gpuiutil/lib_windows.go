//go:build windows

package gpuiutil

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const createNoWindow uint32 = 0x0800_0000

// NewStdCommand returns a command for program that does not open a console window.
func NewStdCommand(program string, args ...string) *exec.Cmd {
	command := exec.Command(program, args...)
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	return command
}

func findPwshInProgramfiles(findAlternate, findPreview bool) (string, bool) {
	var envVar string
	if strconv.IntSize == 64 {
		envVar = "ProgramFiles"
		if findAlternate {
			envVar = "ProgramFiles(x86)"
		}
	} else {
		envVar = "ProgramFiles"
		if findAlternate {
			envVar = "ProgramW6432"
		}
	}
	base, ok := os.LookupEnv(envVar)
	if !ok {
		return "", false
	}
	installBaseDir := filepath.Join(base, "PowerShell")
	entries, err := os.ReadDir(installBaseDir)
	if err != nil {
		return "", false
	}
	found := false
	var bestVersion uint64
	var bestPath string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()
		var version uint64
		if findPreview {
			dashIndex := strings.IndexByte(dirName, '-')
			if dashIndex < 0 || dirName[dashIndex+1:] != "preview" {
				continue
			}
			v, err := strconv.ParseUint(dirName[:dashIndex], 10, 32)
			if err != nil {
				continue
			}
			version = v
		} else {
			v, err := strconv.ParseUint(dirName, 10, 32)
			if err != nil {
				continue
			}
			version = v
		}
		exePath := filepath.Join(installBaseDir, dirName, "pwsh.exe")
		if !isFile(exePath) {
			continue
		}
		if !found || version >= bestVersion {
			found, bestVersion, bestPath = true, version, exePath
		}
	}
	return bestPath, found
}

func findPwshInMsix(findPreview bool) (string, bool) {
	local, ok := os.LookupEnv("LOCALAPPDATA")
	if !ok {
		return "", false
	}
	msixAppDir := filepath.Join(local, "Microsoft\\WindowsApps")
	packageFamilyName := "Microsoft.PowerShell_8wekyb3d8bbwe"
	if findPreview {
		packageFamilyName = "Microsoft.PowerShellPreview_8wekyb3d8bbwe"
	}
	pwshExe := filepath.Join(msixAppDir, packageFamilyName, "pwsh.exe")
	if _, err := os.Stat(pwshExe); err != nil {
		return "", false
	}
	return pwshExe, true
}

func findPwshInScoop() (string, bool) {
	scoopDir, ok := os.LookupEnv("SCOOP")
	if !ok {
		home, ok := os.LookupEnv("USERPROFILE")
		if !ok {
			return "", false
		}
		scoopDir = filepath.Join(home, "scoop")
	}
	pwshExe := filepath.Join(scoopDir, "shims", "pwsh.exe")
	return pwshExe, isFile(pwshExe)
}

func findPwshInDotnetTools() (string, bool) {
	home, ok := os.LookupEnv("USERPROFILE")
	if !ok {
		return "", false
	}
	pwshExe := filepath.Join(home, ".dotnet\\tools\\pwsh.exe")
	return pwshExe, isFile(pwshExe)
}

func findWindowsPowershell() (string, bool) {
	systemRoot, ok := os.LookupEnv("SystemRoot")
	if !ok {
		return "", false
	}
	powershell := filepath.Join(systemRoot, "System32\\WindowsPowerShell\\v1.0\\powershell.exe")
	return powershell, isFile(powershell)
}

func whichGlobal(name string) (string, bool) {
	p, err := exec.LookPath(name)
	return p, err == nil
}

func isFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

var powershell = sync.OnceValues(func() (string, bool) {
	locations := []func() (string, bool){
		func() (string, bool) { return findPwshInProgramfiles(false, false) },
		func() (string, bool) { return findPwshInProgramfiles(true, false) },
		func() (string, bool) { return findPwshInMsix(false) },
		func() (string, bool) { return findPwshInProgramfiles(false, true) },
		func() (string, bool) { return findPwshInMsix(true) },
		func() (string, bool) { return findPwshInProgramfiles(true, true) },
		findPwshInScoop,
		findPwshInDotnetTools,
		func() (string, bool) { return whichGlobal("pwsh.exe") },
		func() (string, bool) { return whichGlobal("powershell.exe") },
		findWindowsPowershell,
	}
	for _, f := range locations {
		if p, ok := f(); ok {
			shell := strings.TrimSpace(p)
			slog.Info("Found powershell in: " + shell)
			return shell, true
		}
	}
	return "", false
})

// GetPowershell returns the path of the preferred PowerShell executable.
func GetPowershell() (string, bool) {
	return powershell()
}

var cmdShell = sync.OnceValue(func() string {
	slog.Warn("Powershell not found, falling back to `cmd`")
	systemRoot, ok := os.LookupEnv("SystemRoot")
	if !ok {
		systemRoot = "C:\\Windows"
	}
	return filepath.Join(systemRoot, "System32\\cmd.exe")
})

// GetWindowsSystemShell returns PowerShell when found and cmd.exe otherwise.
func GetWindowsSystemShell() string {
	if p, ok := GetPowershell(); ok {
		return p
	}
	return cmdShell()
}
