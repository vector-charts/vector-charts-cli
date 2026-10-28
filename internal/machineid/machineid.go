package machineid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Resolve a stable machine fingerprint from hardware serial / machine-id.
// Returns a sha256 hex digest so raw serials are never sent to the portal.

func ID() (string, error) {
	raw := rawFingerprint()
	if raw == "" {
		return "", fmt.Errorf("could not determine machine id")
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:]), nil
}

func Label() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return ""
	}
	return h
}

func rawFingerprint() string {
	for _, path := range []string{
		"/sys/firmware/devicetree/base/serial-number",
		"/sys/class/dmi/id/product_uuid",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		s := strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", ""))
		if s != "" {
			return s
		}
	}

	switch runtime.GOOS {
	case "linux":
		for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			s := strings.TrimSpace(string(data))
			if s != "" {
				return s
			}
		}
	case "darwin":
		out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
		if err == nil {
			s := strings.TrimSpace(string(out))
			if s != "" {
				return s
			}
		}
	}
	return ""
}
