package config

import (
	"fmt"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// Validate validates the configuration and returns any errors found
func Validate(cfg *Config) error {
	var errors ValidationErrors

	// Validate hosts
	for name, host := range cfg.Hosts {
		if host.Hostname == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("hosts.%s.hostname", name),
				Message: "hostname is required",
			})
		}
		if host.Port <= 0 || host.Port > 65535 {
			if host.Port == 0 {
				// Set default port
				host.Port = 22
				cfg.Hosts[name] = host
			} else {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("hosts.%s.port", name),
					Message: "port must be between 1 and 65535",
				})
			}
		}
		if host.User == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("hosts.%s.user", name),
				Message: "user is required",
			})
		}
		// Validate jump host reference exists
		if host.JumpHost != "" {
			if _, ok := cfg.Hosts[host.JumpHost]; !ok {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("hosts.%s.jump_host", name),
					Message: fmt.Sprintf("jump host '%s' not found in hosts", host.JumpHost),
				})
			}
		}
	}

	// Validate tunnels
	tunnelNames := make(map[string]bool)
	for i, tunnel := range cfg.Tunnels {
		prefix := fmt.Sprintf("tunnels[%d]", i)

		if tunnel.Name == "" {
			errors = append(errors, ValidationError{
				Field:   prefix + ".name",
				Message: "name is required",
			})
		} else {
			if tunnelNames[tunnel.Name] {
				errors = append(errors, ValidationError{
					Field:   prefix + ".name",
					Message: fmt.Sprintf("duplicate tunnel name '%s'", tunnel.Name),
				})
			}
			tunnelNames[tunnel.Name] = true
		}

		if tunnel.Host == "" {
			errors = append(errors, ValidationError{
				Field:   prefix + ".host",
				Message: "host is required",
			})
		} else if _, ok := cfg.Hosts[tunnel.Host]; !ok {
			errors = append(errors, ValidationError{
				Field:   prefix + ".host",
				Message: fmt.Sprintf("host '%s' not found in hosts", tunnel.Host),
			})
		}

		// Validate tunnel type
		switch tunnel.Type {
		case TunnelTypeLocal, TunnelTypeRemote:
			// Validate local endpoint
			if tunnel.Local.Port <= 0 || tunnel.Local.Port > 65535 {
				errors = append(errors, ValidationError{
					Field:   prefix + ".local.port",
					Message: "local port must be between 1 and 65535",
				})
			}
			// Validate remote endpoint
			if tunnel.Remote.Port <= 0 || tunnel.Remote.Port > 65535 {
				errors = append(errors, ValidationError{
					Field:   prefix + ".remote.port",
					Message: "remote port must be between 1 and 65535",
				})
			}
			if tunnel.Remote.Address == "" {
				errors = append(errors, ValidationError{
					Field:   prefix + ".remote.address",
					Message: "remote address is required for local/remote tunnels",
				})
			}
		case TunnelTypeDynamic:
			// Dynamic tunnels only need local endpoint
			if tunnel.Local.Port <= 0 || tunnel.Local.Port > 65535 {
				errors = append(errors, ValidationError{
					Field:   prefix + ".local.port",
					Message: "local port must be between 1 and 65535",
				})
			}
		case "":
			errors = append(errors, ValidationError{
				Field:   prefix + ".type",
				Message: "type is required (local, remote, or dynamic)",
			})
		default:
			errors = append(errors, ValidationError{
				Field:   prefix + ".type",
				Message: fmt.Sprintf("invalid tunnel type '%s' (must be local, remote, or dynamic)", tunnel.Type),
			})
		}

		// Set default local address if not specified
		if tunnel.Local.Address == "" {
			cfg.Tunnels[i].Local.Address = "127.0.0.1"
		}
	}

	// Validate groups
	groupNames := make(map[string]bool)
	for i, group := range cfg.Groups {
		prefix := fmt.Sprintf("groups[%d]", i)

		if group.Name == "" {
			errors = append(errors, ValidationError{
				Field:   prefix + ".name",
				Message: "name is required",
			})
		} else {
			if groupNames[group.Name] {
				errors = append(errors, ValidationError{
					Field:   prefix + ".name",
					Message: fmt.Sprintf("duplicate group name '%s'", group.Name),
				})
			}
			groupNames[group.Name] = true
		}

		// Validate tunnel references
		for j, tunnelName := range group.Tunnels {
			if !tunnelNames[tunnelName] {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.tunnels[%d]", prefix, j),
					Message: fmt.Sprintf("tunnel '%s' not found", tunnelName),
				})
			}
		}
	}

	// Validate global settings
	if cfg.Global.Reconnect.BackoffMultiplier < 1 {
		errors = append(errors, ValidationError{
			Field:   "global.reconnect.backoff_multiplier",
			Message: "backoff multiplier must be at least 1",
		})
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}
