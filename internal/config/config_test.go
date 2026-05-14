package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func setupTestEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	return tmp
}

func TestSaveAndLoad(t *testing.T) {
	setupTestEnv(t)

	want := &Config{
		AppID:          12345,
		InstallationID: 67890,
		PrivateKeyPath: filepath.FromSlash("/tmp/test-key.pem"),
	}

	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.AppID != want.AppID {
		t.Errorf("AppID = %d, want %d", got.AppID, want.AppID)
	}
	if got.InstallationID != want.InstallationID {
		t.Errorf("InstallationID = %d, want %d", got.InstallationID, want.InstallationID)
	}
	if got.PrivateKeyPath != want.PrivateKeyPath {
		t.Errorf("PrivateKeyPath = %q, want %q", got.PrivateKeyPath, want.PrivateKeyPath)
	}
}

func TestLoad_NotFound(t *testing.T) {
	setupTestEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatal("Load: expected error for missing config, got nil")
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "missing app_id",
			yaml:    "installation_id: 1\nprivate_key_path: /tmp/k.pem\n",
			wantErr: "app_id must be a positive integer",
		},
		{
			name:    "negative app_id",
			yaml:    "app_id: -1\ninstallation_id: 1\nprivate_key_path: /tmp/k.pem\n",
			wantErr: "app_id must be a positive integer",
		},
		{
			name:    "negative installation_id",
			yaml:    "app_id: 1\ninstallation_id: -5\nprivate_key_path: /tmp/k.pem\n",
			wantErr: "installation_id must not be negative",
		},
		{
			name:    "missing private_key_path",
			yaml:    "app_id: 1\ninstallation_id: 1\n",
			wantErr: "private_key_path is required",
		},
		{
			name:    "whitespace-only private_key_path",
			yaml:    "app_id: 1\ninstallation_id: 1\nprivate_key_path: \"   \"\n",
			wantErr: "private_key_path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := setupTestEnv(t)

			dir := filepath.Join(tmp, ".config", configDir)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, configFile), []byte(tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := Load()
			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLoad_OmittedInstallationID(t *testing.T) {
	tmp := setupTestEnv(t)

	dir := filepath.Join(tmp, ".config", configDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	yml := "app_id: 1\nprivate_key_path: /tmp/k.pem\n"
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.InstallationID != 0 {
		t.Errorf("InstallationID = %d, want 0", cfg.InstallationID)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmp := setupTestEnv(t)

	dir := filepath.Join(tmp, ".config", configDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(":::invalid"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("error = %q, want substring %q", err.Error(), "parsing config")
	}
}

func TestLoad_UnknownField(t *testing.T) {
	tmp := setupTestEnv(t)

	dir := filepath.Join(tmp, ".config", configDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	yml := "app_id: 1\ninstallation_id: 1\nprivate_key_path: /tmp/k.pem\ntypo_field: oops\n"
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmp := setupTestEnv(t)

	cfg := &Config{
		AppID:          1,
		InstallationID: 2,
		PrivateKeyPath: "/tmp/k.pem",
	}

	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(tmp, ".config", configDir, configFile)
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	dirPath := filepath.Join(tmp, ".config", configDir)
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("config dir not created: %v", err)
	}

	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("config file permissions = %o, want 0600", perm)
		}
		if perm := dirInfo.Mode().Perm(); perm != 0o700 {
			t.Errorf("config dir permissions = %o, want 0700", perm)
		}
	}
}

func TestSave_FixesExistingPermissions(t *testing.T) {
	tmp := setupTestEnv(t)

	configPath := filepath.Join(tmp, ".config", configDir, configFile)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		AppID:          1,
		InstallationID: 2,
		PrivateKeyPath: "/tmp/k.pem",
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("config file permissions after Save = %o, want 0600", perm)
		}
	}
}

