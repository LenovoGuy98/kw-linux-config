package main

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	colorGray  = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	colorGreen = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	colorRed   = color.NRGBA{R: 255, G: 0, B: 0, A: 255}

	// Persistent Hardware Status
	wifiStatus  = 0 // 0: Untested, 1: OK, 2: Failed
	soundStatus = 0
	camStatus   = 0
	micStatus   = 0
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("KindWorks Specific Configuration and Application Setup")

	showIntro(myWindow)

	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.ShowAndRun()
}

func showIntro(w fyne.Window) {
	introText := widget.NewLabelWithStyle("KindWorks is dedicated to inspiring action for a kinder world by mobilizing individuals in volunteer service, including providing essential digital tools for those in need. The organization refurbishes used computers and distributes them free of charge to individuals, families, and organizations, aiming to \"level the digital playing field\".",
		fyne.TextAlignLeading, fyne.TextStyle{Bold: false})
	introText.Wrapping = fyne.TextWrapWord

	nextBtn := widget.NewButton("Next", func() {
		showConfiguration(w)
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Introduction", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		introText,
		layout.NewSpacer(),
		nextBtn,
	)
	w.SetContent(content)
}

func showConfiguration(w fyne.Window) {
	passwordLabel := widget.NewLabel("Enter sudo password:")
	passwordEntry := widget.NewPasswordEntry()
	statusLabel := widget.NewLabel("Status: Ready to start configuration")
	progress := widget.NewProgressBar()
	progress.Hide()

	infoRepoCheck := widget.NewCheck("Install KindWorks Information (Optional)", nil)

	logOutput := widget.NewRichText()
	logOutput.ExtendBaseWidget(logOutput)
	
	logScroll := container.NewScroll(logOutput)
	logScroll.SetMinSize(fyne.NewSize(0, 300))
	logScroll.Hide()

	var startBtn *widget.Button
	
	top := container.NewVBox(
		widget.NewLabelWithStyle("Configuration Start", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		passwordLabel,
		passwordEntry,
		infoRepoCheck,
		statusLabel,
		progress,
	)

	startBtn = widget.NewButton("Start Configuration", func() {
		password := passwordEntry.Text
		if password == "" {
			statusLabel.SetText("Error: Password is required")
			return
		}

		progress.Show()
		logScroll.Show()
		startBtn.Disable()
		passwordEntry.Disable()
		infoRepoCheck.Disable()
		
		w.Content().Refresh()

		var logLines []string
		var logMutex sync.Mutex
		
		go func() {
			var err error
			if runtime.GOOS == "darwin" {
				fyne.Do(func() { statusLabel.SetText("Status: Updating Homebrew and installing git, ffmpeg...") })
				_, err = runCommand(password, "brew", "update")
				if err == nil {
					_, err = runCommand(password, "brew", "install", "git", "ffmpeg")
				}
			} else {
				fyne.Do(func() { statusLabel.SetText("Status: Installing git, alsa-utils, cheese...") })
				_, err = runCommand(password, "apt-get", "update")
				if err == nil {
					_, err = runCommand(password, "apt-get", "install", "-y", "git", "alsa-utils", "cheese")
				}
			}

			if err != nil {
				fyne.Do(func() {
					statusLabel.SetText(fmt.Sprintf("Error installing packages: %v", err))
					startBtn.Enable()
					passwordEntry.Enable()
					infoRepoCheck.Enable()
				})
				return
			}
			fyne.Do(func() { progress.SetValue(0.10) })

			if runtime.GOOS != "darwin" {
				fyne.Do(func() { statusLabel.SetText("Status: Downloading kw-linux GitHub repo...") })
				_, err = cloneRepo(password)
				if err != nil {
					fyne.Do(func() {
						statusLabel.SetText(fmt.Sprintf("Error cloning kw-linux repo: %v", err))
						startBtn.Enable()
						passwordEntry.Enable()
						infoRepoCheck.Enable()
					})
					return
				}
			}
			fyne.Do(func() { progress.SetValue(0.20) })

			if infoRepoCheck.Checked {
				fyne.Do(func() { statusLabel.SetText("Status: Downloading KindWorks Information repo...") })
				_, err = cloneInfoRepo(password)
				if err != nil {
					fyne.Do(func() {
						statusLabel.SetText(fmt.Sprintf("Error cloning Information repo: %v", err))
						startBtn.Enable()
						passwordEntry.Enable()
						infoRepoCheck.Enable()
					})
					return
				}
				fyne.Do(func() { progress.SetValue(0.30) })
			}

			fyne.Do(func() { statusLabel.SetText("Status: Running install scripts...") })
			outputChan := make(chan string, 100)
			
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				lastText := ""
				for {
					select {
					case line, ok := <-outputChan:
						logMutex.Lock()
						if ok {
							logLines = append(logLines, line)
							if len(logLines) > 1000 {
								logLines = logLines[len(logLines)-1000:]
							}
						}
						logMutex.Unlock()
						if !ok {
							return
						}
					case <-ticker.C:
						logMutex.Lock()
						currentText := strings.Join(logLines, "\n")
						logMutex.Unlock()
						
						if lastText != currentText {
							fyne.Do(func() {
								logOutput.Segments = []widget.RichTextSegment{
									&widget.TextSegment{
										Style: widget.RichTextStyle{
											ColorName: theme.ColorNameSuccess,
											SizeName:  theme.SizeNameSubHeadingText,
											TextStyle: fyne.TextStyle{Monospace: true},
										},
										Text: currentText,
									},
								}
								logOutput.Refresh()
								logScroll.ScrollToBottom()
							})
							lastText = currentText
						}
					}
				}
			}()

			// Core Apps
			if runtime.GOOS != "darwin" {
				err = runInstallApps(password, outputChan)
				if err != nil {
					fyne.Do(func() {
						statusLabel.SetText(fmt.Sprintf("Error running kw-linux install: %v", err))
						startBtn.Enable()
						passwordEntry.Enable()
						infoRepoCheck.Enable()
					})
					close(outputChan)
					wg.Wait()
					return
				}
			} else {
				outputChan <- "Skipping kw-linux install scripts on macOS...\n"
			}
			fyne.Do(func() { progress.SetValue(0.70) })

			// Optional Info
			if infoRepoCheck.Checked {
				outputChan <- "\n--- INSTALLING KINDWORKS INFORMATION ---\n"
				err = runInstallInfo(password, outputChan)
				if err != nil {
					fyne.Do(func() {
						statusLabel.SetText(fmt.Sprintf("Error running Info install: %v", err))
						startBtn.Enable()
						passwordEntry.Enable()
						infoRepoCheck.Enable()
					})
					close(outputChan)
					wg.Wait()
					return
				}
			}
			fyne.Do(func() { progress.SetValue(0.90) })

			close(outputChan)
			wg.Wait()

			fyne.Do(func() {
				progress.SetValue(1.0)
				statusLabel.SetText("Status: Configuration complete!")

				w.SetContent(container.NewVBox(
					widget.NewLabelWithStyle("Configuration Finished", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
					widget.NewLabel("All applications and information have been installed."),
					widget.NewButton("Continue to Settings", func() {
						showSettings(w)
					}),
				))
			})
		}()
	})

	content := container.NewBorder(top, startBtn, nil, nil, logScroll)
	w.SetContent(content)
}

func showSettings(w fyne.Window) {
	statusLabel := widget.NewLabel("Updating LibreOffice settings to Microsoft Office compatibility...")
	
	go func() {
		time.Sleep(1 * time.Second)
		err := configureLibreOffice()
		fyne.Do(func() {
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error configuring LibreOffice: %v", err))
			} else {
				statusLabel.SetText("LibreOffice settings updated successfully.")
			}
		})
	}()

	nextBtn := widget.NewButton("Continue to Hardware Check", func() {
		showHardwareCheck(w)
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Check and Change Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusLabel,
		nextBtn,
	)
	w.SetContent(content)
}

func createStatusLight(status int) fyne.CanvasObject {
	rect := canvas.NewRectangle(colorGray)
	if status == 1 {
		rect.FillColor = colorGreen
	} else if status == 2 {
		rect.FillColor = colorRed
	}
	rect.SetMinSize(fyne.NewSize(24, 24))
	return container.NewGridWrap(fyne.NewSize(24, 24), rect)
}

func showHardwareCheck(w fyne.Window) {
	title := widget.NewLabelWithStyle("Main Hardware and Connection Tests", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	
	wifiLight := createStatusLight(wifiStatus)
	soundLight := createStatusLight(soundStatus)
	camLight := createStatusLight(camStatus)
	micLight := createStatusLight(micStatus)

	wifiBtn := widget.NewButton("Test Wi-Fi", func() {
		ok, _ := checkWifi()
		if ok {
			wifiStatus = 1
		} else {
			wifiStatus = 2
		}
		showHardwareCheck(w)
	})

	soundBtn := widget.NewButton("Test Sound", func() {
		err := playSound()
		if err != nil {
			soundStatus = 2
			showHardwareCheck(w)
			return
		}
		
		d := dialog.NewConfirm("Sound Test", "Did you hear the sound?", func(ok bool) {
			if ok {
				soundStatus = 1
			} else {
				soundStatus = 2
			}
			showHardwareCheck(w)
		}, w)
		d.Show()
	})

	camBtn := widget.NewButton("Test Camera", func() {
		showCameraTest(w)
	})

	micBtn := widget.NewButton("Test Microphone", func() {
		showMicrophoneTest(w)
	})

	nextBtn := widget.NewButton("Finish Setup", func() {
		showFinish(w)
	})

	grid := container.New(layout.NewFormLayout(),
		widget.NewLabel("Wi-Fi Connection:"), container.NewHBox(wifiBtn, wifiLight),
		widget.NewLabel("Audio Output:"), container.NewHBox(soundBtn, soundLight),
		widget.NewLabel("Video Input:"), container.NewHBox(camBtn, camLight),
		widget.NewLabel("Audio Input:"), container.NewHBox(micBtn, micLight),
	)

	content := container.NewVBox(
		title,
		layout.NewSpacer(),
		grid,
		layout.NewSpacer(),
		nextBtn,
	)
	w.SetContent(content)
}

func showCameraTest(w fyne.Window) {
	var camCmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		camCmd = exec.Command("open", "-a", "Photo Booth")
	} else if _, err := exec.LookPath("cheese"); err == nil {
		camCmd = exec.Command("cheese")
	} else {
		camCmd = exec.Command("ffplay", "/dev/video0")
	}

	go func() {
		camCmd.Run()
	}()

	d := dialog.NewConfirm("Camera Test", "Does the camera work?", func(ok bool) {
		if camCmd.Process != nil && runtime.GOOS != "darwin" {
			camCmd.Process.Kill()
		}
		if ok {
			camStatus = 1
		} else {
			camStatus = 2
		}
		showHardwareCheck(w)
	}, w)
	d.Show()
}

func showMicrophoneTest(w fyne.Window) {
	statusLabel := widget.NewLabel("Testing Microphone...")
	progress := widget.NewProgressBar()
	progress.Hide()
	
	recordBtn := widget.NewButton("Start 5-Second Recording", nil)
	recordBtn.OnTapped = func() {
		recordBtn.Disable()
		progress.Max = 5
		progress.SetValue(0)
		progress.Show()
		
		homeDir, _ := os.UserHomeDir()
		testFile := filepath.Join(homeDir, "test_mic.wav")
		
		go func() {
			var cmd *exec.Cmd
			if runtime.GOOS == "darwin" {
				// Assuming sox is installed via brew or use ffmpeg
				cmd = exec.Command("ffmpeg", "-y", "-f", "avfoundation", "-i", ":0", "-t", "5", testFile)
			} else {
				cmd = exec.Command("arecord", "-d", "5", "-f", "cd", testFile)
			}
			cmd.Start()

			for i := 0; i < 5; i++ {
				fyne.Do(func() {
					statusLabel.SetText(fmt.Sprintf("Recording... %d seconds left", 5-i))
					progress.SetValue(float64(i + 1))
				})
				time.Sleep(1 * time.Second)
			}

			cmd.Wait()
			fyne.Do(func() {
				statusLabel.SetText("Playing back...")
				progress.Hide()
			})
			
			var playCmd *exec.Cmd
			if runtime.GOOS == "darwin" {
				playCmd = exec.Command("afplay", testFile)
			} else {
				playCmd = exec.Command("aplay", testFile)
			}
			playCmd.Run()

			fyne.Do(func() {
				d := dialog.NewConfirm("Microphone Test", "Did you hear your recording?", func(ok bool) {
					if ok {
						micStatus = 1
					} else {
						micStatus = 2
					}
					showHardwareCheck(w)
				}, w)
				d.Show()
			})
		}()
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle("Microphone Test", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusLabel,
		progress,
		recordBtn,
		widget.NewButton("Back to Dashboard", func() {
			showHardwareCheck(w)
		}),
	)
	w.SetContent(content)
}

func showFinish(w fyne.Window) {
	w.SetContent(container.NewVBox(
		widget.NewLabelWithStyle("Setup Complete!", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Thank you for setting up the KindWorks computer."),
		widget.NewButton("Close", func() {
			os.Exit(0)
		}),
	))
}
