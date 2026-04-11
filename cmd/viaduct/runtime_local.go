package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	viaductapi "github.com/eblackrps/viaduct/internal/api"
)

const (
	localRuntimeStateFile = "local-runtime.json"
	localRuntimeLogFile   = "local-runtime.log"
)

type localRuntimePaths struct {
	ConfigPath string
	RuntimeDir string
	StatePath  string
	LogPath    string
}

type localRuntimeState struct {
	PID        int       `json:"pid" yaml:"pid"`
	Detached   bool      `json:"detached" yaml:"detached"`
	Host       string    `json:"host" yaml:"host"`
	Port       int       `json:"port" yaml:"port"`
	BaseURL    string    `json:"base_url" yaml:"base_url"`
	APIURL     string    `json:"api_url" yaml:"api_url"`
	ConfigPath string    `json:"config_path" yaml:"config_path"`
	WebDir     string    `json:"web_dir" yaml:"web_dir"`
	LogPath    string    `json:"log_path,omitempty" yaml:"log_path,omitempty"`
	Mode       string    `json:"mode" yaml:"mode"`
	Version    string    `json:"version" yaml:"version"`
	Commit     string    `json:"commit" yaml:"commit"`
	StartedAt  time.Time `json:"started_at" yaml:"started_at"`
}

type localRuntimeStatusReport struct {
	Recorded  bool               `json:"recorded" yaml:"recorded"`
	Reachable bool               `json:"reachable" yaml:"reachable"`
	State     *localRuntimeState `json:"state,omitempty" yaml:"state,omitempty"`
	Message   string             `json:"message,omitempty" yaml:"message,omitempty"`
}

type doctorCheck struct {
	Name    string `json:"name" yaml:"name"`
	Status  string `json:"status" yaml:"status"`
	Message string `json:"message" yaml:"message"`
}

type doctorReport struct {
	ConfigPath string                   `json:"config_path" yaml:"config_path"`
	BaseURL    string                   `json:"base_url" yaml:"base_url"`
	APIURL     string                   `json:"api_url" yaml:"api_url"`
	WebDir     string                   `json:"web_dir,omitempty" yaml:"web_dir,omitempty"`
	LabDir     string                   `json:"lab_dir,omitempty" yaml:"lab_dir,omitempty"`
	Runtime    localRuntimeStatusReport `json:"runtime" yaml:"runtime"`
	Checks     []doctorCheck            `json:"checks" yaml:"checks"`
}

type localStartContext struct {
	paths         localRuntimePaths
	configCreated bool
	dashboardDir  string
	state         localRuntimeState
}

func prepareLocalRuntime(configFile, webDir, host string, port int, detached bool) (*localStartContext, error) {
	expandedConfig, createdConfig, err := ensureLocalLabConfig(configFile)
	if err != nil {
		return nil, err
	}

	paths, err := resolveLocalRuntimePaths(expandedConfig)
	if err != nil {
		return nil, err
	}

	dashboardDir := viaductapi.ResolveDashboardAssetDir(webDir)
	if dashboardDir == "" {
		return nil, fmt.Errorf("built dashboard assets were not found; run `make web-build` from source or use a packaged release bundle")
	}

	runtimeState, err := readLocalRuntimeState(paths)
	if err != nil {
		return nil, err
	}
	if runtimeState != nil {
		if runtimeReachable(runtimeState, 2*time.Second) {
			return nil, fmt.Errorf("viaduct is already running at %s (pid %d). Use `viaduct status --runtime` or `viaduct stop`", runtimeState.BaseURL, runtimeState.PID)
		}
		_ = clearLocalRuntimeState(paths)
	}

	baseURL := localBaseURL(host, port)
	state := localRuntimeState{
		PID:        os.Getpid(),
		Detached:   detached,
		Host:       strings.TrimSpace(host),
		Port:       port,
		BaseURL:    baseURL,
		APIURL:     strings.TrimRight(baseURL, "/") + "/api/v1/",
		ConfigPath: expandedConfig,
		WebDir:     dashboardDir,
		LogPath:    paths.LogPath,
		Mode:       "local-lab",
		Version:    version,
		Commit:     commit,
		StartedAt:  time.Now().UTC(),
	}

	return &localStartContext{
		paths:         paths,
		configCreated: createdConfig,
		dashboardDir:  dashboardDir,
		state:         state,
	}, nil
}

