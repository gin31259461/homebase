package archlinux

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/ui"
)

type setupRunner func() error

const (
	mkinitcpioConfigPath = "/etc/mkinitcpio.conf"
	amdgpuModulesLine    = "MODULES=(usbhid xhci_pci amdgpu)"
)

func setupExists(key string) bool {
	_, ok := setupRunners(run.New())[key]
	return ok
}

func setupRunners(r run.Runner) map[string]setupRunner {
	return map[string]setupRunner{
		"autologin":      func() error { return autologin(r) },
		"dm":             func() error { return sddm(r) },
		"sddm":           func() error { return sddm(r) },
		"network":        func() error { return networkManager(r) },
		"networkmanager": func() error { return networkManager(r) },
		"shell":          func() error { return zshCompletion(r) },
		"docker":         func() error { return docker(r) },
		"razer":          func() error { return razer(r) },
		"sunshine":       func() error { return sunshine(r) },
	}
}

func runSetupKey(r run.Runner, key string) error {
	fn, ok := setupRunners(r)[key]
	if !ok {
		return os.ErrNotExist
	}
	ui.Section("Setup " + key)
	return fn()
}

func runSetup(r run.Runner, groups []config.PackageGroup, selected []string, installed map[string]bool, yes bool) error {
	ui.Section("Extra configuration")
	ui.Section("Setup system")
	if err := systemSetup(r); err != nil {
		return err
	}
	groupByKey := map[string]config.PackageGroup{}
	for _, group := range groups {
		groupByKey[group.Key] = group
	}
	seen := map[string]bool{}
	var keys []string
	for _, key := range selected {
		group := groupByKey[key]
		if setupExists(key) && groupHasInstalledPackage(group, installed) {
			keys = append(keys, key)
			seen[key] = true
		}
		for _, pkg := range append(group.Pacman, group.AUR...) {
			if !seen[pkg] && setupExists(pkg) && installed[pkg] {
				keys = append(keys, pkg)
				seen[pkg] = true
			}
		}
	}
	for _, key := range keys {
		if err := runSetupKey(r, key); err != nil {
			return err
		}
	}
	if setupExists("autologin") {
		if _, err := os.Stat("/etc/systemd/system/getty@tty1.service.d/override.conf"); err == nil {
			ui.OK("Autologin already configured")
		} else if yes || ui.Confirm("Configure tty1 autologin for "+os.Getenv("USER")+"?", false) {
			if err := runSetupKey(r, "autologin"); err != nil {
				return err
			}
		}
	}
	if len(keys) == 0 {
		ui.Note("No package setup hooks matched the selected groups")
	}
	return nil
}

func systemSetup(r run.Runner) error {
	modules, err := r.Capture("lsmod")
	if err != nil {
		warnManualMkinitcpio("Unable to detect the active GPU driver")
		return nil
	}
	if !usesAMDGPU(modules) {
		warnManualMkinitcpio("amdgpu is not the active GPU driver")
		return nil
	}

	content, err := r.Capture("cat", mkinitcpioConfigPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", mkinitcpioConfigPath, err)
	}
	updated := setMkinitcpioModules(content)
	if updated == content {
		ui.OK("mkinitcpio modules already configured for amdgpu")
		return nil
	}
	if err := writeSudoFile(r, mkinitcpioConfigPath, updated); err != nil {
		return err
	}
	if err := r.Run("sudo", "mkinitcpio", "-P"); err != nil {
		return err
	}
	ui.OK("Configured mkinitcpio modules for amdgpu")
	return nil
}

func usesAMDGPU(modules string) bool {
	for _, line := range strings.Split(strings.ToLower(modules), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "amdgpu" {
			return true
		}
	}
	return false
}

func setMkinitcpioModules(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "MODULES=") {
			lines[i] = amdgpuModulesLine
			return strings.Join(lines, "\n")
		}
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + amdgpuModulesLine + "\n"
}

func warnManualMkinitcpio(reason string) {
	ui.Warn(reason + "; leaving " + mkinitcpioConfigPath + " unchanged")
	ui.Note("Configure MODULES in " + mkinitcpioConfigPath + " for your GPU before rebooting, or the system may boot to a blank screen")
}

func groupHasInstalledPackage(group config.PackageGroup, installed map[string]bool) bool {
	all := append(append([]string{}, group.Pacman...), group.AUR...)
	if len(all) == 0 {
		return true
	}
	for _, pkg := range all {
		if installed[pkg] {
			return true
		}
	}
	return false
}

