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
	// Runtime records the effective configuration, including image env and cwd.
	Runtime RuntimeConfig
	// InitWrapperPath is the actual generated guest init path, empty when the
	// effective runtime needs no wrapper and boots distro init directly.
	InitWrapperPath string
	// PostProcess summarizes the rootfs compatibility fixes applied.
	PostProcess PostProcessResult
}

// imageInspect is the subset of `docker inspect` output we consume.
type imageInspect struct {
	Config struct {
		Entrypoint []string `json:"Entrypoint"`
		Cmd        []string `json:"Cmd"`
		Env        []string `json:"Env"`
		WorkingDir string   `json:"WorkingDir"`
	} `json:"Config"`
}

// inspectImage returns the image's default process, environment and directory.
func inspectImage(image string) (RuntimeConfig, error) {
	out, err := dockerExec("docker", "inspect", image).Output()
	if err != nil {
		return RuntimeConfig{}, fmt.Errorf("failed to inspect image %q: %w", image, err)
	}
	var inspects []imageInspect
	if err := json.Unmarshal(out, &inspects); err != nil {
		return RuntimeConfig{}, fmt.Errorf("failed to parse docker inspect output: %w", err)
	}
	if len(inspects) == 0 {
		return RuntimeConfig{}, fmt.Errorf("docker inspect returned no entries for %q", image)
	}
	config := inspects[0].Config
	runtime := RuntimeConfig{Entrypoint: config.Entrypoint, Command: config.Cmd, WorkingDir: config.WorkingDir, Environment: map[string]string{}}
	for _, variable := range config.Env {
		key, value, ok := strings.Cut(variable, "=")
		if !ok {
			return RuntimeConfig{}, fmt.Errorf("invalid image environment entry %q", variable)
		}
		runtime.Environment[key] = value
	}
	if err := validateRuntime(runtime, nil); err != nil {
		return RuntimeConfig{}, err
	}
	return runtime, nil
}

// ConvertOCIToLXC converts an image and its effective runtime into a gzip LXC
// template. Overrides are validated before pulling or writing to the host.
func ConvertOCIToLXC(imageName, outputPath string, overrides RuntimeOverrides) (*ConvertResult, error) {
	if err := overrides.Validate(); err != nil {
		return nil, err
	}
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

	// Capture and resolve image defaults before creating the rootfs.
	defaults, err := inspectImage(imageName)
	if err != nil {
		return nil, err
	}
	runtime, err := ResolveRuntime(defaults, overrides)
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
	post, err := PostProcessRootfs(rootfs, runtime)
	if err != nil {
		return nil, fmt.Errorf("failed to post-process rootfs: %w", err)
	}

	// Repack as a real gzip tarball that Proxmox accepts.
	if err := packGzip(rootfs, outputPath); err != nil {
		return nil, err
	}

	return &ConvertResult{
		OutputPath:      outputPath,
		Runtime:         runtime,
		InitWrapperPath: post.InitWrapperPath,
		PostProcess:     post,
	}, nil
}

// exportAndExtract confines even adversarial exported archive paths to dest.
func exportAndExtract(containerID, dest string) error {
	export := dockerExec("docker", "export", containerID)
	pipe, err := export.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe docker export: %w", err)
	}
	export.Stderr = os.Stderr
	if err := export.Start(); err != nil {
		return fmt.Errorf("start docker export: %w", err)
	}
	err = extractTar(pipe, dest)
	if err != nil {
		_ = export.Process.Kill()
	}
	waitErr := export.Wait()
	if err != nil {
		return fmt.Errorf("extract container filesystem: %w", err)
	}
	if waitErr != nil {
		return fmt.Errorf("export container: %w", waitErr)
	}
	return nil
}

// packGzip creates a gzip-compressed tarball of the rootfs contents. The tar is
// created from inside rootfs so paths are stored relative (./bin, ./etc, ...),
// which is what Proxmox expects for a CT template.
func packGzip(rootfs, outputPath string) error {
	// Write to a temp file first and rename on success so a failed pack never
	// leaves a partial template that cache-existence checks would trust.
	// Generated archives contain runtime environment values and must remain
	// owner-readable only. CreateTemp also avoids following a stale .tmp link.
	out, err := os.CreateTemp(filepath.Dir(outputPath), ".lxc-compose-template-")
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	tmpPath := out.Name()
	defer func() {
		_ = out.Close()
		_ = os.Remove(tmpPath) // no-op once renamed away
	}()

	// Use the system tar with gzip for correct sparse/xattr handling.
	tarCmd := dockerExec("tar", "-cz", "-C", rootfs, ".")
	tarCmd.Stdout = out
	tarCmd.Stderr = os.Stderr
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("failed to pack template: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to finalize output file: %w", err)
	}
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("failed to move packed template into place: %w", err)
	}
	return nil
}