func resolveLocalRuntimePaths(configFile string) (localRuntimePaths, error) {
	expandedConfig, err := expandPath(configFile)
	if err != nil {
		return localRuntimePaths{}, fmt.Errorf("resolve local runtime paths: %w", err)
	}

	runtimeDir := filepath.Join(filepath.Dir(expandedConfig), "runtime")
	return localRuntimePaths{
		ConfigPath: expandedConfig,
		RuntimeDir: runtimeDir,
		StatePath:  filepath.Join(runtimeDir, localRuntimeStateFile),
		LogPath:    filepath.Join(runtimeDir, localRuntimeLogFile),
	}, nil
}

func ensureLocalLabConfig(configFile string) (string, bool, error) {
	expandedConfig, err := expandPath(configFile)
	if err != nil {
		return "", false, fmt.Errorf("resolve config path: %w", err)
	}

	if _, err := os.Stat(expandedConfig); err == nil {
		return expandedConfig, false, nil
	} else if !os.IsNotExist(err) {
		return "", false, fmt.Errorf("inspect config %s: %w", expandedConfig, err)
	}

	labDir, err := resolveLabFixtureDir()
	if err != nil {
		return "", false, err
	}

	if err := os.MkdirAll(filepath.Dir(expandedConfig), 0o755); err != nil {
		return "", false, fmt.Errorf("create config directory: %w", err)
	}

	payload := fmt.Sprintf(
		"# Generated by `viaduct start` for the local lab workflow.\n# Update this file when you move from the bundled lab into a real environment.\ninsecure: false\n\nsources:\n  kvm:\n    address: %q\n",
		filepath.ToSlash(labDir),
	)
	if err := os.WriteFile(expandedConfig, []byte(payload), 0o644); err != nil {
		return "", false, fmt.Errorf("write generated config %s: %w", expandedConfig, err)
	}

	return expandedConfig, true, nil
}

func resolveLabFixtureDir() (string, error) {
	path, ok := resolveOperatorAsset(filepath.Join("examples", "lab", "kvm"))
	if !ok {
		return "", fmt.Errorf("unable to locate the bundled lab fixtures under examples/lab/kvm; use a release bundle or run from a cloned repository checkout")
	}
	return path, nil
}

func resolveOperatorAsset(relativePath string) (string, bool) {
	trimmed := strings.TrimSpace(relativePath)
	if trimmed == "" {
		return "", false
	}

	candidates := make([]string, 0, 8)
	appendCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			candidates = append(candidates, candidate)
		}
	}

	if executable, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executable)
		appendCandidate(filepath.Join(executableDir, "..", "share", "viaduct", trimmed))
		appendCandidate(filepath.Join(executableDir, "..", "share", trimmed))
		appendCandidate(filepath.Join(executableDir, "..", trimmed))
		appendCandidate(filepath.Join(executableDir, trimmed))
	}
	appendCandidate(trimmed)
	if _, file, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
		appendCandidate(filepath.Join(repoRoot, trimmed))
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			absolute = filepath.Clean(candidate)
		}
		if _, duplicate := seen[absolute]; duplicate {
			continue
		}
		seen[absolute] = struct{}{}
		info, err := os.Stat(absolute)
		if err != nil || !info.IsDir() {
			continue
		}
		return absolute, true
	}

	return "", false
}

func writeLocalRuntimeState(paths localRuntimePaths, state localRuntimeState) error {
	if err := os.MkdirAll(paths.RuntimeDir, 0o755); err != nil {
		return fmt.Errorf("create runtime directory: %w", err)
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal runtime state: %w", err)
	}
	if err := os.WriteFile(paths.StatePath, payload, 0o644); err != nil {
		return fmt.Errorf("write runtime state: %w", err)
	}
	return nil
}

func readLocalRuntimeState(paths localRuntimePaths) (*localRuntimeState, error) {
	payload, err := os.ReadFile(paths.StatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read runtime state: %w", err)
	}

	var state localRuntimeState
	if err := json.Unmarshal(payload, &state); err != nil {
		return nil, fmt.Errorf("decode runtime state: %w", err)
	}
	return &state, nil
}

func clearLocalRuntimeState(paths localRuntimePaths) error {
	if err := os.Remove(paths.StatePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove runtime state: %w", err)
	}
	return nil
}

func runtimeReachable(state *localRuntimeState, timeout time.Duration) bool {
	if state == nil || strings.TrimSpace(state.BaseURL) == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return healthCheck(ctx, strings.TrimRight(state.BaseURL, "/")+"/api/v1/health") == nil
}

func waitForRuntimeHealth(baseURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return healthCheck(ctx, strings.TrimRight(baseURL, "/")+"/api/v1/health")
}

