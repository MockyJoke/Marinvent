package runtimepaths

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	windowsChartsDBF    = `C:\ProgramData\Jeppesen\Common\TerminalCharts\charts.dbf`
	windowsVFRChartsDBF = `C:\ProgramData\Jeppesen\Common\TerminalCharts\vfrchrts.dbf`
	windowsTypesDBF     = `C:\ProgramData\Jeppesen\Common\TerminalCharts\ctypes.dbf`
	windowsAirportsDBF  = `C:\ProgramData\Jeppesen\Common\TerminalCharts\Airports.dbf`
)

func DefaultChartsDBF() string {
	if runtime.GOOS == "windows" {
		return windowsChartsDBF
	}
	return filepath.Join("win", "data", "Charts", "charts.dbf")
}

func DefaultVFRChartsDBF() string {
	if runtime.GOOS == "windows" {
		return windowsVFRChartsDBF
	}
	return filepath.Join("win", "data", "Charts", "vfrchrts.dbf")
}

func DefaultTypesDBF() string {
	if runtime.GOOS == "windows" {
		return windowsTypesDBF
	}
	return filepath.Join("win", "data", "Charts", "ctypes.dbf")
}

func DefaultAirportsDBF() string {
	if runtime.GOOS == "windows" {
		return windowsAirportsDBF
	}
	return filepath.Join("win", "data", "Charts", "airports.dbf")
}

func DefaultToolPath(name string) string {
	if runtime.GOOS == "windows" {
		return name
	}

	for _, candidate := range []string{
		filepath.Join("win_deps", "lib", name),
		filepath.Join("win", "lib", name),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return name
}

func ToolWorkDir(toolPath string) string {
	if runtime.GOOS != "windows" {
		for _, fontsDir := range []string{
			filepath.Join("win_deps", "fonts"),
			filepath.Join("win", "fonts"),
		} {
			if absFontsDir, err := filepath.Abs(fontsDir); err == nil {
				if info, statErr := os.Stat(absFontsDir); statErr == nil && info.IsDir() {
					return absFontsDir
				}
			}
		}
	}

	if absToolDir, err := filepath.Abs(filepath.Dir(toolPath)); err == nil {
		return absToolDir
	}

	return filepath.Dir(toolPath)
}

func RuntimeFilePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "windows" {
		return absPath, nil
	}

	return "Z:" + strings.ReplaceAll(filepath.ToSlash(absPath), "/", `\`), nil
}

func PrepareCommand(toolPath string, args ...string) (*exec.Cmd, error) {
	workDir := ToolWorkDir(toolPath)

	if runtime.GOOS == "windows" {
		cmd := exec.Command(toolPath, args...)
		cmd.Dir = workDir
		return cmd, nil
	}

	runtimeToolPath, err := RuntimeFilePath(toolPath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert tool path for Wine: %w", err)
	}

	cmdArgs := append([]string{runtimeToolPath}, args...)
	commandName := "wine"
	commandArgs := cmdArgs
	if os.Getenv("MARINVENT_WINE_XVFB") == "1" {
		if _, err := exec.LookPath("xvfb-run"); err == nil {
			commandName = "xvfb-run"
			commandArgs = append([]string{"-a", "wine"}, cmdArgs...)
		}
	}

	cmd := exec.Command(commandName, commandArgs...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "WINEDEBUG=-all")
	return cmd, nil
}
