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
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
)

func TestNewWorkspace(t *testing.T) {
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	assert.NotNil(t, ws)
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

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
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Check that the path is not empty and is absolute
	assert.NotEmpty(t, ws.Path())
	assert.True(t, filepath.IsAbs(ws.Path()), "workspace path should be absolute")
}

func TestRunCommand(t *testing.T) {
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	err = ws.runCommand(ws.Path(), "ls")
	assert.NoError(t, err)

	err = ws.runCommand(ws.Path(), "non-existent-command")
	assert.Error(t, err, "running a non-existent command should return an error")
}

func TestWorkspace_PrepareModules_Local(t *testing.T) {
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Create a temporary directory for the local module
	localModuleDir, err := os.MkdirTemp("", "local-module-*")
	assert.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(localModuleDir); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Create a file in the local module directory
	err = os.WriteFile(filepath.Join(localModuleDir, "test.txt"), []byte("test"), 0644)
	assert.NoError(t, err)

	modules := []Module{
		{
			Name:    "local-module",
			Path:    localModuleDir,
			DirName: "local-module",
		},
	}

	err = ws.PrepareModules(modules)
	assert.NoError(t, err)

	// Check if the module was copied to the workspace
	_, err = os.Stat(filepath.Join(ws.Path(), "local-module", "test.txt"))
	assert.NoError(t, err, "module file should exist in the workspace")
}

func TestWorkspace_PrepareModules_Remote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary directory for the remote repository
	remoteRepoDir, err := os.MkdirTemp("", "remote-repo-*")
	assert.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(remoteRepoDir); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Initialize a new git repository
	repo, err := git.PlainInit(remoteRepoDir, false)
	assert.NoError(t, err)

	// Create a file and commit it
	err = os.WriteFile(filepath.Join(remoteRepoDir, "test.txt"), []byte("test"), 0644)
	assert.NoError(t, err)

	w, err := repo.Worktree()
	assert.NoError(t, err)
	_, err = w.Add("test.txt")
	assert.NoError(t, err)

	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	assert.NoError(t, err)

	// Create a new workspace
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	modules := []Module{
		{
			Name:    "remote-module",
			Repo:    remoteRepoDir,
			Path:    ".",
			DirName: "remote-module",
		},
	}

	err = ws.PrepareModules(modules)
	assert.NoError(t, err)

	// Check if the module was cloned to the workspace
	_, err = os.Stat(filepath.Join(ws.Path(), "remote-module", "test.txt"))
	assert.NoError(t, err, "module file should exist in the workspace")
}

func TestWorkspace_PrepareModules_Remote_InvalidVersion(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary directory for the remote repository
	remoteRepoDir, err := os.MkdirTemp("", "remote-repo-*")
	assert.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(remoteRepoDir); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Initialize a new git repository
	repo, err := git.PlainInit(remoteRepoDir, false)
	assert.NoError(t, err)

	// Create a file and commit it
	w, err := repo.Worktree()
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(remoteRepoDir, "dummy.txt"), []byte("hello"), 0644)
	assert.NoError(t, err)
	_, err = w.Add("dummy.txt")
	assert.NoError(t, err)
	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	assert.NoError(t, err)

	// Create a new workspace
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	modules := []Module{
		{
			Name:    "remote-module",
			Repo:    remoteRepoDir,
			Path:    ".",
			DirName: "remote-module",
			Version: "non-existent-version",
		},
	}

	err = ws.PrepareModules(modules)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve version")
}

func TestWorkspace_SetupGoWorkspace(t *testing.T) {
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	modules := []Module{
		{DirName: "module-a"},
		{DirName: "module-b"},
	}

	// Create dummy module directories and go.mod files
	for _, m := range modules {
		modulePath := filepath.Join(ws.Path(), m.DirName)
		err := os.MkdirAll(modulePath, 0755)
		assert.NoError(t, err)
		goModContent := []byte("module example.com/onix/" + m.DirName + "\n\ngo 1.21.0\n")
		err = os.WriteFile(filepath.Join(modulePath, "go.mod"), goModContent, 0644)
		assert.NoError(t, err)
	}

	goVersion := "1.21.0"
	// Use a mock runner to avoid executing 'go' command
	ws.runner = &MockCommandRunner{}
	err = ws.SetupGoWorkspace(modules, goVersion)
	assert.NoError(t, err)

	// Check if go.work was created
	goWorkPath := filepath.Join(ws.Path(), "go.work")
	_, err = os.Stat(goWorkPath)
	assert.NoError(t, err, "go.work file should exist")

	// Check go.work content
	content, err := os.ReadFile(goWorkPath)
	assert.NoError(t, err)
	expectedContent := "go 1.21.0\n\nuse (\n\t\"./module-a\"\n\t\"./module-b\"\n)\n"
	assert.Equal(t, expectedContent, string(content))
}

func TestWorkspace_SetupGoWorkspace_NoGoMod(t *testing.T) {
	ws, err := NewWorkspace()
	assert.NoError(t, err)
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	modules := []Module{
		{DirName: "module-a", Name: "module-a"},
	}

	// Create a dummy module directory without a go.mod file
	modulePath := filepath.Join(ws.Path(), "module-a")
	err = os.MkdirAll(modulePath, 0755)
	assert.NoError(t, err)

	// Simulate failure in go work sync
	ws.runner = &MockCommandRunner{
		ShouldError: assert.AnError,
	}

	goVersion := "1.21.0"
	err = ws.SetupGoWorkspace(modules, goVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync workspace dependencies")
}
