package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Context stores the current context state
type Context struct {
	ConfigFile    string `yaml:"config_file"`
	ActiveBackend string `yaml:"active_backend"`
}

// contextPath returns the path to the context file
func contextPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "resticm", "context.yaml")
}

// LoadContext loads the current context
func LoadContext() (*Context, error) {
	path := contextPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Context{}, nil
		}
		return nil, fmt.Errorf("failed to read context: %w", err)
	}

	var ctx Context
	if err := yaml.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("failed to parse context: %w", err)
	}

	return &ctx, nil
}

// SaveContext saves the current context
func SaveContext(ctx *Context) error {
	path := contextPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create context directory: %w", err)
	}

	data, err := yaml.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write context: %w", err)
	}

	return nil
}

// ResetContext removes the context file
func ResetContext() error {
	path := contextPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove context: %w", err)
	}
	return nil
}

// SetConfigFile sets the active config file in context
func SetConfigFile(configFile string) error {
	ctx, err := LoadContext()
	if err != nil {
		ctx = &Context{}
	}

	ctx.ConfigFile = configFile
	return SaveContext(ctx)
}

// SetActiveBackend sets the active backend in context
func SetActiveBackend(backend string) error {
	ctx, err := LoadContext()
	if err != nil {
		ctx = &Context{}
	}

	ctx.ActiveBackend = backend
	return SaveContext(ctx)
}

// GetActiveBackend returns the active backend from context
func GetActiveBackend() (string, error) {
	ctx, err := LoadContext()
	if err != nil {
		return "", err
	}
	return ctx.ActiveBackend, nil
}

// ListConfigs returns a list of available config files
func ListConfigs() ([]string, error) {
	var configs []string

	// Check user config directory
	home, _ := os.UserHomeDir()
	userConfigDir := filepath.Join(home, ".config", "resticm")

	if entries, err := os.ReadDir(userConfigDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
				configs = append(configs, filepath.Join(userConfigDir, entry.Name()))
			}
		}
	}

	// Check system config directory
	if entries, err := os.ReadDir("/etc/resticm"); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
				configs = append(configs, filepath.Join("/etc/resticm", entry.Name()))
			}
		}
	}

	return configs, nil
}
