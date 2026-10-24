package server

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/tyhal/hytale/pkg/downloader"
)

//go:embed hytale.service.tmpl
var serviceTemplateContent string

//go:embed hytale-stop.sh
var stopScriptContent string

var serviceTemplate = template.Must(template.New("hytale.service").Parse(serviceTemplateContent))

type serviceTemplateData struct {
	WorkingDirectory string
	ExecStart        string
}

func renderUnitFile(workingDir, execStart string) (string, error) {
	var buf bytes.Buffer
	err := serviceTemplate.Execute(&buf, serviceTemplateData{
		WorkingDirectory: workingDir,
		ExecStart:        execStart,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render systemd service template: %w", err)
	}
	return buf.String(), nil
}

func EnableService(
	serverPath downloader.GameJarPath,
	assetsPath downloader.GameAssetsPath,
	aotPath downloader.GameAotPath,
	worldPath string,
	opts ...Options,
) error {
	var serverOpts serverOptions
	for _, opt := range opts {
		opt(&serverOpts)
	}

	absWorldPath, err := filepath.Abs(worldPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute world path: %w", err)
	}

	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemctl not found: systemd is required to enable service: %w", err)
	}

	javaBin, err := exec.LookPath("java")
	if err != nil {
		javaBin = "/usr/bin/java"
	}

	args := buildJavaArgs(serverPath, assetsPath, aotPath, serverOpts)
	execStartParts := []string{javaBin}
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\n\"'") {
			execStartParts = append(execStartParts, fmt.Sprintf("%q", arg))
		} else {
			execStartParts = append(execStartParts, arg)
		}
	}
	execStart := strings.Join(execStartParts, " ")

	unitContent, err := renderUnitFile(absWorldPath, execStart)
	if err != nil {
		return err
	}

	const stopScriptPath = "/usr/local/bin/hytale-stop.sh"
	if err := os.MkdirAll(filepath.Dir(stopScriptPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", stopScriptPath, err)
	}
	if err := os.WriteFile(stopScriptPath, []byte(stopScriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write %s: %w", stopScriptPath, err)
	}

	const fifoPath = "/run/hytale/stdin.fifo"
	_ = os.MkdirAll(filepath.Dir(fifoPath), 0755)
	if _, err := os.Stat(fifoPath); os.IsNotExist(err) {
		_ = exec.Command("mkfifo", fifoPath).Run()
	}

	const unitPath = "/etc/systemd/system/hytale.service"
	if err := os.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", unitPath, err)
	}

	for _, cmdArgs := range [][]string{
		{"daemon-reload"},
		{"enable", "hytale.service"},
		{"restart", "hytale.service"},
	} {
		if out, err := exec.Command("systemctl", cmdArgs...).CombinedOutput(); err != nil {
			return fmt.Errorf("failed systemctl %s: %s: %w", cmdArgs[0], string(out), err)
		}
	}

	fmt.Println("Waiting for hytale.service to start and verify health...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastState string
	for {
		select {
		case <-ctx.Done():
			journalOut, _ := exec.Command("journalctl", "-u", "hytale.service", "-n", "20", "--no-pager").CombinedOutput()
			return fmt.Errorf("health check timed out (last state: %s):\n%s", lastState, string(journalOut))
		case <-ticker.C:
			out, _ := exec.Command("systemctl", "is-active", "hytale.service").Output()
			lastState = strings.TrimSpace(string(out))
			if lastState == "active" {
				time.Sleep(2 * time.Second)
				out, _ := exec.Command("systemctl", "is-active", "hytale.service").Output()
				if state := strings.TrimSpace(string(out)); state != "active" {
					journalOut, _ := exec.Command("journalctl", "-u", "hytale.service", "-n", "20", "--no-pager").CombinedOutput()
					return fmt.Errorf("hytale.service stopped unexpectedly (state: %s):\n%s", state, string(journalOut))
				}
				fmt.Println("hytale.service is active and healthy.")
				return nil
			}
			if lastState == "failed" {
				journalOut, _ := exec.Command("journalctl", "-u", "hytale.service", "-n", "20", "--no-pager").CombinedOutput()
				return fmt.Errorf("hytale.service failed to start:\n%s", string(journalOut))
			}
		}
	}
}
