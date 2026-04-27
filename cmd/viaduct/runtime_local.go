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
	"github.com/eblackrps/viaduct/internal/store"
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
	Recorded  bool                    `json:"recorded" yaml:"recorded"`
	Reachable bool                    `json:"reachable" yaml:"reachable"`
	State     *localRuntimeState      `json:"state,omitempty" yaml:"state,omitempty"`
	Readiness *doctorRuntimeReadiness `json:"readiness,omitempty" yaml:"readiness,omitempty"`
	Message   string                  `json:"message,omitempty" yaml:"message,omitempty"`
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
	Config     doctorConfigReport       `json:"config" yaml:"config"`
	Store      doctorStoreReport        `json:"store" yaml:"store"`
	Auth       doctorAuthReport         `json:"auth" yaml:"auth"`
	Runtime    localRuntimeStatusReport `json:"runtime" yaml:"runtime"`
	Checks     []doctorCheck            `json:"checks" yaml:"checks"`
}

type doctorConfigReport struct {
	Exists               bool   `json:"exists" yaml:"exists"`
	Valid                bool   `json:"valid" yaml:"valid"`
	SourceCount          int    `json:"source_count,omitempty" yaml:"source_count,omitempty"`
	CredentialCount      int    `json:"credential_count,omitempty" yaml:"credential_count,omitempty"`
	PluginCount          int    `json:"plugin_count,omitempty" yaml:"plugin_count,omitempty"`
	StateStoreConfigured bool   `json:"state_store_configured" yaml:"state_store_configured"`
	StateStoreMode       string `json:"state_store_mode,omitempty" yaml:"state_store_mode,omitempty"`
}

type doctorStoreReport struct {
	Backend          string `json:"backend,omitempty" yaml:"backend,omitempty"`
	Persistent       bool   `json:"persistent" yaml:"persistent"`
	SchemaVersion    int    `json:"schema_version,omitempty" yaml:"schema_version,omitempty"`
	TenantCount      int    `json:"tenant_count,omitempty" yaml:"tenant_count,omitempty"`
	ServiceAccounts  int    `json:"service_account_count,omitempty" yaml:"service_account_count,omitempty"`
	InspectionStatus string `json:"inspection_status,omitempty" yaml:"inspection_status,omitempty"`
}

type doctorAuthReport struct {
	AdminKeyConfigured    bool `json:"admin_key_configured" yaml:"admin_key_configured"`
	TenantKeyTenants      int  `json:"tenant_key_tenants,omitempty" yaml:"tenant_key_tenants,omitempty"`
	ServiceAccountKeys    int  `json:"service_account_keys,omitempty" yaml:"service_account_keys,omitempty"`
	SharedAccessReady     bool `json:"shared_access_ready" yaml:"shared_access_ready"`
	LocalOnlyFallbackMode bool `json:"local_only_fallback_mode" yaml:"local_only_fallback_mode"`
}

type doctorRuntimeAbout struct {
	Version              string `json:"version,omitempty" yaml:"version,omitempty"`
	Commit               string `json:"commit,omitempty" yaml:"commit,omitempty"`
	StoreBackend         string `json:"store_backend,omitempty" yaml:"store_backend,omitempty"`
	PersistentStore      bool   `json:"persistent_store" yaml:"persistent_store"`
	LocalOperatorSession bool   `json:"local_operator_session_enabled" yaml:"local_operator_session_enabled"`
}

