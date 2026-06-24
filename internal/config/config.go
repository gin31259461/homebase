package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type App struct {
	Dotfiles struct {
		SSHRepo    string `toml:"ssh_repo"`
		HTTPSRepo  string `toml:"https_repo"`
		Dir        string `toml:"dir"`
		Branch     string `toml:"branch"`
		MemoryFile string `toml:"memory_file"`
	} `toml:"dotfiles"`
	PackageManager PackageManager `toml:"package_manager"`
	Bootstrap      struct {
		Basics []string `toml:"basics"`
	} `toml:"bootstrap"`
}

type Global struct {
	ActivePlatform  string            `toml:"active_platform"`
	PlatformAliases map[string]string `toml:"platform_aliases"`
}

type PackageManager struct {
	Official string `toml:"official"`
	AUR      string `toml:"aur"`
}

type PackageGroup struct {
	Key          string
	Label        string   `toml:"label"`
	Pacman       []string `toml:"pacman"`
	AUR          []string `toml:"aur"`
	ScoopBuckets []string `toml:"scoop_buckets"`
	Winget       []string `toml:"winget"`
	Scoop        []string `toml:"scoop"`
	PSModules    []string `toml:"psmodules"`
	Features     []string `toml:"features"`
}

type CleanupTask struct {
	Key      string
	Label    string `toml:"label"`
	Detail   string `toml:"detail"`
	Requires string `toml:"requires"`
	Sudo     bool   `toml:"sudo"`
}

type syncGroup struct {
	Paths []string `toml:"paths"`
}

func Ensure(force bool) error {
	return EnsureForPlatform("archlinux", force)
}

func EnsureGlobal(force bool) error {
	src := Expand("~/.local/lib/homebase/config/homebase.toml")
	dst := Expand("~/.config/homebase/homebase.toml")
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("default global config not found at %s", src)
	}
	return copyFile(src, dst, force)
}

func EnsureForPlatform(platformID string, force bool) error {
	if err := EnsureGlobal(force); err != nil {
		return err
	}
	src := filepath.Join(Expand("~/.local/lib/homebase/config/platforms"), platformID)
	dst := PlatformConfigDir(platformID)
	if !force && platformConfigReady(platformID) {
		return nil
	}
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("default config for platform %q not found at %s", platformID, src)
	}
	return copyTree(src, dst, force)
}

func copyTree(src, dst string, force bool) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil || rel == "." {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target, force)
	})
}

func copyFile(src, dst string, force bool) error {
	if _, err := os.Stat(dst); err == nil && !force {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func platformConfigReady(platformID string) bool {
	dir := PlatformConfigDir(platformID)
	required := []string{
		filepath.Join(dir, "config.toml"),
		filepath.Join(dir, "packages.d"),
		filepath.Join(dir, "cleanup.toml"),
		filepath.Join(dir, "sync.toml"),
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}

func LoadGlobal() (Global, error) {
	var cfg Global
	b, err := os.ReadFile(Expand("~/.config/homebase/homebase.toml"))
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if strings.TrimSpace(cfg.ActivePlatform) == "" {
		cfg.ActivePlatform = "auto"
	}
	if cfg.PlatformAliases == nil {
		cfg.PlatformAliases = map[string]string{}
	}
	return cfg, nil
}

func LoadForPlatform(platformID string) (App, error) {
	var cfg App
	b, err := os.ReadFile(filepath.Join(PlatformConfigDir(platformID), "config.toml"))
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	cfg.Dotfiles.Dir = Expand(defaultString(cfg.Dotfiles.Dir, "~/.dotfiles"))
	cfg.Dotfiles.MemoryFile = Expand(defaultString(cfg.Dotfiles.MemoryFile, "~/.dotfiles-repo"))
	cfg.Dotfiles.Branch = defaultString(cfg.Dotfiles.Branch, "main")
	cfg.PackageManager.Official = defaultString(cfg.PackageManager.Official, "pacman")
	cfg.PackageManager.AUR = defaultString(cfg.PackageManager.AUR, "yay")
	return cfg, nil
}

func LoadPackageGroupsForPlatform(platformID string) ([]PackageGroup, error) {
	return LoadPackageGroupsFromDir(filepath.Join(PlatformConfigDir(platformID), "packages.d"))
}

func LoadPackageGroupsFromDir(dir string) ([]PackageGroup, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.toml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no package group TOML files found in %s", dir)
	}
	var groups []PackageGroup
	seen := map[string]bool{}
	for _, file := range files {
		var raw map[string]PackageGroup
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		if err := toml.Unmarshal(b, &raw); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		keys := make([]string, 0, len(raw))
		for key := range raw {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if seen[key] {
				return nil, fmt.Errorf("duplicate package group: %s", key)
			}
			group := raw[key]
			group.Key = key
			if strings.TrimSpace(group.Label) == "" {
				return nil, fmt.Errorf("%s: [%s].label is required", file, key)
			}
			groups = append(groups, group)
			seen[key] = true
		}
	}
	return groups, nil
}

func LoadCleanupTasksForPlatform(platformID string, commandExists func(string) bool) ([]CleanupTask, error) {
	return LoadCleanupTasksFromFile(filepath.Join(PlatformConfigDir(platformID), "cleanup.toml"), commandExists)
}

func LoadCleanupTasksFromFile(path string, commandExists func(string) bool) ([]CleanupTask, error) {
	var raw map[string]CleanupTask
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var tasks []CleanupTask
	for _, key := range keys {
		task := raw[key]
		if task.Requires != "" && commandExists != nil && !commandExists(task.Requires) {
			continue
		}
		task.Key = key
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func LoadSyncPathsForPlatform(platformID string) ([]string, error) {
	return LoadSyncPathsFromFile(filepath.Join(PlatformConfigDir(platformID), "sync.toml"))
}

func LoadSyncPathsFromFile(path string) ([]string, error) {
	var raw map[string]syncGroup
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	return SyncPaths(raw), nil
}

func SyncPaths(raw map[string]syncGroup) []string {
	seen := map[string]bool{}
	var paths []string
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, path := range raw[key].Paths {
			path = strings.TrimSpace(path)
			if path == "" || seen[path] {
				continue
			}
			paths = append(paths, path)
			seen[path] = true
		}
	}
	return paths
}

func Expand(path string) string {
	if path == "~" {
		if home := homeDir(); home != "" {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home := homeDir(); home != "" {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return os.ExpandEnv(path)
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}

func PlatformConfigDir(platformID string) string {
	return filepath.Join(Expand("~/.config/homebase/platforms"), platformID)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
