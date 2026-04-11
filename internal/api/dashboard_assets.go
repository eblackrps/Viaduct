package api

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolveDashboardAssetDir returns the first built dashboard asset directory that matches the supplied preference or Viaduct's packaged layouts.
func ResolveDashboardAssetDir(preferred string) string {
	candidates := make([]string, 0, 10)
	seen := make(map[string]struct{}, 10)
	appendCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		candidates = append(candidates, candidate)
	}

	appendCandidate(preferred)
	appendCandidate(os.Getenv("VIADUCT_WEB_DIR"))

	if executable, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executable)
		appendCandidate(filepath.Join(executableDir, "..", "share", "viaduct", "web"))
		appendCandidate(filepath.Join(executableDir, "..", "share", "web"))
		appendCandidate(filepath.Join(executableDir, "..", "web"))
		appendCandidate(filepath.Join(executableDir, "web"))
		appendCandidate(filepath.Join(executableDir, "..", "web", "dist"))
	}

	appendCandidate(filepath.Join("web", "dist"))
	appendCandidate("web")
	appendCandidate(filepath.Join(string(filepath.Separator), "opt", "viaduct", "web"))

	if _, file, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
		appendCandidate(filepath.Join(repoRoot, "web", "dist"))
	}

	for _, candidate := range candidates {
		resolved, ok := resolveExistingDirCandidate(candidate)
		if !ok {
			continue
		}
		if _, duplicate := seen[resolved]; duplicate {
			continue
		}
		seen[resolved] = struct{}{}
		if isBuiltDashboardDir(resolved) {
			return resolved
		}
	}

	return ""
}

func resolveDashboardAssetDir(preferred string) string {
	return ResolveDashboardAssetDir(preferred)
}

func resolveExistingDirCandidate(candidate string) (string, bool) {
	info, err := os.Stat(candidate)
	if err != nil || !info.IsDir() {
		return "", false
	}
	resolved, err := filepath.Abs(candidate)
	if err != nil {
		return filepath.Clean(candidate), true
	}
	return resolved, true
}

func isBuiltDashboardDir(root string) bool {
	indexInfo, err := os.Stat(filepath.Join(root, "index.html"))
	if err != nil || indexInfo.IsDir() {
		return false
	}

	assetsInfo, err := os.Stat(filepath.Join(root, "assets"))
	return err == nil && assetsInfo.IsDir()
}

func (s *Server) dashboardHandler() http.Handler {
	if s == nil || !isBuiltDashboardDir(s.dashboardDir) {
		return nil
	}

	indexPath := filepath.Join(s.dashboardDir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		cleanedPath := path.Clean("/" + strings.TrimSpace(r.URL.Path))
		if cleanedPath == "." {
			cleanedPath = "/"
		}
		relativePath := strings.TrimPrefix(cleanedPath, "/")
		if cleanedPath == "/" {
			applyDashboardCacheHeaders(w, cleanedPath)
			http.ServeFile(w, r, indexPath)
			return
		}

		candidate := filepath.Join(s.dashboardDir, filepath.FromSlash(relativePath))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			applyDashboardCacheHeaders(w, cleanedPath)
			http.ServeFile(w, r, candidate)
			return
		}

		if relativePath == "assets" || strings.HasPrefix(relativePath, "assets/") || strings.Contains(path.Base(cleanedPath), ".") {
			http.NotFound(w, r)
			return
		}

		applyDashboardCacheHeaders(w, cleanedPath)
		http.ServeFile(w, r, indexPath)
	})
}

func applyDashboardCacheHeaders(w http.ResponseWriter, cleanedPath string) {
	if strings.HasPrefix(cleanedPath, "/assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
}
