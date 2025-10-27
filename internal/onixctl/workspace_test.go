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

func TestNewWorkspace(t *testing.T) {
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	assert.NotNil(t, ws)
	defer ws.Close()

	// Check if the directory was created
	_, err = os.Stat(ws.Path())
	assert.NoError(t, err, "workspace directory should exist")
}

func TestWorkspace_Close(t *testing.T) {
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	assert.NotNil(t, ws)

	path := ws.Path()
	err = ws.Close()
	assert.NoError(t, err)

	// Check if the directory was removed
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err), "workspace directory should not exist after close")
}

func TestWorkspace_Path(t *testing.T) {
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	assert.NotNil(t, ws)
	defer ws.Close()

	// Check that the path is not empty and is absolute
	assert.NotEmpty(t, ws.Path())
	assert.True(t, filepath.IsAbs(ws.Path()), "workspace path should be absolute")
}

func TestRunCommand(t *testing.T) {
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	defer ws.Close()

	err = ws.runCommand(ws.Path(), "ls")
	assert.NoError(t, err)
}
