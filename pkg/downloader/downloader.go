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

	"github.com/tyhal/hytale/pkg/auth"
)

const defaultDownloaderURL = "https://downloader.hytale.com/hytale-downloader.zip"

type Downloader interface {
	Update() error
	BuildAOTCache() error
	GameJarPath() GameJarPath
	GameAotPath() GameAotPath
	GameAssetsPath() GameAssetsPath
}

type (
	GameJarPath    string
	GameAssetsPath string
	GameAotPath    string
)

type downloader struct {
	downloaderDir     string
	gameAssetsZipPath string
	gameServerPath    string
	credentialsPath   string
	allowAOTFail      bool
}

var _ Downloader = (*downloader)(nil)

func (d downloader) GameJarPath() GameJarPath {
	return GameJarPath(filepath.Join(d.gameServerPath, "Server", "HytaleServer.jar"))
}

func (d downloader) GameAotPath() GameAotPath {
	return GameAotPath(filepath.Join(d.gameServerPath, "Server", "HytaleServer.aot"))
}

func (d downloader) GameAssetsPath() GameAssetsPath {
	return GameAssetsPath(filepath.Join(d.gameServerPath, "Assets.zip"))
}

func New(opts ...Options) (Downloader, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	base := filepath.Join(configDir, "tyhal", "hytale")
	credsPath, err := auth.DefaultDownloaderAuthFilePath()
	if err != nil {
		credsPath = filepath.Join(base, "downloader.json")
	}
	d := &downloader{
		downloaderDir:     filepath.Join(base, "downloader"),
		gameAssetsZipPath: filepath.Join(base, "server.zip"),
		gameServerPath:    filepath.Join(base, "server"),
		credentialsPath:   credsPath,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d, nil
}

func (d *downloader) downloaderBinaryPath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(d.downloaderDir, "hytale-downloader-windows-amd64.exe"), nil
	case "linux":
		return filepath.Join(d.downloaderDir, "hytale-downloader-linux-amd64"), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func (d *downloader) ensureDownloader() (string, error) {
	binPath, err := d.downloaderBinaryPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		if err := d.downloadDownloader(); err != nil {
			return "", err
		}
	}
	return binPath, nil
}

func (d *downloader) downloadDownloader() error {
	resp, err := http.Get(defaultDownloaderURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	z, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}
	return extractZip(z, d.downloaderDir)
}

func extractZip(z *zip.Reader, destDir string) error {
	for _, f := range z.File {
		target := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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

func (d *downloader) Update() error {
	bin, err := d.ensureDownloader()
	if err != nil {
		return err
	}

	if d.credentialsPath != "" {
		if _, err := auth.EnsureDownloaderAuth(d.credentialsPath); err != nil {
			return fmt.Errorf("failed to authenticate downloader: %w", err)
		}
	}

	args := []string{"-download-path", d.gameAssetsZipPath}
	if d.credentialsPath != "" {
		args = append(args, "-credentials-path", d.credentialsPath)
	}

	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	z, err := zip.OpenReader(d.gameAssetsZipPath)
	if err != nil {
		return err
	}
	defer z.Close()

	if err := extractZip(&z.Reader, d.gameServerPath); err != nil {
		return err
	}

	return d.BuildAOTCache()
}

func (d *downloader) BuildAOTCache() error {
	aotConfig := filepath.Join(d.gameServerPath, "Server", "HytaleServer.aot.config")
	aotCache := string(d.GameAotPath())
	jarPath := string(d.GameJarPath())

	if _, err := os.Stat(aotConfig); err != nil {
		return nil
	}

	cmd := exec.Command("java",
		"-Xlog:aot",
		"-XX:AOTMode=create",
		fmt.Sprintf("-XX:AOTConfiguration=%s", aotConfig),
		fmt.Sprintf("-XX:AOTCache=%s", aotCache),
		"-jar", jarPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(aotCache)
		if d.allowAOTFail {
			fmt.Printf("Warning: AOT cache generation failed (ignored): %v\n", err)
			return nil
		}
		return fmt.Errorf("AOT cache generation failed: %w", err)
	}
	return nil
}
