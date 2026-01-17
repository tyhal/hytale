// Package downloader relates to usage of https://downloader.hytale.com/hytale-downloader.zip
// TODO create a second implementation purely in Go
// but still keep around the downloader from hytale themselves that only supports certain OS and Arch
package downloader

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/log"
)

var url = "https://downloader.hytale.com/hytale-downloader.zip"

type Downloader interface {
	// Update downloads the latest version of the game assets
	Update() error
	GameJarPath() GameJarPath
	GameAssetsPath() GameAssetsPath
}

type GameJarPath string
type GameAssetsPath string

type downloader struct {
	downloaderDir     string
	gameAssetsZipPath string
	gameServerPath    string
}

func (d downloader) GameJarPath() GameJarPath {
	return GameJarPath(filepath.Join(d.gameServerPath, "Server", "HytaleServer.jar"))
}

func (d downloader) GameAssetsPath() GameAssetsPath {
	return GameAssetsPath(filepath.Join(d.gameServerPath, "Assets.zip"))
}

// Ensure downloader implements Downloader
var _ Downloader = (*downloader)(nil)

// New creates a downloader with defaults
func New() (Downloader, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &downloader{
		// TODO think of a unique name to store various config
		downloaderDir:     filepath.Join(configDir, "tyhal", "hytale", "downloader"),
		gameAssetsZipPath: filepath.Join(configDir, "tyhal", "hytale", "assets.zip"),
		gameServerPath:    filepath.Join(configDir, "tyhal", "hytale", "server"), // Assets.zip, Hytale.jar
	}, nil
}

func (d *downloader) downloaderBinaryPath() (string, error) {
	systemOs := runtime.GOOS
	systemArch := runtime.GOARCH
	var executable string
	switch systemOs + systemArch {
	case "windowsamd64":
		executable = "hytale-downloader-windows-amd64.exe"
	case "linuxamd64":
		executable = "hytale-downloader-linux-amd64"
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", systemOs, systemArch)
	}
	return filepath.Join(d.downloaderDir, executable), nil
}

func logCleanup(err error) {
	if err != nil {
		log.Warn(err)
	}
}

func extract(z *zip.Reader, outpath string) error {

	log.Infof("Extracting %s\n", outpath)
	for _, f := range z.File {
		fpath := filepath.Join(outpath, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// downloadDownloader downloads the downloader so you can download the downloads
func (d downloader) downloadDownloader() error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() { logCleanup(resp.Body.Close()) }()
	log.Info("Downloading")

	b, _ := io.ReadAll(resp.Body)
	br := bytes.NewReader(b)
	z, err := zip.NewReader(br, int64(len(b)))
	if err != nil {
		return err
	}

	return extract(z, d.downloaderDir)
}

func (d *downloader) ensureDownloader() (string, error) {
	downloaderBin, err := d.downloaderBinaryPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(downloaderBin); os.IsNotExist(err) {
		return downloaderBin, d.downloadDownloader()
	}
	log.Info("Checking for downloader updates")
	cmd := exec.Command(downloaderBin, "-check-update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return downloaderBin, cmd.Run()
}

func (d *downloader) Update() error {
	downloaderBin, err := d.ensureDownloader()
	if err != nil {
		return err
	}
	log.Info("Downloading game assets")

	cmd := exec.Command(downloaderBin, "-download-path", d.gameAssetsZipPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	z, err := zip.OpenReader(d.gameAssetsZipPath)
	if err != nil {
		return err
	}
	defer func() { logCleanup(z.Close()) }()

	return extract(&z.Reader, d.gameServerPath)
}
