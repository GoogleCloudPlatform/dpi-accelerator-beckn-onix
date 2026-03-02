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
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Level    string
	Target   string
	FilePath string
}

// Setup initializes the global slog logger with the specified level.
func Setup(cfg *Config) error {
	if err := valid(cfg); err != nil {
		return err
	}
	var level slog.Level
	switch strings.ToUpper(cfg.Level) {
	case "FATAL", "ERROR": // slog doesn't have FATAL, maps to ERROR. We'd os.Exit(1) after logging fatal.
		level = slog.LevelError // Use slog.LevelError for both FATAL and ERROR
	case "WARN":
		level = slog.LevelWarn
	case "INFO":
		level = slog.LevelInfo
	case "DEBUG":
		level = slog.LevelDebug
	case "OFF":
		level = slog.Level(slog.LevelError + 100) // Effectively disable logging by setting a very high level
	default:
		slog.Warn("Invalid log level specified, defaulting to INFO", "specified_level", cfg.Level)
		level = slog.LevelInfo
	}

	var handler slog.Handler
	switch strings.ToUpper(cfg.Target) {
	case "FILE":
		path := "app.log"
		if cfg.FilePath != "" {
			path = cfg.FilePath
		}
		logFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		handler = slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: level})
	case "STDOUT", "": // Default to stdout if target is not specified or empty
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	default:
		return fmt.Errorf("invalid log target: %s", cfg.Target)
	}

	slog.SetDefault(slog.New(handler))
	// This log might not appear if the level is set higher than INFO by default before this runs
	slog.Log(context.Background(), level, "Logger initialized", "configured_level", level.String())
	return nil
}

// valid checks if the log level in the configuration is valid.
func valid(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	switch strings.ToUpper(cfg.Level) {
	case "FATAL", "ERROR", "WARN", "INFO", "DEBUG", "OFF", "":
		return nil
	default:
		return fmt.Errorf("invalid log level: %s", cfg.Level)
	}
}
