package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	bundleModulePath = "github.com/eblackrps/viaduct-release-bundle"
	bundleGoVersion  = "1.24"
)

type releaseOptions struct {
	Workspace string
	Version   string
	Commit    string
	Date      string
	Binary    string
	WebDir    string
	OutputDir string
}

type releaseManifest struct {
	Name       string    `json:"name"`
	Version    string    `json:"version"`
	Commit     string    `json:"commit"`
	BuiltAt    string    `json:"built_at"`
	PackagedAt time.Time `json:"packaged_at"`
	GOOS       string    `json:"goos"`
	GOARCH     string    `json:"goarch"`
	Binary     string    `json:"binary"`
	WebDir     string    `json:"web_dir"`
	Files      []string  `json:"files"`
}

func main() {
	var options releaseOptions
	flag.StringVar(&options.Workspace, "workspace", ".", "Workspace root to package")
	flag.StringVar(&options.Version, "version", "dev", "Release version")
	flag.StringVar(&options.Commit, "commit", "none", "Release commit")
	flag.StringVar(&options.Date, "date", "unknown", "Binary build timestamp")
	flag.StringVar(&options.Binary, "binary", filepath.Join("bin", "viaduct"), "Path to the built Viaduct binary")
	flag.StringVar(&options.WebDir, "web-dir", filepath.Join("web", "dist"), "Path to the built dashboard assets")
	flag.StringVar(&options.OutputDir, "output-dir", "dist", "Directory that will receive the packaged release")
	flag.Parse()

	if err := packageRelease(options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func packageRelease(options releaseOptions) error {
	workspace, err := filepath.Abs(options.Workspace)
	if err != nil {
		return fmt.Errorf("package release: resolve workspace: %w", err)
	}

	binaryPath := filepath.Join(workspace, filepath.FromSlash(options.Binary))
	if err := requireFile(binaryPath); err != nil {
		return fmt.Errorf("package release: %w", err)
	}

	webDir := filepath.Join(workspace, filepath.FromSlash(options.WebDir))
	if err := requireDir(webDir); err != nil {
		return fmt.Errorf("package release: %w", err)
	}

	outputDir := filepath.Join(workspace, filepath.FromSlash(options.OutputDir))
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("package release: create output dir: %w", err)
	}

	packageName := fmt.Sprintf("viaduct_%s_%s_%s", sanitizeVersion(options.Version), runtime.GOOS, runtime.GOARCH)
	bundleDir := filepath.Join(outputDir, packageName)
	if err := os.RemoveAll(bundleDir); err != nil {
		return fmt.Errorf("package release: reset bundle dir: %w", err)
	}
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return fmt.Errorf("package release: create bundle dir: %w", err)
	}

	copyJobs := []struct {
		source string
		target string
		dir    bool
	}{
		{source: filepath.Join(workspace, "README.md"), target: filepath.Join(bundleDir, "README.md")},
		{source: filepath.Join(workspace, "LICENSE"), target: filepath.Join(bundleDir, "LICENSE")},
		{source: filepath.Join(workspace, "CHANGELOG.md"), target: filepath.Join(bundleDir, "CHANGELOG.md")},
		{source: filepath.Join(workspace, "CODE_OF_CONDUCT.md"), target: filepath.Join(bundleDir, "CODE_OF_CONDUCT.md")},
		{source: filepath.Join(workspace, "CONTRIBUTING.md"), target: filepath.Join(bundleDir, "CONTRIBUTING.md")},
		{source: filepath.Join(workspace, "INSTALL.md"), target: filepath.Join(bundleDir, "INSTALL.md")},
		{source: filepath.Join(workspace, "QUICKSTART.md"), target: filepath.Join(bundleDir, "QUICKSTART.md")},
		{source: filepath.Join(workspace, "RELEASE.md"), target: filepath.Join(bundleDir, "RELEASE.md")},
		{source: filepath.Join(workspace, "SECURITY.md"), target: filepath.Join(bundleDir, "SECURITY.md")},
		{source: filepath.Join(workspace, "SUPPORT.md"), target: filepath.Join(bundleDir, "SUPPORT.md")},
		{source: filepath.Join(workspace, "UPGRADE.md"), target: filepath.Join(bundleDir, "UPGRADE.md")},
		{source: filepath.Join(workspace, ".env.example"), target: filepath.Join(bundleDir, ".env.example")},
		{source: filepath.Join(workspace, "scripts", "install.sh"), target: filepath.Join(bundleDir, "install.sh")},
		{source: filepath.Join(workspace, "scripts", "install.ps1"), target: filepath.Join(bundleDir, "install.ps1")},
		{source: binaryPath, target: filepath.Join(bundleDir, "bin", filepath.Base(binaryPath))},
		{source: webDir, target: filepath.Join(bundleDir, "web"), dir: true},
		{source: filepath.Join(workspace, "docs"), target: filepath.Join(bundleDir, "docs"), dir: true},
		{source: filepath.Join(workspace, "configs"), target: filepath.Join(bundleDir, "configs"), dir: true},
		{source: filepath.Join(workspace, "examples"), target: filepath.Join(bundleDir, "examples"), dir: true},
	}

	for _, job := range copyJobs {
		if job.dir {
			if err := copyDir(job.source, job.target); err != nil {
				return fmt.Errorf("package release: copy %s: %w", job.source, err)
			}
			continue
		}
		if err := copyFile(job.source, job.target); err != nil {
			return fmt.Errorf("package release: copy %s: %w", job.source, err)
		}
	}
	if err := writeBundleModuleMarker(bundleDir); err != nil {
		return fmt.Errorf("package release: write module marker: %w", err)
	}
	if err := writeExampleModuleMarkers(filepath.Join(bundleDir, "examples")); err != nil {
		return fmt.Errorf("package release: write example module markers: %w", err)
	}

	files, err := collectFiles(bundleDir)
	if err != nil {
		return fmt.Errorf("package release: list bundle files: %w", err)
	}

	manifest := releaseManifest{
		Name:       "Viaduct",
		Version:    options.Version,
		Commit:     options.Commit,
		BuiltAt:    options.Date,
		PackagedAt: time.Now().UTC(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		Binary:     filepath.ToSlash(filepath.Join("bin", filepath.Base(binaryPath))),
		WebDir:     "web",
		Files:      files,
	}

	if err := writeJSON(filepath.Join(bundleDir, "release-manifest.json"), manifest); err != nil {
		return fmt.Errorf("package release: write manifest: %w", err)
	}
	if err := writeChecksums(bundleDir); err != nil {
		return fmt.Errorf("package release: write checksums: %w", err)
	}

	archivePath := bundleDir + ".zip"
	if err := os.RemoveAll(archivePath); err != nil {
		return fmt.Errorf("package release: reset archive: %w", err)
	}
	if err := zipDir(bundleDir, archivePath); err != nil {
		return fmt.Errorf("package release: create archive: %w", err)
	}

	return nil
}

func sanitizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "dev"
	}
	version = strings.ReplaceAll(version, "/", "-")
	version = strings.ReplaceAll(version, "\\", "-")
	version = strings.ReplaceAll(version, " ", "-")
	return version
}

func requireFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required file %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("required file %s: is a directory", path)
	}
	return nil
}

func requireDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required directory %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("required directory %s: is a file", path)
	}
	return nil
}

func copyFile(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	return os.Chmod(target, info.Mode())
}

func copyDir(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(target, 0o755)
		}

		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		return copyFile(path, destination)
	})
}

func collectFiles(root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func writeJSON(path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeBundleModuleMarker(bundleDir string) error {
	return writeModuleMarker(bundleDir, bundleModulePath)
}

func writeExampleModuleMarkers(examplesRoot string) error {
	return filepath.WalkDir(examplesRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		dir := filepath.Dir(path)
		moduleMarker := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(moduleMarker); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}

		relative, err := filepath.Rel(examplesRoot, dir)
		if err != nil {
			return err
		}

		modulePath := fmt.Sprintf("%s/examples/%s", bundleModulePath, filepath.ToSlash(relative))
		return writeModuleMarker(dir, modulePath)
	})
}

func writeModuleMarker(dir, modulePath string) error {
	contents := fmt.Sprintf("module %s\n\ngo %s\n", modulePath, bundleGoVersion)
	return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(contents), 0o644)
}

func writeChecksums(root string) error {
	files, err := collectFiles(root)
	if err != nil {
		return err
	}

	lines := make([]string, 0, len(files))
	for _, relative := range files {
		if relative == "SHA256SUMS.txt" {
			continue
		}

		sum, err := checksumFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return err
		}
		lines = append(lines, fmt.Sprintf("%s  %s", sum, relative))
	}
	sort.Strings(lines)
	return os.WriteFile(filepath.Join(root, "SHA256SUMS.txt"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func checksumFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func zipDir(sourceDir, archivePath string) error {
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer archiveFile.Close()

	writer := zip.NewWriter(archiveFile)
	defer writer.Close()

	baseName := filepath.Base(sourceDir)
	return filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(filepath.Join(baseName, relative))
		header.Method = zip.Deflate

		target, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		source, err := os.Open(path)
		if err != nil {
			return err
		}
		defer source.Close()

		_, err = io.Copy(target, source)
		return err
	})
}