func gitCredentials(r run.Runner) error {
	initialized, err := passwordStoreInitialized()
	if err != nil {
		return err
	}
	if initialized {
		ui.OK("Password store already initialized")
		return configureGitCredentialManager(r)
	}

	before, err := secretKeyFingerprints(r)
	if err != nil {
		return fmt.Errorf("list existing GPG secret keys: %w", err)
	}
	ui.Note("GPG will prompt for your real name, email, and key passphrase")
	if err := r.Run("gpg", "--gen-key"); err != nil {
		return fmt.Errorf("generate GPG key: %w", err)
	}
	after, err := secretKeyFingerprints(r)
	if err != nil {
		return fmt.Errorf("list generated GPG secret key: %w", err)
	}
	fingerprint, err := addedSecretKeyFingerprint(before, after)
	if err != nil {
		return err
	}
	if err := r.Run("pass", "init", fingerprint); err != nil {
		return fmt.Errorf("initialize password store: %w", err)
	}
	ui.OK("Initialized password store with GPG key " + fingerprint)
	return configureGitCredentialManager(r)
}

func passwordStoreInitialized() (bool, error) {
	dir := strings.TrimSpace(os.Getenv("PASSWORD_STORE_DIR"))
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false, fmt.Errorf("find home directory: %w", err)
		}
		dir = filepath.Join(home, ".password-store")
	}
	content, err := os.ReadFile(filepath.Join(dir, ".gpg-id"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read password store GPG ID: %w", err)
	}
	return strings.TrimSpace(string(content)) != "", nil
}

func secretKeyFingerprints(r run.Runner) ([]string, error) {
	output, err := r.Capture("gpg", "--batch", "--with-colons", "--fingerprint", "--list-secret-keys")
	if err != nil {
		return nil, err
	}
	return parsePrimarySecretKeyFingerprints(output), nil
}

func parsePrimarySecretKeyFingerprints(output string) []string {
	var fingerprints []string
	wantFingerprint := false
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "sec":
			wantFingerprint = true
		case "ssb":
			wantFingerprint = false
		case "fpr":
			if wantFingerprint && len(fields) > 9 && fields[9] != "" {
				fingerprints = append(fingerprints, fields[9])
				wantFingerprint = false
			}
		}
	}
	return fingerprints
}

func addedSecretKeyFingerprint(before, after []string) (string, error) {
	existing := make(map[string]bool, len(before))
	for _, fingerprint := range before {
		existing[fingerprint] = true
	}
	var added []string
	for _, fingerprint := range after {
		if !existing[fingerprint] {
			added = append(added, fingerprint)
		}
	}
	if len(added) == 0 {
		return "", fmt.Errorf("GPG key generation completed but no new secret key was found")
	}
	if len(added) > 1 {
		return "", fmt.Errorf("GPG key generation added multiple secret keys; initialize pass manually with the intended key")
	}
	return added[0], nil
}

func configureGitCredentialManager(r run.Runner) error {
	if err := r.Run("git-credential-manager", "configure"); err != nil {
		return fmt.Errorf("configure Git Credential Manager: %w", err)
	}
	if err := r.Run("git", "config", "--global", "credential.credentialStore", "gpg"); err != nil {
		return fmt.Errorf("select GPG credential store: %w", err)
	}
	ui.OK("Configured Git Credential Manager to use the GPG credential store")
	return nil
}

func autologin(r run.Runner) error {
	user := os.Getenv("USER")
	content := "[Service]\nExecStart=\nExecStart=-/sbin/agetty --autologin " + user + " --noclear %I $TERM\n"
	return writeSudoFile(r, "/etc/systemd/system/getty@tty1.service.d/override.conf", content)
}

func sddm(r run.Runner) error {
	user := os.Getenv("USER")
	if err := r.Run("sudo", "systemctl", "enable", "sddm.service"); err != nil {
		return err
	}
	if err := writeSudoFile(r, "/etc/sddm.conf.d/autologin.conf", "[Autologin]\nUser="+user+"\nSession=hyprland-uwsm\n"); err != nil {
		return err
	}
	session := `[Desktop Entry]
Name=Hyprland (uwsm)
Comment=Wayland compositor, uwsm session
Exec=uwsm start -- hyprland.desktop
TryExec=uwsm
DesktopNames=Hyprland
Type=Application
`
	return writeSudoFile(r, "/usr/share/wayland-sessions/hyprland-uwsm.desktop", session)
}

