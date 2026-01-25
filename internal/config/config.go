package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Reserved keys that cannot be used for action bindings
var reservedKeys = map[string]bool{
	"q": true, "Q": true,
	"tab": true, "enter": true, "esc": true,
	"j": true, "k": true, "g": true, "G": true,
	"up": true, "down": true, "home": true, "end": true,
	"y": true, "Y": true, "n": true, "N": true,
}

// ExecutionMode defines how a menu item or action should be executed
type ExecutionMode string

const (
	// ModeDefault requires explicit Enter/action key to execute
	ModeDefault ExecutionMode = ""
	// ModeAuto executes automatically when the item is selected/navigated to
	ModeAuto ExecutionMode = "auto"
	// ModeConfirm requires user confirmation (y/n) before executing
	ModeConfirm ExecutionMode = "confirm"
)

// Valid returns true if the mode is a recognized value
func (m ExecutionMode) Valid() bool {
	switch m {
	case ModeDefault, ModeAuto, ModeConfirm:
		return true
	}
	return false
}

type Config struct {
	Status StatusConfig     `yaml:"status"`
	Menu   []CategoryConfig `yaml:"menu"`
}

// StatusConfig defines how to fetch status indicators
type StatusConfig struct {
	Command  string        `yaml:"command"`
	Interval time.Duration `yaml:"interval"`
}

// CategoryConfig defines a menu category with items
type CategoryConfig struct {
	Category string       `yaml:"category"`
	Items    []ItemConfig `yaml:"items"`
}

// ItemConfig defines a menu item
type ItemConfig struct {
	Name     string         `yaml:"name"`
	Run      string         `yaml:"run"`
	Open     string         `yaml:"open"`
	Mode     ExecutionMode  `yaml:"mode,omitempty"`
	Interval time.Duration  `yaml:"interval,omitempty"` // Re-run command periodically while focused
	Actions  []ActionConfig `yaml:"actions"`
}

// ActionConfig defines a keybinding action
type ActionConfig struct {
	Key  string        `yaml:"key"`
	Name string        `yaml:"name"`
	Run  string        `yaml:"run"`
	Open string        `yaml:"open"`
	Mode ExecutionMode `yaml:"mode,omitempty"`
}

func Load() (*Config, error) {
	// Try multiple config locations
	paths := []string{
		"config.yaml",
		"config.yml",
		filepath.Join(os.Getenv("HOME"), ".config", "quarterdeck", "config.yaml"),
		"/etc/quarterdeck/config.yaml",
	}

	var configPath string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			configPath = p
			break
		}
	}

	if configPath == "" {
		// Return default config if no file found
		return defaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		// Provide helpful hints for common YAML errors
		errMsg := err.Error()
		hint := getYAMLErrorHint(errMsg)
		return nil, fmt.Errorf("failed to parse config file %s: %w%s", configPath, err, hint)
	}

	// Validate config
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Set default interval if not specified
	if cfg.Status.Interval == 0 {
		cfg.Status.Interval = 5 * time.Second
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if err := c.validateMenuNotEmpty(); err != nil {
		return err
	}
	if err := c.validateStatusInterval(); err != nil {
		return err
	}
	return c.validateCategories()
}

func (c *Config) validateMenuNotEmpty() error {
	totalItems := 0
	for _, cat := range c.Menu {
		totalItems += len(cat.Items)
	}
	if totalItems == 0 {
		return fmt.Errorf("menu must have at least one item")
	}
	return nil
}

func (c *Config) validateStatusInterval() error {
	if c.Status.Interval < 0 {
		return fmt.Errorf("status interval cannot be negative: %v", c.Status.Interval)
	}
	if c.Status.Interval > 0 && c.Status.Interval < 1*time.Second {
		return fmt.Errorf("status interval too short (minimum 1s): %v", c.Status.Interval)
	}
	return nil
}

func (c *Config) validateCategories() error {
	for _, cat := range c.Menu {
		if cat.Category == "" {
			return fmt.Errorf("category name cannot be empty")
		}
		for _, item := range cat.Items {
			if err := validateItem(item, cat.Category); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateItem(item ItemConfig, category string) error {
	if strings.TrimSpace(item.Name) == "" {
		return fmt.Errorf("item name cannot be empty in category %q", category)
	}
	if !item.Mode.Valid() {
		return fmt.Errorf("invalid mode %q for item %q (valid: auto, confirm)", item.Mode, item.Name)
	}
	if item.Interval > 0 && item.Interval < 1*time.Second {
		return fmt.Errorf("interval too short for item %q (minimum 1s): %v", item.Name, item.Interval)
	}
	if item.Interval > 0 && item.Run == "" {
		return fmt.Errorf("interval requires a run command for item %q", item.Name)
	}
	if item.Run == "" && item.Open == "" && len(item.Actions) == 0 {
		return fmt.Errorf("item %q in category %q has no action (needs run, open, or actions)", item.Name, category)
	}
	return validateItemActions(item)
}

func validateItemActions(item ItemConfig) error {
	seenKeys := make(map[string]bool)
	for _, action := range item.Actions {
		if err := validateAction(action, item.Name, seenKeys); err != nil {
			return err
		}
		seenKeys[action.Key] = true
	}
	return nil
}

func validateAction(action ActionConfig, itemName string, seenKeys map[string]bool) error {
	if !action.Mode.Valid() {
		return fmt.Errorf("invalid mode %q for action %q on item %q (valid: auto, confirm)", action.Mode, action.Name, itemName)
	}
	if len(action.Key) != 1 {
		return fmt.Errorf("action key must be single character, got %q in item %q", action.Key, itemName)
	}
	if reservedKeys[action.Key] {
		return fmt.Errorf("action key %q is reserved for application controls in item %q", action.Key, itemName)
	}
	if seenKeys[action.Key] {
		return fmt.Errorf("duplicate action key %q in item %q", action.Key, itemName)
	}
	if action.Run == "" && action.Open == "" {
		return fmt.Errorf("action %q on item %q has no command (needs run or open)", action.Name, itemName)
	}
	return nil
}

func defaultConfig() *Config {
	return &Config{
		Status: StatusConfig{
			Command:  "echo '[{\"label\": \"Status\", \"value\": \"OK\", \"status\": \"ok\"}]'",
			Interval: 5 * time.Second,
		},
		Menu: []CategoryConfig{
			{
				Category: "System",
				Items: []ItemConfig{
					{
						Name: "Disk Usage",
						Run:  "df -h",
						Mode: ModeAuto,
					},
					{
						Name: "Processes",
						Run:  "ps aux | head -20",
					},
				},
			},
		},
	}
}

// getYAMLErrorHint returns a helpful hint for common YAML parsing errors
func getYAMLErrorHint(errMsg string) string {
	switch {
	case strings.Contains(errMsg, "did not find expected key"):
		return "\nHint: Check for unclosed quotes or missing colons"
	case strings.Contains(errMsg, "found character that cannot start any token"):
		return "\nHint: YAML doesn't allow tabs - use spaces for indentation"
	case strings.Contains(errMsg, "mapping values are not allowed"):
		return "\nHint: Check indentation - items should be indented under their parent"
	default:
		return ""
	}
}