func TestSave_NilConfig(t *testing.T) {
	setupTestEnv(t)

	err := Save(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestDir(t *testing.T) {
	tmp := setupTestEnv(t)

	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(tmp, ".config", configDir)
	if dir != want {
		t.Errorf("Dir() = %q, want %q", dir, want)
	}
}

func TestLoad_DirectoriesValid(t *testing.T) {
	tmp := setupTestEnv(t)

	dir := filepath.Join(tmp, ".config", configDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	projectsFoo := filepath.Join(tmp, "projects", "foo")
	yml := fmt.Sprintf("app_id: 1\nprivate_key_path: /tmp/k.pem\n"+
		"directories:\n"+
		"  - path: %q\n"+
		"    installation_id: 11\n"+
		"  - path: ~/work/bar\n"+
		"    org: myorg\n", filepath.ToSlash(projectsFoo))
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Directories) != 2 {
		t.Fatalf("Directories len = %d, want 2", len(cfg.Directories))
	}
	if cfg.Directories[0].Path != filepath.Clean(projectsFoo) {
		t.Errorf("Directories[0].Path = %q, want %q", cfg.Directories[0].Path, filepath.Clean(projectsFoo))
	}
	wantExpanded := filepath.Join(tmp, "work", "bar")
	if cfg.Directories[1].Path != wantExpanded {
		t.Errorf("Directories[1].Path = %q, want %q", cfg.Directories[1].Path, wantExpanded)
	}
	if cfg.Directories[1].Org != "myorg" {
		t.Errorf("Directories[1].Org = %q", cfg.Directories[1].Org)
	}
}

func TestLoad_DirectoriesValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "missing path",
			yaml: "app_id: 1\nprivate_key_path: /tmp/k.pem\n" +
				"directories:\n  - installation_id: 1\n",
			wantErr: "directories[0].path is required",
		},
		{
			name: "missing both id and org",
			yaml: "app_id: 1\nprivate_key_path: /tmp/k.pem\n" +
				"directories:\n  - path: /opt/foo\n",
			wantErr: "either installation_id or org is required",
		},
		{
			name: "negative installation_id",
			yaml: "app_id: 1\nprivate_key_path: /tmp/k.pem\n" +
				"directories:\n  - path: /opt/foo\n    installation_id: -1\n",
			wantErr: "must not be negative",
		},
		{
			name: "relative path",
			yaml: "app_id: 1\nprivate_key_path: /tmp/k.pem\n" +
				"directories:\n  - path: ./foo\n    installation_id: 1\n",
			wantErr: "must be absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := setupTestEnv(t)
			dir := filepath.Join(tmp, ".config", configDir)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, configFile), []byte(tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := Load()
			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestResolveDirectory_LongestMatch(t *testing.T) {
	cfg := &Config{
		Directories: []DirectoryRule{
			{Path: filepath.Clean("/opt/projects"), InstallationID: 1},
			{Path: filepath.Clean("/opt/projects/foo"), InstallationID: 2},
		},
	}

	rule := cfg.ResolveDirectory(filepath.Clean("/opt/projects/foo/sub"))
	if rule == nil || rule.InstallationID != 2 {
		t.Fatalf("ResolveDirectory = %+v, want id=2", rule)
	}

	rule = cfg.ResolveDirectory(filepath.Clean("/opt/projects/bar"))
	if rule == nil || rule.InstallationID != 1 {
		t.Fatalf("ResolveDirectory = %+v, want id=1", rule)
	}

	if got := cfg.ResolveDirectory(filepath.Clean("/elsewhere")); got != nil {
		t.Errorf("ResolveDirectory = %+v, want nil", got)
	}
}

func TestResolveDirectory_NotPrefixOfSiblingWithSameName(t *testing.T) {
	cfg := &Config{
		Directories: []DirectoryRule{
			{Path: filepath.Clean("/opt/foo"), InstallationID: 1},
		},
	}
	// "/opt/foobar" must NOT match rule "/opt/foo".
	if got := cfg.ResolveDirectory(filepath.Clean("/opt/foobar")); got != nil {
		t.Errorf("ResolveDirectory matched a sibling: %+v", got)
	}
}

func TestResolveDirectory_ExactMatch(t *testing.T) {
	cfg := &Config{
		Directories: []DirectoryRule{
			{Path: filepath.Clean("/opt/foo"), InstallationID: 1},
		},
	}
	rule := cfg.ResolveDirectory(filepath.Clean("/opt/foo"))
	if rule == nil || rule.InstallationID != 1 {
		t.Fatalf("ResolveDirectory = %+v, want id=1", rule)
	}
}

func TestResolveDirectory_EmptyConfig(t *testing.T) {
	var cfg *Config
	if got := cfg.ResolveDirectory("/anything"); got != nil {
		t.Errorf("nil config should return nil, got %+v", got)
	}
	cfg = &Config{}
	if got := cfg.ResolveDirectory("/anything"); got != nil {
		t.Errorf("empty config should return nil, got %+v", got)
	}
}

func TestResolveDirectory_EmptyCwd(t *testing.T) {
	cfg := &Config{
		Directories: []DirectoryRule{
			{Path: filepath.Clean("/opt/foo"), InstallationID: 1},
		},
	}
	if got := cfg.ResolveDirectory(""); got != nil {
		t.Errorf("empty cwd should return nil, got %+v", got)
	}
}

func TestResolveDirectory_UncleanedCwd(t *testing.T) {
	cfg := &Config{
		Directories: []DirectoryRule{
			{Path: filepath.Clean("/opt/foo"), InstallationID: 1},
		},
	}
	// Path with trailing slash and "." segments should still match.
	rule := cfg.ResolveDirectory("/opt/foo/./sub/")
	if rule == nil || rule.InstallationID != 1 {
		t.Errorf("ResolveDirectory should normalize cwd, got %+v", rule)
	}
}

func TestResolveDirectory_OrgRule(t *testing.T) {
	cfg := &Config{
		Directories: []DirectoryRule{
			{Path: filepath.Clean("/opt/foo"), Org: "myorg"},
		},
	}
	rule := cfg.ResolveDirectory(filepath.Clean("/opt/foo/sub"))
	if rule == nil || rule.Org != "myorg" {
		t.Fatalf("ResolveDirectory = %+v, want org=myorg", rule)
	}
}

func TestLoad_DirectoryRuleWithBothIDAndOrgValid(t *testing.T) {
	// When both are set, both should be preserved; precedence is handled later.
	tmp := setupTestEnv(t)
	dir := filepath.Join(tmp, ".config", configDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	foo := filepath.Join(tmp, "foo")
	yml := fmt.Sprintf("app_id: 1\nprivate_key_path: /tmp/k.pem\n"+
		"directories:\n  - path: %q\n    installation_id: 11\n    org: myorg\n",
		filepath.ToSlash(foo))
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Directories[0].InstallationID != 11 || cfg.Directories[0].Org != "myorg" {
		t.Errorf("rule = %+v, want both fields preserved", cfg.Directories[0])
	}
}

func TestLoad_DirectoriesWhitespaceOrgIsInvalid(t *testing.T) {
	tmp := setupTestEnv(t)
	dir := filepath.Join(tmp, ".config", configDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	yml := "app_id: 1\nprivate_key_path: /tmp/k.pem\n" +
		"directories:\n  - path: /opt/foo\n    org: \"   \"\n"
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for whitespace-only org")
	}
	if !strings.Contains(err.Error(), "either installation_id or org is required") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestSaveAndLoad_RoundTripDirectories(t *testing.T) {
	tmp := setupTestEnv(t)

	foo := filepath.Join(tmp, "foo")
	bar := filepath.Join(tmp, "bar")
	want := &Config{
		AppID:          1,
		PrivateKeyPath: "/tmp/k.pem",
		Directories: []DirectoryRule{
			{Path: foo, InstallationID: 11},
			{Path: bar, Org: "myorg"},
		},
	}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Directories) != 2 {
		t.Fatalf("Directories len = %d, want 2", len(got.Directories))
	}
	if got.Directories[0].InstallationID != 11 || got.Directories[1].Org != "myorg" {
		t.Errorf("roundtrip mismatch: %+v", got.Directories)
	}
}

func TestDir_XDGConfigHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(tmp, configDir)
	if dir != want {
		t.Errorf("Dir() = %q, want %q", dir, want)
	}
}
