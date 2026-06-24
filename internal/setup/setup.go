package setup

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/system"
	"github.com/gin31259461/homebase/internal/ui"
)

type Runner func() error

func Exists(key string) bool {
	_, ok := Runners(run.New())[key]
	return ok
}

func Runners(r run.Runner) map[string]Runner {
	return map[string]Runner{
		"autologin":      func() error { return Autologin(r) },
		"dm":             func() error { return SDDM(r) },
		"sddm":           func() error { return SDDM(r) },
		"network":        func() error { return NetworkManager(r) },
		"networkmanager": func() error { return NetworkManager(r) },
		"dnsmasq":        func() error { return DNSMasq(r) },
		"docker":         func() error { return Docker(r) },
		"razer":          func() error { return Razer(r) },
		"sunshine":       func() error { return Sunshine(r) },
	}
}

func RunKey(r run.Runner, key string) error {
	fn, ok := Runners(r)[key]
	if !ok {
		return os.ErrNotExist
	}
	ui.Section("Setup " + key)
	return fn()
}

func Autologin(r run.Runner) error {
	user := os.Getenv("USER")
	content := "[Service]\nExecStart=\nExecStart=-/sbin/agetty --autologin " + user + " --noclear %I $TERM\n"
	return writeSudoFile(r, "/etc/systemd/system/getty@tty1.service.d/override.conf", content)
}

func SDDM(r run.Runner) error {
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

func DNSMasq(r run.Runner) error {
	if system.SystemActive(r, "dnsmasq") {
		return r.Run("sudo", "systemctl", "restart", "dnsmasq")
	}
	return r.Run("sudo", "systemctl", "enable", "dnsmasq", "--now")
}

func NetworkManager(r run.Runner) error {
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
	if system.SystemActive(r, "NetworkManager") {
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
	commands := [][]string{
		{"con", "add", "con-name", "Arch-Hyprland", "ifname", "wlan0", "type", "wifi", "ssid", "Arch-Hyprland"},
		{"con", "modify", "Arch-Hyprland", "wifi.band", "bg", "wifi.channel", "6", "wifi.mode", "ap"},
		{"con", "modify", "Arch-Hyprland", "wifi-sec.key-mgmt", "wpa-psk", "wifi-sec.psk", "ilovearchlinux", "802-11-wireless-security.pmf", "1"},
		{"con", "modify", "Arch-Hyprland", "ipv4.addresses", "192.168.10.1/24", "ipv4.method", "shared", "ipv4.never-default", "yes"},
		{"con", "modify", "Arch-Hyprland", "ipv6.method", "shared", "ipv6.addr-gen-mode", "default"},
		{"con", "modify", "Arch-Hyprland", "802-11-wireless-security.proto", "rsn", "802-11-wireless-security.pairwise", "ccmp", "802-11-wireless-security.group", "ccmp"},
	}
	for _, args := range commands {
		if err := r.Run("nmcli", args...); err != nil {
			return err
		}
	}
	ui.Note("Hotspot profile uses wlan0; adjust with nmcli if your adapter differs")
	return nil
}

func Docker(r run.Runner) error {
	if err := r.Run("sudo", "systemctl", "enable", "--now", "docker.service"); err != nil {
		return err
	}
	if !system.UserInGroup(r, os.Getenv("USER"), "docker") {
		if err := r.Run("sudo", "gpasswd", "-a", os.Getenv("USER"), "docker"); err != nil {
			return err
		}
		ui.Note("Log out and back in for docker group changes")
	}
	return nil
}

func Razer(r run.Runner) error {
	if !system.UserInGroup(r, os.Getenv("USER"), "openrazer") {
		if err := r.Run("sudo", "gpasswd", "-a", os.Getenv("USER"), "openrazer"); err != nil {
			return err
		}
		ui.Note("Log out and back in for openrazer group changes")
	}
	module, err := system.LatestOpenRazerModule("/usr/src")
	if err == nil && module != "" {
		if err := r.Run("sudo", "dkms", "install", module); err != nil {
			return err
		}
	} else {
		ui.Warn("No OpenRazer DKMS source found in /usr/src")
	}
	return r.Run("systemctl", "--user", "enable", "openrazer-daemon.service")
}

func Sunshine(r run.Runner) error {
	path, err := execLookPath("sunshine")
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
	return r.Run("systemctl", "--user", "enable", "sunshine.service")
}

var execLookPath = func(file string) (string, error) {
	return systemExecLookPath(file)
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