type doctorRuntimeReadiness struct {
	Ready            bool               `json:"ready" yaml:"ready"`
	Status           string             `json:"status,omitempty" yaml:"status,omitempty"`
	HTTPStatus       int                `json:"http_status,omitempty" yaml:"http_status,omitempty"`
	PoliciesLoaded   bool               `json:"policies_loaded" yaml:"policies_loaded"`
	OpenCircuitCount int                `json:"open_circuit_count,omitempty" yaml:"open_circuit_count,omitempty"`
	Issues           []string           `json:"issues,omitempty" yaml:"issues,omitempty"`
	About            doctorRuntimeAbout `json:"about" yaml:"about"`
	InspectionErr    string             `json:"inspection_error,omitempty" yaml:"inspection_error,omitempty"`
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

	if err := os.MkdirAll(filepath.Dir(expandedConfig), 0o750); err != nil {
		return "", false, fmt.Errorf("create config directory: %w", err)
	}

	payload := fmt.Sprintf(
		"# Generated by `viaduct start` for the local lab workflow.\n# Update this file when you move from the bundled lab into a real environment.\ninsecure: false\n\nsources:\n  kvm:\n    address: %q\n",
		filepath.ToSlash(labDir),
	)
	if err := os.WriteFile(expandedConfig, []byte(payload), 0o600); err != nil {
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
	if err := os.MkdirAll(paths.RuntimeDir, 0o750); err != nil {
		return fmt.Errorf("create runtime directory: %w", err)
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal runtime state: %w", err)
	}
	if err := os.WriteFile(paths.StatePath, payload, 0o600); err != nil {
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

func recordedRuntimeVerified(state *localRuntimeState, timeout time.Duration) (bool, string) {
	if state == nil {
		return false, "No runtime state is recorded."
	}
	if state.PID <= 0 {
		return false, "The recorded process ID is invalid."
	}
	if ok, reason := runtimeProcessMatchesCurrentExecutable(state.PID); !ok {
		return false, reason
	}
	if !runtimeReachable(state, timeout) {
		return false, "The recorded runtime is no longer reachable."
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	about, err := doctorEndpointAbout(ctx, strings.TrimRight(state.BaseURL, "/")+"/api/v1/about")
	if err != nil {
		return false, fmt.Sprintf("The recorded runtime did not return matching build metadata: %v.", err)
	}
	if !about.LocalOperatorSession {
		return false, "The recorded endpoint is not a local runtime session."
	}
	if state.Version != "" && about.Version != "" && state.Version != about.Version {
		return false, fmt.Sprintf("The recorded runtime version is %s, but the endpoint reports %s.", state.Version, about.Version)
	}
	if state.Commit != "" && state.Commit != "none" && about.Commit != "" && about.Commit != state.Commit {
		return false, fmt.Sprintf("The recorded runtime commit is %s, but the endpoint reports %s.", state.Commit, about.Commit)
	}
	return true, "The recorded runtime matches the active process."
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	labCheck := doctorCheck{Name: "lab", Status: "fail"}
	if labDir, err := resolveLabFixtureDir(); err == nil {
		report.LabDir = labDir
		labCheck = doctorCheck{Name: "lab", Status: "pass", Message: fmt.Sprintf("Local lab fixtures found at %s.", labDir)}
	} else {
		labCheck.Message = err.Error()
	}

	cfg, configCheck := inspectDoctorConfig(paths.ConfigPath, report.LabDir)
	report.Config = cfg.report
	report.Checks = append(report.Checks, configCheck, labCheck)

	if detectedWebDir := viaductapi.ResolveDashboardAssetDir(webDir); detectedWebDir != "" {
		report.WebDir = detectedWebDir
		report.Checks = append(report.Checks, doctorCheck{Name: "web", Status: "pass", Message: fmt.Sprintf("Built dashboard assets found at %s.", detectedWebDir)})
	} else {
		report.Checks = append(report.Checks, doctorCheck{Name: "web", Status: "fail", Message: "Built dashboard assets were not found; run `make web-build` from source or use a packaged release bundle."})
	}

	if cfg.validConfig != nil {
		storeReport, authReport, storeCheck, authCheck := inspectDoctorStoreAndAuth(ctx, cfg.validConfig)
		report.Store = storeReport
		report.Auth = authReport
		report.Checks = append(report.Checks, storeCheck, authCheck)
	} else {
		report.Store = doctorStoreReport{InspectionStatus: "skipped"}
		report.Auth = doctorAuthReport{LocalOnlyFallbackMode: true}
		report.Checks = append(
			report.Checks,
			doctorCheck{Name: "store", Status: "warn", Message: "Skipped state-store inspection because the config could not be parsed."},
			doctorCheck{Name: "auth", Status: "warn", Message: "Skipped shared-auth inspection because the config could not be parsed."},
		)
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
		readiness := inspectRecordedRuntime(ctx, runtimeState.BaseURL)
		report.Runtime.Readiness = &readiness
		report.Runtime.Message = readinessSummary(runtimeState.BaseURL, readiness)
		report.Checks = append(report.Checks, readinessCheck(runtimeState.BaseURL, readiness))
	} else {
		report.Runtime.Message = fmt.Sprintf("Recorded runtime state exists for pid %d, but %s is not reachable.", runtimeState.PID, runtimeState.BaseURL)
		report.Checks = append(report.Checks, doctorCheck{Name: "runtime", Status: "warn", Message: report.Runtime.Message})
	}

	return report, nil
}

type inspectedDoctorConfig struct {
	report      doctorConfigReport
	validConfig *appConfig
}

func inspectDoctorConfig(configPath, labDir string) (inspectedDoctorConfig, doctorCheck) {
	if _, err := os.Stat(configPath); err == nil {
		cfg, loadErr := loadAppConfig(configPath)
		if loadErr != nil {
			return inspectedDoctorConfig{}, doctorCheck{
				Name:    "config",
				Status:  "fail",
				Message: fmt.Sprintf("Config file exists at %s, but it could not be parsed: %v", configPath, loadErr),
			}
		}

		report := configSummary(cfg)
		report.Exists = true
		report.Valid = true
		return inspectedDoctorConfig{report: report, validConfig: cfg}, doctorCheck{
			Name:    "config",
			Status:  "pass",
			Message: fmt.Sprintf("Config parsed successfully with %d sources, %d credential refs, and %s state.", report.SourceCount, report.CredentialCount, report.StateStoreMode),
		}
	} else if os.IsNotExist(err) {
		cfg := &appConfig{}
		report := configSummary(cfg)
		report.Exists = false
		report.Valid = false

		message := fmt.Sprintf("Config file is missing; `viaduct start` will generate a local lab config at %s.", configPath)
		if labDir == "" {
			message = fmt.Sprintf("Config file is missing at %s.", configPath)
		}
		return inspectedDoctorConfig{report: report, validConfig: cfg}, doctorCheck{
			Name:    "config",
			Status:  "warn",
			Message: message,
		}
	} else {
		return inspectedDoctorConfig{}, doctorCheck{
			Name:    "config",
			Status:  "fail",
			Message: fmt.Sprintf("Unable to inspect config: %v", err),
		}
	}
}

func configSummary(cfg *appConfig) doctorConfigReport {
	report := doctorConfigReport{}
	if cfg == nil {
		return report
	}

	report.SourceCount = len(cfg.Sources)
	report.CredentialCount = len(cfg.Credentials)
	report.PluginCount = len(cfg.Plugins)
	report.StateStoreConfigured = strings.TrimSpace(cfg.StateStoreDSN) != ""
	report.StateStoreMode = stateStoreModeLabel(cfg)
	return report
}

func inspectDoctorStoreAndAuth(ctx context.Context, cfg *appConfig) (doctorStoreReport, doctorAuthReport, doctorCheck, doctorCheck) {
	storeReport := doctorStoreReport{}
	authReport := doctorAuthReport{
		AdminKeyConfigured: strings.TrimSpace(os.Getenv("VIADUCT_ADMIN_KEY")) != "",
	}

	stateStore, err := openStateStore(ctx, cfg)
	if err != nil {
		storeReport.InspectionStatus = "failed"
		storeCheck := doctorCheck{
			Name:    "store",
			Status:  "fail",
			Message: fmt.Sprintf("Unable to open the configured state store: %v", err),
		}
		authReport.LocalOnlyFallbackMode = !authReport.AdminKeyConfigured
		authCheck := doctorCheck{
			Name:    "auth",
			Status:  "warn",
			Message: "Shared-auth inspection was skipped because the configured state store could not be opened.",
		}
		return storeReport, authReport, storeCheck, authCheck
	}
	defer func() {
		_ = stateStore.Close()
	}()

	storeReport = collectDoctorStoreReport(ctx, stateStore)
	storeCheck := doctorCheck{
		Name:    "store",
		Status:  "pass",
		Message: fmt.Sprintf("State store backend is %s.", firstNonEmpty(storeReport.Backend, "unknown")),
	}
	if !storeReport.Persistent {
		storeCheck.Status = "warn"
		storeCheck.Message = "State store backend is memory; local evaluation is fine, but serious pilots should use PostgreSQL."
	}
	if storeReport.Persistent && storeReport.SchemaVersion <= 0 {
		storeCheck.Status = "fail"
		storeCheck.Message = fmt.Sprintf("Persistent state store %s is reachable, but no schema version was reported.", storeReport.Backend)
	}

	authReport = collectDoctorAuthReport(ctx, stateStore, authReport)
	authCheck := doctorCheck{
		Name:   "auth",
		Status: "pass",
		Message: fmt.Sprintf(
			"Shared access is ready with %d tenant key tenant(s) and %d service account key(s).",
			authReport.TenantKeyTenants,
			authReport.ServiceAccountKeys,
		),
	}
	if !authReport.SharedAccessReady {
		authCheck.Status = "warn"
		authCheck.Message = "No admin, tenant, or service account credentials are configured; loopback local sessions still work, but shared access is not ready."
	}

	return storeReport, authReport, storeCheck, authCheck
}

func collectDoctorStoreReport(ctx context.Context, stateStore store.Store) doctorStoreReport {
	report := doctorStoreReport{InspectionStatus: "ok"}
	if provider, ok := stateStore.(store.DiagnosticsProvider); ok {
		if diagnostics, err := provider.Diagnostics(ctx); err == nil {
			report.Backend = diagnostics.Backend
			report.Persistent = diagnostics.Persistent
			report.SchemaVersion = diagnostics.SchemaVersion
		} else {
			report.InspectionStatus = "degraded"
		}
	}

	if tenants, err := stateStore.ListTenants(ctx); err == nil {
		report.TenantCount = len(tenants)
		for _, tenant := range tenants {
			report.ServiceAccounts += len(tenant.ServiceAccounts)
		}
	} else {
		report.InspectionStatus = "degraded"
	}

	return report
}

func collectDoctorAuthReport(ctx context.Context, stateStore store.Store, report doctorAuthReport) doctorAuthReport {
	tenants, err := stateStore.ListTenants(ctx)
	if err != nil {
		report.LocalOnlyFallbackMode = !report.AdminKeyConfigured
		report.SharedAccessReady = report.AdminKeyConfigured
		return report
	}

	for _, tenant := range tenants {
		if !tenant.Active {
			continue
		}
		if store.HasAPIKeyConfigured(tenant.APIKeyHash, tenant.APIKey) {
			report.TenantKeyTenants++
		}
		for _, account := range tenant.ServiceAccounts {
			if !account.Active {
				continue
			}
			if store.HasAPIKeyConfigured(account.APIKeyHash, account.APIKey) {
				report.ServiceAccountKeys++
			}
		}
	}

	report.SharedAccessReady = report.AdminKeyConfigured || report.TenantKeyTenants > 0 || report.ServiceAccountKeys > 0
	report.LocalOnlyFallbackMode = !report.SharedAccessReady
	return report
}

func inspectRecordedRuntime(ctx context.Context, baseURL string) doctorRuntimeReadiness {
	readiness, readyErr := doctorEndpointReadiness(ctx, strings.TrimRight(baseURL, "/")+"/readyz")
	if readyErr != nil {
		readiness.InspectionErr = readyErr.Error()
	}

	about, err := doctorEndpointAbout(ctx, strings.TrimRight(baseURL, "/")+"/api/v1/about")
	if err == nil {
		readiness.About = about
	} else if readiness.InspectionErr == "" {
		readiness.InspectionErr = err.Error()
	}

	return readiness
}

func doctorEndpointReadiness(ctx context.Context, url string) (doctorRuntimeReadiness, error) {
	type circuitSnapshot struct {
		State string `json:"state"`
	}

	type response struct {
		Status          string            `json:"status"`
		PoliciesLoaded  bool              `json:"policies_loaded"`
		Issues          []string          `json:"issues"`
		CircuitBreakers []circuitSnapshot `json:"circuit_breakers"`
	}

	payload, statusCode, err := doctorGET(ctx, url, &response{})
	readiness := doctorRuntimeReadiness{HTTPStatus: statusCode}
	if err != nil {
		return readiness, err
	}
	status, _ := payload.(*response)
	readiness.Status = strings.TrimSpace(status.Status)
	readiness.PoliciesLoaded = status.PoliciesLoaded
	if len(status.Issues) > 0 {
		readiness.Issues = append([]string(nil), status.Issues...)
	}
	for _, circuit := range status.CircuitBreakers {
		if strings.EqualFold(strings.TrimSpace(circuit.State), "open") {
			readiness.OpenCircuitCount++
		}
	}
	readiness.Ready = statusCode == http.StatusOK && strings.EqualFold(readiness.Status, "ready")
	return readiness, nil
}

func doctorEndpointAbout(ctx context.Context, url string) (doctorRuntimeAbout, error) {
	type response struct {
		Version              string `json:"version"`
		Commit               string `json:"commit"`
		StoreBackend         string `json:"store_backend"`
		PersistentStore      bool   `json:"persistent_store"`
		LocalOperatorSession bool   `json:"local_operator_session_enabled"`
	}

	payload, _, err := doctorGET(ctx, url, &response{})
	if err != nil {
		return doctorRuntimeAbout{}, err
	}
	about, _ := payload.(*response)
	return doctorRuntimeAbout{
		Version:              strings.TrimSpace(about.Version),
		Commit:               strings.TrimSpace(about.Commit),
		StoreBackend:         strings.TrimSpace(about.StoreBackend),
		PersistentStore:      about.PersistentStore,
		LocalOperatorSession: about.LocalOperatorSession,
	}, nil
}

func doctorGET(ctx context.Context, url string, payload any) (any, int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build request for %s: %w", url, err)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("request %s: %w", url, err)
	}
	defer response.Body.Close()

	if payload == nil {
		return nil, response.StatusCode, nil
	}
	if err := json.NewDecoder(response.Body).Decode(payload); err != nil {
		return nil, response.StatusCode, fmt.Errorf("decode %s: %w", url, err)
	}
	return payload, response.StatusCode, nil
}

func readinessSummary(baseURL string, readiness doctorRuntimeReadiness) string {
	if readiness.Ready {
		return fmt.Sprintf(
			"Recorded runtime is ready at %s%s.",
			baseURL,
			runtimeAboutSuffix(readiness.About),
		)
	}
	if readiness.InspectionErr != "" {
		return fmt.Sprintf("Recorded runtime is reachable at %s, but readiness inspection failed: %s", baseURL, readiness.InspectionErr)
	}
	if issues := readinessIssuesSummary(readiness); issues != "" {
		return fmt.Sprintf(
			"Recorded runtime is reachable at %s, but /readyz reported %s (%d): %s.",
			baseURL,
			firstNonEmpty(readiness.Status, "unknown"),
			readiness.HTTPStatus,
			issues,
		)
	}
	return fmt.Sprintf("Recorded runtime is reachable at %s, but /readyz reported %s (%d).", baseURL, firstNonEmpty(readiness.Status, "unknown"), readiness.HTTPStatus)
}

func readinessCheck(baseURL string, readiness doctorRuntimeReadiness) doctorCheck {
	if readiness.Ready {
		return doctorCheck{Name: "runtime", Status: "pass", Message: readinessSummary(baseURL, readiness)}
	}
	return doctorCheck{Name: "runtime", Status: "warn", Message: readinessSummary(baseURL, readiness)}
}

func runtimeAboutSuffix(about doctorRuntimeAbout) string {
	details := make([]string, 0, 3)
	if about.Version != "" {
		details = append(details, fmt.Sprintf(" version %s", about.Version))
	}
	if about.StoreBackend != "" {
		details = append(details, fmt.Sprintf(" store %s", about.StoreBackend))
	}
	if about.LocalOperatorSession {
		details = append(details, " local session available")
	}
	if len(details) == 0 {
		return ""
	}
	return " (" + strings.TrimSpace(strings.Join(details, ",")) + ")"
}

func readinessIssuesSummary(readiness doctorRuntimeReadiness) string {
	if len(readiness.Issues) == 0 {
		return ""
	}
	items := make([]string, 0, len(readiness.Issues))
	for _, issue := range readiness.Issues {
		if trimmed := strings.TrimSpace(issue); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return strings.Join(items, "; ")
}

func stateStoreModeLabel(cfg *appConfig) string {
	if cfg == nil || strings.TrimSpace(cfg.StateStoreDSN) == "" {
		return "memory"
	}
	return "configured external store"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
