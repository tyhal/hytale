package server

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/tyhal/hytale/pkg/downloader"
)

func buildJavaArgs(
	serverPath downloader.GameJarPath,
	assetsPath downloader.GameAssetsPath,
	aotPath downloader.GameAotPath,
	serverOpts serverOptions,
) []string {
	args := []string{
		"-XX:MaxRAMPercentage=80.0",
		"-XX:+UseG1GC",
		"-XX:+UseStringDeduplication",
	}
	if aotPath != "" {
		if _, err := os.Stat(string(aotPath)); err == nil {
			args = append(args, fmt.Sprintf("-XX:AOTCache=%s", aotPath))
		}
	}
	if len(serverOpts.javaFlags) > 0 {
		args = append(args, serverOpts.javaFlags...)
	}
	if serverOpts.ipv6 {
		args = append(args, "-Djava.net.preferIPv6Addresses=true")
	}
	args = append(args, "-jar", string(serverPath), "--assets", string(assetsPath))

	if serverOpts.backups != "" {
		args = append(args, "--backup", "--backup-dir", serverOpts.backups)
	}
	if serverOpts.sessionToken != "" {
		args = append(args, "--session-token", string(serverOpts.sessionToken))
	}
	if serverOpts.idToken != "" {
		args = append(args, "--identity-token", string(serverOpts.idToken))
	}
	return args
}

func RunServer(
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

	if serverOpts.service {
		return EnableService(serverPath, assetsPath, aotPath, worldPath, opts...)
	}

	args := buildJavaArgs(serverPath, assetsPath, aotPath, serverOpts)
	cmd := exec.Command("java", args...)
	cmd.Dir = worldPath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if !serverOpts.exec {
		fmt.Println(cmd.String())
		return nil
	}
	return cmd.Run()
}
