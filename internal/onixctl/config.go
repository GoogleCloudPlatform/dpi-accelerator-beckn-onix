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

package onixctl

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the structure of the YAML configuration file.
type Config struct {
	Modules     []Module `yaml:"modules"`
	Registry    string   `yaml:"registry,omitempty"`
	Output      string   `yaml:"output,omitempty"`
	ZipFileName string   `yaml:"zipFileName,omitempty"`
	GSPath      string   `yaml:"gsPath,omitempty"`
	GoVersion   string   `yaml:"goVersion"`
}

// Module represents a single Go module, which can be local or from a Git repository.
type Module struct {
	Repo    string            `yaml:"repo,omitempty"`
	Version string            `yaml:"version,omitempty"`
	Path    string            `yaml:"path,omitempty"`
	Name    string            `yaml:"name"`
	Plugins map[string]string `yaml:"plugins,omitempty"`
	Images  map[string]Image  `yaml:"images,omitempty"`
	DirName string            `yaml:"-"` // Should not be read from yaml
}

// Image represents a Docker image to be built.
type Image struct {
	Dockerfile string `yaml:"dockerfile"`
	Tag        string `yaml:"tag"`
}

// LoadConfig reads the YAML configuration file, validates it, and returns the Config struct.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config YAML: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	setDefaults(&config)

	// Set DirName for each module
	var moduleCount int
	for i := range config.Modules {
		if len(config.Modules[i].Images) > 0 {
			config.Modules[i].DirName = "app"
		} else {
			config.Modules[i].DirName = fmt.Sprintf("module%d", moduleCount)
			moduleCount++
		}
	}

	return &config, nil
}

// validateConfig checks for required fields and logical consistency.
func validateConfig(config *Config) error {
	if config.GoVersion == "" {
		return fmt.Errorf("'goVersion' field is required")
	}

	if len(config.Modules) == 0 {
		return fmt.Errorf("at least one module must be defined in 'modules'")
	}

	var imageModuleFound bool
	for i, module := range config.Modules {
		if module.Name == "" {
			return fmt.Errorf("module at index %d is missing required 'name' field", i)
		}
		if len(module.Images) > 0 {
			if imageModuleFound {
				return fmt.Errorf("only one module can have images")
			}
			imageModuleFound = true
		}
	}

	return nil
}

// setDefaults sets default values for optional fields.
func setDefaults(config *Config) {
	if config.Output == "" {
		config.Output = "./dist"
	}
	if config.ZipFileName == "" {
		config.ZipFileName = "plugins_bundle.zip"
	}
	for i := range config.Modules {
		if config.Modules[i].Path == "" {
			config.Modules[i].Path = "."
		}
	}
}