func networkManager(r run.Runner) error {
	mainConf := `[main]
dns=dnsmasq
ignore-carrier=true

[connection]
wifi.powersave = 2
`
	if err := writeSudoFile(r, "/etc/NetworkManager/conf.d/main.conf", mainConf); err != nil {
		return err
	}
	if err := writeSudoFile(r, "/etc/NetworkManager/conf.d/99-tailscale.conf", "[keyfile]\nunmanaged-devices=interface-name:tailscale0\n"); err != nil {
		return err
	}
	if systemActive(r, "NetworkManager") {
		if err := r.Run("sudo", "systemctl", "restart", "NetworkManager"); err != nil {
			return err
		}
	} else if err := r.Run("sudo", "systemctl", "enable", "NetworkManager", "--now"); err != nil {
		return err
	}
	sysctl := `net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr
net.ipv4.ip_forward = 1
net.ipv6.conf.all.forwarding = 1
`
	if err := writeSudoFile(r, "/etc/sysctl.d/99-sysctl.conf", sysctl); err != nil {
		return err
	}
	if err := r.Run("sudo", "sysctl", "--system"); err != nil {
		return err
	}
	if r.Quiet("nmcli", "con", "show", "Arch-Hyprland") == nil {
		ui.OK("Hotspot connection profile already exists")
		return nil
	}
	hotspotPSK, err := generateHotspotPSK()
	if err != nil {
		return err
	}
	commands := [][]string{
		{"con", "add", "con-name", "Arch-Hyprland", "ifname", "wlan0", "type", "wifi", "ssid", "Arch-Hyprland"},
		{"con", "modify", "Arch-Hyprland", "wifi.band", "bg", "wifi.channel", "6", "wifi.mode", "ap"},
		{"con", "modify", "Arch-Hyprland", "wifi-sec.key-mgmt", "wpa-psk", "wifi-sec.psk", hotspotPSK, "802-11-wireless-security.pmf", "1"},
		{"con", "modify", "Arch-Hyprland", "ipv4.addresses", "192.168.10.1/24", "ipv4.method", "shared", "ipv4.never-default", "yes"},
		{"con", "modify", "Arch-Hyprland", "ipv6.method", "shared", "ipv6.addr-gen-mode", "default"},
		{"con", "modify", "Arch-Hyprland", "802-11-wireless-security.proto", "rsn", "802-11-wireless-security.pairwise", "ccmp", "802-11-wireless-security.group", "ccmp"},
	}
	for _, args := range commands {
		if err := r.Run("nmcli", args...); err != nil {
			return err
		}
	}
	ui.Note("Generated a unique hotspot PSK; view it with: nmcli -s -g 802-11-wireless-security.psk connection show Arch-Hyprland")
	ui.Note("Hotspot profile uses wlan0; adjust with nmcli if your adapter differs")
	ui.Note("nmcli conn modify Arch-Hyprland connection.interface-name __ip link__")
	ui.Note("nmcli conn modify Arch-Hyprland 802-11-wireless-security.psk __new passwd__")
	return nil
}

func generateHotspotPSK() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func docker(r run.Runner) error {
	if err := r.Run("sudo", "systemctl", "enable", "--now", "docker.service"); err != nil {
		return err
	}
	if !userInGroup(r, os.Getenv("USER"), "docker") {
		if err := r.Run("sudo", "gpasswd", "-a", os.Getenv("USER"), "docker"); err != nil {
			return err
		}
		ui.Note("Log out and back in for docker group changes")
	}
	return nil
}

func razer(r run.Runner) error {
	if !userInGroup(r, os.Getenv("USER"), "openrazer") {
		if err := r.Run("sudo", "gpasswd", "-a", os.Getenv("USER"), "openrazer"); err != nil {
			return err
		}
		ui.Note("Log out and back in for openrazer group changes")
	}
	module, err := latestOpenRazerModule("/usr/src")
	if err == nil && module != "" {
		if err := r.Run("sudo", "dkms", "install", module); err != nil {
			return err
		}
	} else {
		ui.Warn("No OpenRazer DKMS source found in /usr/src")
	}
	return r.Run("systemctl", "--user", "enable", "openrazer-daemon.service")
}

func sunshine(r run.Runner) error {
	path, err := exec.LookPath("sunshine")
	if err != nil {
		ui.Warn("Sunshine executable not found")
		return nil
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	}
	out, _ := r.Capture("getcap", path)
	if !strings.Contains(out, "cap_sys_admin") {
		if err := r.Run("sudo", "setcap", "cap_sys_admin+p", path); err != nil {
			return err
		}
	}
	return r.Run("systemctl", "--user", "enable", "app-dev.lizardbyte.app.Sunshine.service")
}

func writeSudoFile(r run.Runner, path, content string) error {
	tmp, err := os.CreateTemp("", "homebase-file-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return r.Run("sudo", "install", "-Dm644", tmpPath, path)
}

func systemActive(r run.Runner, unit string) bool {
	return r.Quiet("systemctl", "is-active", "--quiet", unit) == nil
}

func userInGroup(r run.Runner, user, group string) bool {
	out, err := r.Capture("groups", user)
	return err == nil && containsWord(out, group)
}

func containsWord(text, word string) bool {
	for _, field := range strings.Fields(text) {
		if field == word {
			return true
		}
	}
	return false
}

func latestOpenRazerModule(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "openrazer-driver-") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", nil
	}
	version := strings.TrimPrefix(names[len(names)-1], "openrazer-driver-")
	return "openrazer-driver/" + version, nil
}
