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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_Success(t *testing.T) {
	yamlContent := `
goVersion: 1.24
output: ./build
zipFileName: my_plugins.zip
modules:
  - name: app
    repo: https://github.com/example/project.git
    version: v1.0.0
    path: .
    plugins:
      myplugin: cmd/myplugin
    images:
      myimage:
        dockerfile: Dockerfile
        tag: v1
`
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	assert.NoError(t, err)

	config, err := LoadConfig(configFile)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, "1.24", config.GoVersion)
	assert.Equal(t, "./build", config.Output)
	assert.Equal(t, "my_plugins.zip", config.ZipFileName)
	assert.Len(t, config.Modules, 1)
	assert.Equal(t, "app", config.Modules[0].Name)
	assert.Equal(t, "https://github.com/example/project.git", config.Modules[0].Repo)
	assert.Equal(t, "v1.0.0", config.Modules[0].Version)
	assert.Equal(t, ".", config.Modules[0].Path)
	assert.Equal(t, "cmd/myplugin", config.Modules[0].Plugins["myplugin"])
	assert.Equal(t, "app", config.Modules[0].DirName)
}

func TestLoadConfig_Defaults(t *testing.T) {
	yamlContent := `
goVersion: 1.24
modules:
  - name: github.com/example/project
`
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	assert.NoError(t, err)

	config, err := LoadConfig(configFile)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, "./dist", config.Output)
	assert.Equal(t, "plugins_bundle.zip", config.ZipFileName)
	assert.Equal(t, ".", config.Modules[0].Path)
}

func TestLoadConfig_ValidationFails(t *testing.T) {
	testCases := []struct {
		name          string
		yamlContent   string
		expectedError string
	}{
		{
			name:          "Missing GoVersion",
			yamlContent:   `modules: [{name: "test"}]`,
			expectedError: "config validation failed: 'goVersion' field is required",
		},
		{
			name:          "Missing Modules",
			yamlContent:   `goVersion: 1.24`,
			expectedError: "config validation failed: at least one module must be defined in 'modules'",
		},
		{
			name:          "Missing Module Name",
			yamlContent:   `goVersion: 1.24
modules: [{}]`,
			expectedError: "config validation failed: module at index 0 is missing required 'name' field",
		},
		{
			name: "Multiple Modules with Images",
			yamlContent: `goVersion: 1.24
modules:
  - name: app
    images:
      myimage:
        dockerfile: Dockerfile
        tag: v1
  - name: another-app
    images:
      another-image:
        dockerfile: Dockerfile
        tag: v1`,
			expectedError: "config validation failed: only one module can have images",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configFile, []byte(tc.yamlContent), 0644)
			assert.NoError(t, err)

			_, err = LoadConfig(configFile)
			assert.Error(t, err)
			assert.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestLoadConfig_FileNotExist(t *testing.T) {
	_, err := LoadConfig("nonexistent.yaml")
	assert.Error(t, err)
}
