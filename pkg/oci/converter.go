package oci

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// dockerExec is the indirection point for running docker; tests override it.
var dockerExec = exec.Command

// ConvertResult describes the outcome of an OCI->LXC conversion.
type ConvertResult struct {
	// OutputPath is the gzip-compressed LXC template tarball.
	OutputPath string
	// Entrypoint and Command are the image's declared process, captured so the
	// caller can reproduce it under LXC (e.g. via an init command).
	Entrypoint []string
	Command    []string
	// InitWrapperPath is the in-container path of the generated init wrapper
	// (empty if the image declared no command).
	InitWrapperPath string
	// PostProcess summarizes the rootfs compatibility fixes applied.
	PostProcess PostProcessResult
}

// imageInspect is the subset of `docker inspect` output we consume.
type imageInspect struct {
	Config struct {
		Entrypoint []string `json:"Entrypoint"`
		Cmd        []string `json:"Cmd"`
	} `json:"Config"`
}

// inspectImage returns the entrypoint and command declared by an image.
func inspectImage(image string) (entrypoint, cmd []string, err error) {
	out, err := dockerExec("docker", "inspect", image).Output()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to inspect image %q: %w", image, err)
	}
	var inspects []imageInspect
	if err := json.Unmarshal(out, &inspects); err != nil {
		return nil, nil, fmt.Errorf("failed to parse docker inspect output: %w", err)
	}
	if len(inspects) == 0 {
		return nil, nil, fmt.Errorf("docker inspect returned no entries for %q", image)
	}
	return inspects[0].Config.Entrypoint, inspects[0].Config.Cmd, nil
}

// ConvertOCIToLXC converts an OCI image into a Proxmox-compatible, gzip
// compressed LXC template. It pulls the image, exports its filesystem, applies
// OCI->LXC compatibility fixes (log symlinks, networking, init wrapper), and
// repacks the result. The image's entrypoint/command are captured in the
// returned ConvertResult so callers can reproduce the runtime under LXC.
func ConvertOCIToLXC(imageName, outputPath string) (*ConvertResult, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker is not installed: %w", err)
	}

	// Proxmox/pveam expect a compressed archive; enforce a .tar.gz name.
	if !strings.HasSuffix(outputPath, ".tar.gz") && !strings.HasSuffix(outputPath, ".tgz") {
		outputPath += ".tar.gz"
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Pull the image.
	pull := dockerExec("docker", "pull", imageName)
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		return nil, fmt.Errorf("failed to pull image %q: %w", imageName, err)
	}

	// Capture the image's declared entrypoint/command.
	entrypoint, cmd, err := inspectImage(imageName)
	if err != nil {
		return nil, err
	}

	// Create a container so we can export a flattened filesystem.
	createOut, err := dockerExec("docker", "create", imageName).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	containerID := strings.TrimSpace(string(createOut))
	if containerID == "" {
		return nil, fmt.Errorf("docker create returned empty container id")
	}
	defer func() {
		_ = dockerExec("docker", "rm", "-f", containerID).Run()
	}()

	// Work in a temp directory: export -> extract -> post-process -> repack.
	workDir, err := os.MkdirTemp("", "lxc-compose-convert-")
	if err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	rootfs := filepath.Join(workDir, "rootfs")
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rootfs dir: %w", err)
	}

	// Stream `docker export` straight into `tar -x` to avoid a large temp file.
	if err := exportAndExtract(containerID, rootfs); err != nil {
		return nil, err
	}

	// Apply OCI->LXC compatibility fixes.
	post, err := PostProcessRootfs(rootfs, entrypoint, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to post-process rootfs: %w", err)
	}

	// Repack as a real gzip tarball that Proxmox accepts.
	if err := packGzip(rootfs, outputPath); err != nil {
		return nil, err
	}

	return &ConvertResult{
		OutputPath:      outputPath,
		Entrypoint:      entrypoint,
		Command:         cmd,
		InitWrapperPath: post.InitWrapperPath,
		PostProcess:     post,
	}, nil
}

// exportAndExtract pipes `docker export <id>` into `tar -x` rooted at dest.
func exportAndExtract(containerID, dest string) error {
	export := dockerExec("docker", "export", containerID)
	untar := dockerExec("tar", "-x", "-C", dest)

	pipe, err := export.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe docker export: %w", err)
	}
	untar.Stdin = pipe
	export.Stderr = os.Stderr
	untar.Stderr = os.Stderr

	if err := untar.Start(); err != nil {
		return fmt.Errorf("failed to start tar extract: %w", err)
	}
	if err := export.Run(); err != nil {
		return fmt.Errorf("failed to export container: %w", err)
	}
	if err := untar.Wait(); err != nil {
		return fmt.Errorf("failed to extract container filesystem: %w", err)
	}
	return nil
}

// packGzip creates a gzip-compressed tarball of the rootfs contents. The tar is
// created from inside rootfs so paths are stored relative (./bin, ./etc, ...),
// which is what Proxmox expects for a CT template.
func packGzip(rootfs, outputPath string) error {
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	// Use the system tar with gzip for correct sparse/xattr handling.
	tarCmd := dockerExec("tar", "-cz", "-C", rootfs, ".")
	tarCmd.Stdout = out
	tarCmd.Stderr = os.Stderr
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("failed to pack template: %w", err)
	}
	return nil
}
