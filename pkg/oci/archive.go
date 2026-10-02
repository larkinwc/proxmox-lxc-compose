package oci

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// guestPath resolves symlinks in the guest namespace, never in the host namespace.
// Absolute guest symlinks are relative to rootfs, including during post-processing.
func guestPath(rootfs, name string) (string, error) {
	parts := strings.Split(filepath.ToSlash(name), "/")
	resolved := []string{}
	links := 0
	for len(parts) > 0 {
		part := parts[0]
		parts = parts[1:]
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(resolved) == 0 {
				return "", fmt.Errorf("guest path escapes rootfs: %q", name)
			}
			resolved = resolved[:len(resolved)-1]
			continue
		}
		candidate := filepath.Join(append([]string{rootfs}, append(resolved, part)...)...)
		info, err := os.Lstat(candidate)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			links++
			if links > 40 {
				return "", fmt.Errorf("too many guest symlinks: %q", name)
			}
			target, err := os.Readlink(candidate)
			if err != nil {
				return "", err
			}
			if strings.HasPrefix(target, "/") {
				resolved = nil
			}
			parts = append(strings.Split(target, "/"), parts...)
		} else {
			resolved = append(resolved, part)
		}
	}
	return filepath.Join(append([]string{rootfs}, resolved...)...), nil
}

func archiveName(name string) (string, error) {
	if strings.HasPrefix(name, "/") || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	clean := path.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	return clean, nil
}

// extractTar handles each entry through guestPath; archives cannot escape via
// traversal, absolute names, hardlinks, or symlinked parent directories.
func extractTar(reader io.Reader, rootfs string) error {
	tr := tar.NewReader(reader)
	type directory struct {
		name string
		mode os.FileMode
	}
	var dirs []directory
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name, err := archiveName(header.Name)
		if err != nil {
			return err
		}
		if name == "." {
			continue
		}
		parent, err := guestPath(rootfs, path.Dir(name))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(parent, 0755); err != nil {
			return err
		}
		dest := filepath.Join(parent, path.Base(name))
		mode := os.FileMode(header.Mode & 0777)
		if header.Mode&04000 != 0 {
			mode |= os.ModeSetuid
		}
		if header.Mode&02000 != 0 {
			mode |= os.ModeSetgid
		}
		if header.Mode&01000 != 0 {
			mode |= os.ModeSticky
		}
		if header.Typeflag != tar.TypeDir {
			if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		switch header.Typeflag {
		case tar.TypeDir:
			// Resolve existing directory symlinks in the guest namespace.
			dest, err = guestPath(rootfs, name)
			if err != nil {
				return err
			}
			if err = os.MkdirAll(dest, 0755); err != nil {
				return err
			}
			dirs = append(dirs, directory{name, mode})
		case tar.TypeReg:
			f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(f, tr)
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeSymlink:
			if strings.ContainsRune(header.Linkname, 0) {
				return fmt.Errorf("NUL in archive symlink")
			}
			if err := os.Symlink(header.Linkname, dest); err != nil {
				return err
			}
		case tar.TypeLink:
			link, err := archiveName(header.Linkname)
			if err != nil {
				return err
			}
			target, err := guestPath(rootfs, link)
			if err != nil {
				return err
			}
			info, err := os.Stat(target)
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("hardlink target is not a regular file")
			}
			if err := os.Link(target, dest); err != nil {
				return err
			}
		case tar.TypeChar, tar.TypeBlock:
			// LXC populates /dev itself. Never materialize attacker-controlled
			// devices in the host's temporary directory.
			continue
		case tar.TypeFifo:
			return fmt.Errorf("FIFO archive entries are not supported: %q", name)
		default:
			return fmt.Errorf("unsupported archive entry type %d for %q", header.Typeflag, name)
		}
		if os.Geteuid() == 0 {
			if err := os.Lchown(dest, header.Uid, header.Gid); err != nil {
				return err
			}
		}
		if header.Typeflag != tar.TypeSymlink && header.Typeflag != tar.TypeDir {
			if err := os.Chmod(dest, mode); err != nil {
				return err
			}
		}
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		dest, err := guestPath(rootfs, dirs[i].name)
		if err != nil {
			return err
		}
		if err := os.Chmod(dest, dirs[i].mode); err != nil {
			return err
		}
	}
	return nil
}

func extractArchive(archive, rootfs string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	var reader io.Reader = f
	var decompressor *exec.Cmd
	switch {
	case strings.HasSuffix(archive, ".gz"), strings.HasSuffix(archive, ".tgz"):
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader = gz
	case strings.HasSuffix(archive, ".bz2"):
		reader = bzip2.NewReader(f)
	case strings.HasSuffix(archive, ".xz"), strings.HasSuffix(archive, ".zst"), strings.HasSuffix(archive, ".zstd"):
		binary := "xz"
		if !strings.HasSuffix(archive, ".xz") {
			binary = "zstd"
		}
		decompressor = exec.Command(binary, "-dc")
		decompressor.Stdin = f
		decompressor.Stderr = os.Stderr
		pipe, err := decompressor.StdoutPipe()
		if err != nil {
			return err
		}
		if err := decompressor.Start(); err != nil {
			return err
		}
		reader = pipe
	case strings.HasSuffix(archive, ".tar"):
	default:
		return fmt.Errorf("unsupported template compression: %q", archive)
	}
	err = extractTar(reader, rootfs)
	if decompressor != nil {
		if err != nil {
			_ = decompressor.Process.Kill()
		}
		waitErr := decompressor.Wait()
		if err == nil {
			err = waitErr
		}
	}
	return err
}

// ConvertTemplateRuntime derives a new archive without changing the source template.
// Only the runtime wrapper is replaced; distro configuration is left intact.
func ConvertTemplateRuntime(source, output string, runtime RuntimeConfig) (*ConvertResult, error) {
	if err := validateRuntime(runtime, nil); err != nil {
		return nil, err
	}
	work, err := os.MkdirTemp("", "lxc-compose-runtime-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)
	if err := extractArchive(source, work); err != nil {
		return nil, fmt.Errorf("extract template: %w", err)
	}
	wrapper, err := WriteInitWrapper(work, runtime)
	if err != nil {
		return nil, err
	}
	if wrapper == "" {
		// A cleared process must not leave the base image wrapper active.
		wrapperPath, err := guestPath(work, InitWrapperPath)
		if err != nil {
			return nil, err
		}
		if err := os.Remove(wrapperPath); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	if err := packGzip(work, output); err != nil {
		return nil, err
	}
	return &ConvertResult{OutputPath: output, Runtime: runtime, InitWrapperPath: wrapper}, nil
}
