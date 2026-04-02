package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

func runCommandStream(password string, outputChan chan string, command ...string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		// On macOS, some brew commands don't need sudo, but we use it if requested
		// However, brew specifically dislikes being run as root.
		if command[0] == "brew" {
			cmd = exec.Command(command[0], command[1:]...)
		} else {
			cmd = exec.Command("sudo", append([]string{"-S"}, command...)...)
			cmd.Stdin = bytes.NewBufferString(password + "\n")
		}
	} else {
		cmd = exec.Command("sudo", append([]string{"-S"}, command...)...)
		cmd.Stdin = bytes.NewBufferString(password + "\n")
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	readPipe := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			outputChan <- scanner.Text()
		}
	}

	go readPipe(stdout)
	go readPipe(stderr)

	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		return fmt.Errorf("command failed: %v", err)
	}
	return nil
}

func runCommand(password string, command ...string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" && command[0] == "brew" {
		cmd = exec.Command(command[0], command[1:]...)
	} else {
		cmd = exec.Command("sudo", append([]string{"-S"}, command...)...)
		cmd.Stdin = bytes.NewBufferString(password + "\n")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %v, output: %s", err, string(output))
	}
	return string(output), nil
}

func installGit(password string) (string, error) {
	if runtime.GOOS == "darwin" {
		return runCommand(password, "brew", "update")
	}
	return runCommand(password, "apt-get", "update", "-y")
}

func cloneRepo(password string) (string, error) {
	cwd, _ := os.Getwd()
	targetDir := filepath.Join(cwd, "kw-linux")
	os.RemoveAll(targetDir)
	cmd := exec.Command("git", "clone", "https://github.com/LenovoGuy98/kw-linux.git", targetDir)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func cloneInfoRepo(password string) (string, error) {
	cwd, _ := os.Getwd()
	targetDir := filepath.Join(cwd, "kw-info")
	os.RemoveAll(targetDir)
	cmd := exec.Command("git", "clone", "https://github.com/LenovoGuy98/KindWorks_Infomation.git", targetDir)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runInstallApps(password string, outputChan chan string) error {
	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "kw-linux", "install_apps.sh")
	err := exec.Command("chmod", "+x", scriptPath).Run()
	if err != nil {
		return err
	}
	return runCommandStream(password, outputChan, "bash", scriptPath)
}

func runInstallInfo(password string, outputChan chan string) error {
	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "kw-info", "install.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = filepath.Join(cwd, "kw-info", "setup.sh")
	}
	err := exec.Command("chmod", "+x", scriptPath).Run()
	if err != nil {
		return err
	}
	return runCommandStream(password, outputChan, "bash", scriptPath)
}

func configureLibreOffice() error {
	var configPath string
	if runtime.GOOS == "darwin" {
		configPath = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "libreoffice", "4", "user", "registrymodifications.xcu")
	} else {
		configPath = filepath.Join(os.Getenv("HOME"), ".config", "libreoffice", "4", "user", "registrymodifications.xcu")
	}
	
	os.MkdirAll(filepath.Dir(configPath), 0755)
	settings := []string{
		`<item oor:path="/org.openoffice.Office.Common/Save/DefaultSaveOptions"><prop oor:name="WordDocument" oor:op="fuse"><value>MS Word 2007 XML</value></prop></item>`,
		`<item oor:path="/org.openoffice.Office.Common/Save/DefaultSaveOptions"><prop oor:name="ExcelDocument" oor:op="fuse"><value>MS Excel 2007 XML</value></prop></item>`,
		`<item oor:path="/org.openoffice.Office.Common/Save/DefaultSaveOptions"><prop oor:name="PowerPointDocument" oor:op="fuse"><value>MS PowerPoint 2007 XML</value></prop></item>`,
	}
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, s := range settings {
		_, err := f.WriteString(s + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func checkWifi() (bool, error) {
	cmd := exec.Command("ping", "-c", "2", "cnn.com")
	err := cmd.Run()
	return err == nil, err
}

func playSound() error {
	if runtime.GOOS == "darwin" {
		return exec.Command("afplay", "/System/Library/Sounds/Glass.aiff").Run()
	}
	return exec.Command("aplay", "/usr/share/sounds/alsa/Front_Center.wav").Run()
}

