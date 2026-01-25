package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExecutionModeValid(t *testing.T) {
	tests := []struct {
		mode  ExecutionMode
		valid bool
	}{
		{ModeDefault, true},
		{ModeAuto, true},
		{ModeConfirm, true},
		{"invalid", false},
		{"AUTO", false}, // case sensitive
	}

	for _, tt := range tests {
		if got := tt.mode.Valid(); got != tt.valid {
			t.Errorf("ExecutionMode(%q).Valid() = %v, want %v", tt.mode, got, tt.valid)
		}
	}
}

func TestConfigParsesMode(t *testing.T) {
	yamlInput := `
menu:
  - category: Test
    items:
      - name: Auto Item
        run: "echo auto"
        mode: auto
      - name: Confirm Item
        run: "echo confirm"
        mode: confirm
      - name: Default Item
        run: "echo default"
      - name: With Action
        run: "echo test"
        actions:
          - key: x
            name: Delete
            run: "rm file"
            mode: confirm
`
	var cfg Config
	if err := unmarshalConfig([]byte(yamlInput), &cfg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	items := cfg.Menu[0].Items
	if items[0].Mode != ModeAuto {
		t.Errorf("item[0].Mode = %q, want %q", items[0].Mode, ModeAuto)
	}
	if items[1].Mode != ModeConfirm {
		t.Errorf("item[1].Mode = %q, want %q", items[1].Mode, ModeConfirm)
	}
	if items[2].Mode != ModeDefault {
		t.Errorf("item[2].Mode = %q, want %q", items[2].Mode, ModeDefault)
	}
	if items[3].Actions[0].Mode != ModeConfirm {
		t.Errorf("action.Mode = %q, want %q", items[3].Actions[0].Mode, ModeConfirm)
	}
}

func TestConfigValidatesMode(t *testing.T) {
	yamlData := `
menu:
  - category: Test
    items:
      - name: Bad Item
        run: "echo bad"
        mode: invalid_mode
`
	var cfg Config
	if err := unmarshalConfig([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	err := cfg.validate()
	if err == nil {
		t.Error("expected validation error for invalid mode, got nil")
	}
}

func unmarshalConfig(data []byte, cfg *Config) error {
	return yaml.Unmarshal(data, cfg)
}
