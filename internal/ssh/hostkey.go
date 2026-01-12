package ssh

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// HostKeyCallback returns an ssh.HostKeyCallback that verifies host keys
// against the user's known_hosts file. If the host key is not found, it
// prompts for confirmation and optionally adds it to known_hosts.
func HostKeyCallback() (ssh.HostKeyCallback, error) {
	knownHostsPath := getKnownHostsPath()

	// Ensure the .ssh directory exists
	sshDir := filepath.Dir(knownHostsPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create ssh directory: %w", err)
	}

	// Ensure known_hosts file exists
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		f, err := os.Create(knownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create known_hosts file: %w", err)
		}
		f.Close()
	}

	// Parse existing known_hosts
	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse known_hosts: %w", err)
	}

	// Wrap the callback to handle unknown hosts
	return wrapHostKeyCallback(callback, knownHostsPath), nil
}

// wrapHostKeyCallback wraps a knownhosts callback to handle unknown hosts
// by auto-accepting them and adding to known_hosts (TOFU - Trust On First Use).
func wrapHostKeyCallback(callback ssh.HostKeyCallback, knownHostsPath string) ssh.HostKeyCallback {
	var mu sync.Mutex

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := callback(hostname, remote, key)
		if err == nil {
			return nil
		}

		// Check if it's a key mismatch (potential MITM attack)
		var keyErr *knownhosts.KeyError
		if isKeyError(err, &keyErr) {
			if len(keyErr.Want) > 0 {
				// Host key changed - this is serious, don't auto-accept
				return fmt.Errorf("HOST KEY VERIFICATION FAILED for %s: key has changed (possible MITM attack). "+
					"Remove the old key from %s to connect", hostname, knownHostsPath)
			}

			// Unknown host - auto-accept and add to known_hosts (TOFU)
			mu.Lock()
			defer mu.Unlock()

			if err := addHostKey(knownHostsPath, hostname, key); err != nil {
				// Log warning but don't fail the connection
				fmt.Fprintf(os.Stderr, "Warning: could not add host key to known_hosts: %v\n", err)
			}
			return nil
		}

		return err
	}
}

// isKeyError checks if the error is a knownhosts.KeyError and extracts it.
func isKeyError(err error, keyErr **knownhosts.KeyError) bool {
	if ke, ok := err.(*knownhosts.KeyError); ok {
		*keyErr = ke
		return true
	}
	return false
}

// addHostKey adds a host key to the known_hosts file.
func addHostKey(knownHostsPath, hostname string, key ssh.PublicKey) error {
	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Format: hostname key-type base64-key
	line := knownhosts.Line([]string{hostname}, key)
	_, err = fmt.Fprintln(f, line)
	return err
}

// getKnownHostsPath returns the path to the known_hosts file.
func getKnownHostsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "known_hosts")
}

// FormatFingerprint returns the SHA256 fingerprint of a public key.
func FormatFingerprint(key ssh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])
}

// HostKeyVerifier provides methods to verify and manage host keys.
type HostKeyVerifier struct {
	knownHostsPath string
	mu             sync.RWMutex
}

// NewHostKeyVerifier creates a new host key verifier.
func NewHostKeyVerifier() *HostKeyVerifier {
	return &HostKeyVerifier{
		knownHostsPath: getKnownHostsPath(),
	}
}

// IsKnown checks if a host key is known.
func (v *HostKeyVerifier) IsKnown(hostname string, port int, key ssh.PublicKey) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.knownHostsPath == "" {
		return false, fmt.Errorf("could not determine known_hosts path")
	}

	f, err := os.Open(v.knownHostsPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Normalize hostname with port
	normalizedHost := normalizeHostname(hostname, port)
	keyType := key.Type()
	keyData := base64.StdEncoding.EncodeToString(key.Marshal())

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		hosts := strings.Split(fields[0], ",")
		for _, h := range hosts {
			if h == normalizedHost || h == hostname {
				if fields[1] == keyType {
					if fields[2] == keyData {
						return true, nil
					}
					// Key type matches but key data differs - key changed
					return false, fmt.Errorf("host key has changed")
				}
			}
		}
	}

	return false, scanner.Err()
}

// normalizeHostname returns the hostname in the format used by known_hosts.
func normalizeHostname(hostname string, port int) string {
	if port == 22 {
		return hostname
	}
	return fmt.Sprintf("[%s]:%d", hostname, port)
}
