// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// saveAndRestoreDefaultSlog is a helper to manage the global slog.Default logger during tests.
func saveAndRestoreDefaultSlog(t *testing.T) func() {
	t.Helper()
	originalLogger := slog.Default()
	return func() {
		slog.SetDefault(originalLogger)
	}
}

func TestValid_Success(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{name: "valid level INFO", cfg: &Config{Level: "INFO"}},
		{name: "valid level info (lowercase)", cfg: &Config{Level: "info"}},
		{name: "valid level ERROR", cfg: &Config{Level: "ERROR"}},
		{name: "valid level FATAL", cfg: &Config{Level: "FATAL"}},
		{name: "valid level WARN", cfg: &Config{Level: "WARN"}},
		{name: "valid level DEBUG", cfg: &Config{Level: "DEBUG"}},
		{name: "valid level OFF", cfg: &Config{Level: "OFF"}},
		{name: "valid level empty (defaults to INFO in Setup, valid here)", cfg: &Config{Level: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := valid(tt.cfg); err != nil {
				t.Errorf("valid() error = %v, wantErr false", err)
			}
		})
	}
}

func TestValid_Error(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		errText string
	}{
		{name: "nil config", cfg: nil, errText: "config is nil"},
		{name: "invalid level", cfg: &Config{Level: "INVALID"}, errText: "invalid log level: INVALID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := valid(tt.cfg)
			if err == nil {
				t.Fatalf("valid() error = nil, wantErr true with text %q", tt.errText)
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("valid() error text = %q, wantText %q", err.Error(), tt.errText)
			}
		})
	}
}

func TestSetup_Success_Stdout(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *Config
		checkLogLevel slog.Level // Level to check if enabled after setup
		expectEnabled bool       // Whether checkLogLevel is expected to be enabled
	}{
		{
			name:          "INFO level STDOUT",
			cfg:           &Config{Level: "INFO", Target: "STDOUT"},
			checkLogLevel: slog.LevelInfo,
			expectEnabled: true,
		},
		{
			name:          "DEBUG level STDOUT",
			cfg:           &Config{Level: "DEBUG", Target: "STDOUT"},
			checkLogLevel: slog.LevelDebug,
			expectEnabled: true,
		},
		{
			name:          "WARN level STDOUT, check INFO (should be disabled)",
			cfg:           &Config{Level: "WARN", Target: "STDOUT"},
			checkLogLevel: slog.LevelInfo,
			expectEnabled: false,
		},
		{
			name:          "ERROR level STDOUT, check WARN (should be disabled)",
			cfg:           &Config{Level: "ERROR", Target: "STDOUT"},
			checkLogLevel: slog.LevelWarn,
			expectEnabled: false,
		},
		{
			name:          "FATAL level STDOUT, check ERROR (should be enabled)",
			cfg:           &Config{Level: "FATAL", Target: "STDOUT"},
			checkLogLevel: slog.LevelError,
			expectEnabled: true,
		},
		{
			name:          "OFF level STDOUT, check ERROR (should be disabled)",
			cfg:           &Config{Level: "OFF", Target: "STDOUT"},
			checkLogLevel: slog.LevelError,
			expectEnabled: false,
		},
		{
			name:          "OFF level STDOUT, check DEBUG (should be disabled)",
			cfg:           &Config{Level: "OFF", Target: "STDOUT"},
			checkLogLevel: slog.LevelDebug,
			expectEnabled: false,
		},
		{
			name:          "Empty level (defaults to INFO) STDOUT",
			cfg:           &Config{Level: "", Target: "STDOUT"},
			checkLogLevel: slog.LevelInfo,
			expectEnabled: true,
		},
		{
			name:          "Empty target (defaults to STDOUT)",
			cfg:           &Config{Level: "DEBUG", Target: ""},
			checkLogLevel: slog.LevelDebug,
			expectEnabled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer saveAndRestoreDefaultSlog(t)()
			err := Setup(tt.cfg)
			if err != nil {
				t.Fatalf("Setup() unexpected error = %v", err)
			}
			if slog.Default().Enabled(context.Background(), tt.checkLogLevel) != tt.expectEnabled {
				// Capture the actual level of the default logger for better error reporting
				currentHandler := slog.Default().Handler()
				var actualLevelEnabled bool
				if currentHandler != nil { // Check if handler is nil before calling Enabled
					actualLevelEnabled = currentHandler.Enabled(context.Background(), tt.checkLogLevel)
				}
				t.Errorf("Setup() for level %s, target %s. Expected enabled status for level %s to be %v, but got %v",
					tt.cfg.Level, tt.cfg.Target, tt.checkLogLevel, tt.expectEnabled, actualLevelEnabled)
			}
		})
	}
}

func TestSetup_Success_File(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *Config
		checkLogLevel slog.Level
		expectEnabled bool
	}{
		{
			name:          "INFO level FILE target",
			cfg:           &Config{Level: "INFO", Target: "FILE"},
			checkLogLevel: slog.LevelInfo,
			expectEnabled: true,
		},
		{
			name:          "DEBUG level FILE target",
			cfg:           &Config{Level: "DEBUG", Target: "FILE"},
			checkLogLevel: slog.LevelDebug,
			expectEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer saveAndRestoreDefaultSlog(t)()
			logFilePath := filepath.Join(t.TempDir(), "app.log")
			tt.cfg.FilePath = logFilePath

			err := Setup(tt.cfg)
			if err != nil {
				t.Fatalf("Setup() unexpected error = %v", err)
			}

			if slog.Default().Enabled(context.Background(), tt.checkLogLevel) != tt.expectEnabled {
				currentHandler := slog.Default().Handler()
				var actualLevelEnabled bool
				if currentHandler != nil {
					actualLevelEnabled = currentHandler.Enabled(context.Background(), tt.checkLogLevel)
				}
				t.Errorf("Setup() for level %s (FILE), Expected enabled status for level %s to be %v, but got %v",
					tt.cfg.Level, tt.checkLogLevel, tt.expectEnabled, actualLevelEnabled)
			}

			if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
				t.Errorf("Setup() with Target FILE did not create log file: %s", logFilePath)
			}
		})
	}
}

func TestSetup_Error(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		errText string
	}{
		{
			name:    "Invalid level in Setup",
			cfg:     &Config{Level: "INVALID_LEVEL", Target: "STDOUT"},
			errText: "invalid log level: INVALID_LEVEL",
		},
		{
			name:    "Invalid target in Setup",
			cfg:     &Config{Level: "INFO", Target: "INVALID_TARGET"},
			errText: "invalid log target: INVALID_TARGET",
		},
		{
			name:    "Nil config in Setup",
			cfg:     nil,
			errText: "config is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer saveAndRestoreDefaultSlog(t)()
			err := Setup(tt.cfg)
			if err == nil {
				t.Fatalf("Setup() error = nil, wantErr true with text %q", tt.errText)
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("Setup() error text = %q, wantText %q", err.Error(), tt.errText)
			}
		})
	}
}
