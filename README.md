# KindWorks Config (kw-config)

**KindWorks Specific Configuration and Application Setup**

This application is designed to automate the setup and verification of refurbished computers for KindWorks. It provides a user-friendly GUI to install necessary software, configure system settings, and verify hardware functionality. It supports both **Debian-based Linux** (e.g., Ubuntu, Debian) and **macOS**.

## Features

- **Automated Software Installation**:
  - **Linux**: Installs `git`, `alsa-utils`, and `cheese` using `apt-get`.
  - **macOS**: Installs `git` and `ffmpeg` using `Homebrew`.
  - Clones the core `kw-linux` repository and executes its installation script.
  - **Optional**: Choice to install the `KindWorks_Information` repository.
- **System Configuration**:
  - Automatically configures LibreOffice to save in Microsoft Office compatible formats (Word, Excel, PowerPoint).
- **Hardware & Connectivity Dashboard**:
  - **Wi-Fi Test**: Verifies internet connectivity by pinging cnn.com.
  - **Audio Output**: Plays a test sound to verify speakers/headphones (`aplay` on Linux, `afplay` on macOS).
  - **Camera Test**: Launches a camera preview (`cheese`/`ffplay` on Linux, `Photo Booth` on macOS).
  - **Microphone Test**: Records a 5-second clip (`arecord` on Linux, `ffmpeg` on macOS) and plays back for verification.
  - **Visual Status Indicators**: Real-time Green/Red status lights for each hardware component.
- **Real-time Logging**: Displays a high-visibility, "Matrix-style" green terminal log for all background installation tasks.

## Prerequisites

- **Go**: Version 1.16 or higher.
- **Fyne Dependencies**: See the [Fyne setup guide](https://developer.fyne.io/started/) for your OS.
- **Linux**: The application uses `apt-get` for package management.
- **macOS**: The application requires [Homebrew](https://brew.sh/) to be installed.
- **Hardware**: Camera and Audio hardware are required for the verification tests.

## Installation & Building

1. **Clone the project**:
   ```bash
   git clone <repository-url>
   cd kw-linux-config
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Build the application**:
   ```bash
   go build -o kw-config
   ```

## Usage

Run the compiled binary:
```bash
./kw-config
```

Follow the on-screen instructions:
1. Read the KindWorks introduction.
2. Enter the `sudo` password to begin software installation.
3. Monitor the installation progress in the real-time log.
4. Verify system settings and perform hardware tests in the dashboard.
5. Finish the setup once all lights are Green!

## Troubleshooting

- **aplay/arecord not found**: Ensure `alsa-utils` is installed (Linux).
- **brew not found**: Ensure Homebrew is installed (macOS).
- **Camera not opening**: Ensure `cheese` is installed (Linux) or check your camera permissions (macOS).
- **Fyne Thread Panic**: Ensure all UI calls are wrapped in `fyne.Do()` if modifying from a goroutine.

---
*KindWorks: Inspiring action for a kinder world.*
