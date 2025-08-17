package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Manager provides centralized configuration management with validation and watching
type Manager struct {
	mu       sync.RWMutex
	config   *Config
	watchers []func(*Config)

	// File watching
	configPath   string
	lastModTime  time.Time
	watchCancel  context.CancelFunc
	watchRunning bool
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		config:   DefaultConfig(),
		watchers: make([]func(*Config), 0),
	}
}

// LoadFromFile loads configuration from a file with validation
func (m *Manager) LoadFromFile(configPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Expand ~ to home directory if present
	if strings.HasPrefix(configPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot expand home directory: %w", err)
		}
		configPath = filepath.Join(home, configPath[2:])
	}

	// Load configuration
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := m.validateConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Apply defaults for missing values
	m.applyDefaults(cfg)

	// Update state
	m.config = cfg
	m.configPath = configPath

	// Update last modified time for file watching
	if stat, err := os.Stat(configPath); err == nil {
		m.lastModTime = stat.ModTime()
	}

	// Notify watchers
	m.notifyWatchers(cfg)

	return nil
}

// LoadFromDefaults loads default configuration
func (m *Manager) LoadFromDefaults() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg := DefaultConfig()
	m.applyDefaults(cfg)

	m.config = cfg
	m.configPath = ""
	m.lastModTime = time.Time{}

	m.notifyWatchers(cfg)
}

// GetConfig returns a copy of the current configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a deep copy to prevent external modifications
	return m.copyConfig(m.config)
}

// UpdateConfig updates the configuration with validation
func (m *Manager) UpdateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate the new configuration
	if err := m.validateConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Apply defaults
	m.applyDefaults(cfg)

	// Update config
	m.config = cfg

	// Notify watchers
	m.notifyWatchers(cfg)

	return nil
}

// SaveToFile saves the current configuration to a file
func (m *Manager) SaveToFile(filePath string) error {
	m.mu.RLock()
	cfg := m.copyConfig(m.config)
	m.mu.RUnlock()

	if err := cfg.SaveConfig(filePath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// Watch starts watching the configuration file for changes
func (m *Manager) Watch(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.configPath == "" {
		return fmt.Errorf("no config file path set")
	}

	if m.watchRunning {
		return fmt.Errorf("already watching configuration file")
	}

	watchCtx, cancel := context.WithCancel(ctx)
	m.watchCancel = cancel
	m.watchRunning = true

	go m.watchConfigFile(watchCtx)

	return nil
}

// StopWatching stops watching the configuration file
func (m *Manager) StopWatching() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.watchCancel != nil {
		m.watchCancel()
		m.watchCancel = nil
	}
	m.watchRunning = false
}

// AddWatcher adds a configuration change watcher
func (m *Manager) AddWatcher(watcher func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.watchers = append(m.watchers, watcher)
}

// GetCredentialPaths returns the credential and token paths with proper expansion
func (m *Manager) GetCredentialPaths() (string, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	credPath, tokenPath := DefaultCredentialPaths()

	if m.config.Credentials != "" {
		credPath = m.expandPath(m.config.Credentials)
	}

	if m.config.Token != "" {
		tokenPath = m.expandPath(m.config.Token)
	}

	return credPath, tokenPath
}

// GetLLMConfig returns LLM configuration
func (m *Manager) GetLLMConfig() LLMConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config.LLM
}

// validateConfig validates the configuration
func (m *Manager) validateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate LLM configuration if enabled
	if cfg.LLM.Enabled {
		if cfg.LLM.Provider == "" && cfg.LLM.Model == "" {
			return fmt.Errorf("LLM is enabled but no model specified")
		}

		if cfg.LLM.Timeout != "" {
			if _, err := time.ParseDuration(cfg.LLM.Timeout); err != nil {
				return fmt.Errorf("invalid LLM timeout: %w", err)
			}
		}
	}

	// Validate layout configuration
	if cfg.Layout.WideBreakpoint.Width <= 0 || cfg.Layout.WideBreakpoint.Height <= 0 {
		return fmt.Errorf("invalid wide breakpoint dimensions")
	}

	if cfg.Layout.MediumBreakpoint.Width <= 0 || cfg.Layout.MediumBreakpoint.Height <= 0 {
		return fmt.Errorf("invalid medium breakpoint dimensions")
	}

	if cfg.Layout.NarrowBreakpoint.Width <= 0 || cfg.Layout.NarrowBreakpoint.Height <= 0 {
		return fmt.Errorf("invalid narrow breakpoint dimensions")
	}

	return nil
}

// applyDefaults applies default values for missing configuration
func (m *Manager) applyDefaults(cfg *Config) {
	if cfg.Keys == (KeyBindings{}) {
		cfg.Keys = DefaultKeyBindings()
	}

	if cfg.Layout == (LayoutConfig{}) {
		cfg.Layout = DefaultLayoutConfig()
	}

	// Apply default template paths if empty (template files take precedence over inline prompts)
	if cfg.LLM.SummarizeTemplate == "" {
		cfg.LLM.SummarizeTemplate = DefaultLLMConfig().SummarizeTemplate
	}

	if cfg.LLM.ReplyTemplate == "" {
		cfg.LLM.ReplyTemplate = DefaultLLMConfig().ReplyTemplate
	}

	if cfg.LLM.LabelTemplate == "" {
		cfg.LLM.LabelTemplate = DefaultLLMConfig().LabelTemplate
	}

	if cfg.LLM.TouchUpTemplate == "" {
		cfg.LLM.TouchUpTemplate = DefaultLLMConfig().TouchUpTemplate
	}
}

// copyConfig creates a deep copy of the configuration
func (m *Manager) copyConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}

	copy := *cfg
	return &copy
}

// expandPath expands ~ to home directory
func (m *Manager) expandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, path[2:])
}

// notifyWatchers notifies all configuration watchers
func (m *Manager) notifyWatchers(cfg *Config) {
	for _, watcher := range m.watchers {
		go watcher(m.copyConfig(cfg))
	}
}

// watchConfigFile watches the configuration file for changes
func (m *Manager) watchConfigFile(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkConfigFileChanges()
		}
	}
}

// checkConfigFileChanges checks if the configuration file has changed
func (m *Manager) checkConfigFileChanges() {
	m.mu.RLock()
	configPath := m.configPath
	lastModTime := m.lastModTime
	m.mu.RUnlock()

	if configPath == "" {
		return
	}

	stat, err := os.Stat(configPath)
	if err != nil {
		return
	}

	if stat.ModTime().After(lastModTime) {
		// File has been modified, reload it
		_ = m.LoadFromFile(configPath)
	}
}