func healthCheck(ctx context.Context, healthURL string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err == nil {
			response, err := client.Do(req)
			if err == nil {
				_ = response.Body.Close()
				if response.StatusCode == http.StatusOK {
					return nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for health %s: %w", healthURL, ctx.Err())
		case <-ticker.C:
		}
	}
}

func localBaseURL(host string, port int) string {
	trimmedHost := strings.Trim(strings.TrimSpace(host), "[]")
	switch trimmedHost {
	case "", "0.0.0.0", "::":
		trimmedHost = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(trimmedHost, strconv.Itoa(port))
}

func openBrowser(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		command = exec.Command("open", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

func isLikelyInteractiveSession() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("CI")), "true") {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func killRuntimeProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := process.Kill(); err != nil {
		return fmt.Errorf("stop process %d: %w", pid, err)
	}
	return nil
}

func collectDoctorReport(configFile, webDir, host string, port int) (doctorReport, error) {
	paths, err := resolveLocalRuntimePaths(configFile)
	if err != nil {
		return doctorReport{}, err
	}

	report := doctorReport{
		ConfigPath: paths.ConfigPath,
		BaseURL:    localBaseURL(host, port),
		APIURL:     strings.TrimRight(localBaseURL(host, port), "/") + "/api/v1/",
	}

	if _, err := os.Stat(paths.ConfigPath); err == nil {
		report.Checks = append(report.Checks, doctorCheck{Name: "config", Status: "pass", Message: fmt.Sprintf("Config file found at %s.", paths.ConfigPath)})
	} else if os.IsNotExist(err) {
		if labDir, labErr := resolveLabFixtureDir(); labErr == nil {
			report.LabDir = labDir
			report.Checks = append(report.Checks, doctorCheck{Name: "config", Status: "warn", Message: fmt.Sprintf("Config file is missing; `viaduct start` will generate a local lab config at %s.", paths.ConfigPath)})
		} else {
			report.Checks = append(report.Checks, doctorCheck{Name: "config", Status: "fail", Message: fmt.Sprintf("Config file is missing and the lab fixtures were not found: %v", labErr)})
		}
	} else {
		report.Checks = append(report.Checks, doctorCheck{Name: "config", Status: "fail", Message: fmt.Sprintf("Unable to inspect config: %v", err)})
	}

	if labDir, err := resolveLabFixtureDir(); err == nil {
		report.LabDir = labDir
		report.Checks = append(report.Checks, doctorCheck{Name: "lab", Status: "pass", Message: fmt.Sprintf("Local lab fixtures found at %s.", labDir)})
	} else {
		report.Checks = append(report.Checks, doctorCheck{Name: "lab", Status: "fail", Message: err.Error()})
	}

	if detectedWebDir := viaductapi.ResolveDashboardAssetDir(webDir); detectedWebDir != "" {
		report.WebDir = detectedWebDir
		report.Checks = append(report.Checks, doctorCheck{Name: "web", Status: "pass", Message: fmt.Sprintf("Built dashboard assets found at %s.", detectedWebDir)})
	} else {
		report.Checks = append(report.Checks, doctorCheck{Name: "web", Status: "fail", Message: "Built dashboard assets were not found; run `make web-build` from source or use a packaged release bundle."})
	}

	runtimeState, err := readLocalRuntimeState(paths)
	if err != nil {
		report.Checks = append(report.Checks, doctorCheck{Name: "runtime", Status: "fail", Message: fmt.Sprintf("Unable to read runtime state: %v", err)})
		return report, nil
	}
	report.Runtime.Recorded = runtimeState != nil
	report.Runtime.State = runtimeState
	if runtimeState == nil {
		report.Runtime.Message = "No detached or recorded local runtime is active."
		report.Checks = append(report.Checks, doctorCheck{Name: "runtime", Status: "warn", Message: report.Runtime.Message})
		return report, nil
	}

	report.Runtime.Reachable = runtimeReachable(runtimeState, 2*time.Second)
	if report.Runtime.Reachable {
		report.Runtime.Message = fmt.Sprintf("Recorded runtime is reachable at %s.", runtimeState.BaseURL)
		report.Checks = append(report.Checks, doctorCheck{Name: "runtime", Status: "pass", Message: report.Runtime.Message})
	} else {
		report.Runtime.Message = fmt.Sprintf("Recorded runtime state exists for pid %d, but %s is not reachable.", runtimeState.PID, runtimeState.BaseURL)
		report.Checks = append(report.Checks, doctorCheck{Name: "runtime", Status: "warn", Message: report.Runtime.Message})
	}

	return report, nil
}
