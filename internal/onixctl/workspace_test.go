// Copyright 2026 Google LLC
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
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

func TestNewWorkspace(t *testing.T) {
	ws, err := NewWorkspace()
	if err != nil {
		t.Fatalf("NewWorkspace() failed: %v", err)
	}
	if ws == nil {
		t.Errorf("NewWorkspace() got nil, want non-nil")
	}
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Check if the directory was created
	_, err = os.Stat(ws.Path())
	if err != nil {
		t.Errorf("os.Stat(%s) failed: %v, workspace directory should exist", ws.Path(), err)
	}
}

func TestWorkspace_Close(t *testing.T) {
	ws, err := NewWorkspace()
	if err != nil {
		t.Fatalf("NewWorkspace() failed: %v", err)
	}
	if ws == nil {
		t.Errorf("NewWorkspace() got nil, want non-nil")
	}

	path := ws.Path()
	err = ws.Close()
	if err != nil {
		t.Fatalf("ws.Close() failed: %v", err)
	}

	// Check if the directory was removed
	_, err = os.Stat(path)
	if !os.IsNotExist(err) {
		t.Errorf("os.Stat(%s) got error %v, want os.IsNotExist", path, err)
	}
}

func TestWorkspace_Path(t *testing.T) {
	ws, err := NewWorkspace()
	if err != nil {
		t.Fatalf("NewWorkspace() failed: %v", err)
	}
	if ws == nil {
		t.Errorf("NewWorkspace() got nil, want non-nil")
	}
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Check that the path is not empty and is absolute
	if ws.Path() == "" {
		t.Errorf("ws.Path() got empty string, want non-empty")
	}
	if !filepath.IsAbs(ws.Path()) {
		t.Errorf("ws.Path() got %q, want an absolute path", ws.Path())
	}
}

func TestRunCommand(t *testing.T) {
	ws, err := NewWorkspace()
	if err != nil {
		t.Fatalf("NewWorkspace() failed: %v", err)
	}
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	err = ws.runCommand(ws.Path(), "ls")
	if err != nil {
		t.Errorf("ws.runCommand(\"ls\") failed: %v", err)
	}

	err = ws.runCommand(ws.Path(), "non-existent-command")
	if err == nil {
		t.Errorf("ws.runCommand(\"non-existent-command\") got nil, want error")
	}
}

func TestWorkspace_PrepareModules_Local(t *testing.T) {
	ws, err := NewWorkspace()
	if err != nil {
		t.Fatalf("NewWorkspace() failed: %v", err)
	}
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Create a temporary directory for the local module
	localModuleDir, err := os.MkdirTemp("", "local-module-*")
	if err != nil {
		t.Fatalf("os.MkdirTemp failed: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(localModuleDir); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Create a file in the local module directory
	err = os.WriteFile(filepath.Join(localModuleDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}

	modules := []Module{
		{
			Name:    "local-module",
			Path:    localModuleDir,
			DirName: "local-module",
		},
	}

	err = ws.PrepareModules(modules)
	if err != nil {
		t.Fatalf("ws.PrepareModules failed: %v", err)
	}

	// Check if the module was copied to the workspace
	_, err = os.Stat(filepath.Join(ws.Path(), "local-module", "test.txt"))
	if err != nil {
		t.Errorf("os.Stat failed for copied module file: %v", err)
	}
}

func TestWorkspace_PrepareModules_Remote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary directory for the remote repository
	remoteRepoDir, err := os.MkdirTemp("", "remote-repo-*")
	if err != nil {
		t.Fatalf("os.MkdirTemp failed: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(remoteRepoDir); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Initialize a new git repository
	repo, err := git.PlainInit(remoteRepoDir, false)
	if err != nil {
		t.Fatalf("git.PlainInit failed: %v", err)
	}

	// Create a file and commit it
	err = os.WriteFile(filepath.Join(remoteRepoDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("repo.Worktree failed: %v", err)
	}
	_, err = w.Add("test.txt")
	if err != nil {
		t.Fatalf("w.Add failed: %v", err)
	}

	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("w.Commit failed: %v", err)
	}

	// Create a new workspace
	ws, err := NewWorkspace()
	if err != nil {
		t.Fatalf("NewWorkspace() failed: %v", err)
	}
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
	if err != nil {
		t.Fatalf("ws.PrepareModules failed: %v", err)
	}

	// Check if the module was cloned to the workspace
	_, err = os.Stat(filepath.Join(ws.Path(), "remote-module", "test.txt"))
	if err != nil {
		t.Errorf("os.Stat failed for cloned module file: %v", err)
	}
}

func TestWorkspace_PrepareModules_LocalDockerfile(t *testing.T) {
	ws, err := NewWorkspace()
	if err != nil {
		t.Fatalf("NewWorkspace() failed: %v", err)
	}
	defer func() {
		if err := ws.Close(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Temporarily chdir to a temp directory since Bazel tests run in readonly directories.
	originalWd, _ := os.Getwd()
	tmpWd := t.TempDir()
	require.NoError(t, os.Chdir(tmpWd))
	defer os.Chdir(originalWd) // ensure clean state

	// Create a dummy local module
	localModuleDir := filepath.Join(tmpWd, "local-module")
	require.NoError(t, os.MkdirAll(localModuleDir, 0755))

	// Create a dummy dockerfile locally (in the current working directory)
	tempDockerfile := "TestDockerfile.custom"
	require.NoError(t, os.WriteFile(tempDockerfile, []byte("FROM scratch"), 0644))

	modules := []Module{
		{
			Name:    "local-module",
			Path:    localModuleDir,
			DirName: "app",
			Images: map[string]Image{
				"myimage": {Dockerfile: tempDockerfile, Tag: "v1"},
			},
		},
	}

	err = ws.PrepareModules(modules)
	if err != nil {
		t.Fatalf("ws.PrepareModules failed: %v", err)
	}

	// Check that the file was actually copied into the module workspace
	copiedDockerfilePath := filepath.Join(ws.Path(), "app", tempDockerfile)
	if _, err := os.Stat(copiedDockerfilePath); err != nil {
		t.Errorf("expected Dockerfile at %s, but os.Stat failed: %v", copiedDockerfilePath, err)
	}

	content, err := os.ReadFile(copiedDockerfilePath)
	if err != nil {
		t.Fatalf("os.ReadFile failed: %v", err)
	}
	if string(content) != "FROM scratch" {
		t.Errorf("copied content got %q, want %q", string(content), "FROM scratch")
	}
}

func TestWorkspace_PrepareModules_Remote_InvalidVersion(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary directory for the remote repository
	remoteRepoDir, err := os.MkdirTemp("", "remote-repo-*")
	if err != nil {
		t.Fatalf("os.MkdirTemp failed: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(remoteRepoDir); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()

	// Initialize a new git repository
	repo, err := git.PlainInit(remoteRepoDir, false)
	if err != nil {
		t.Fatalf("git.PlainInit failed: %v", err)
	}

	// Create a file and commit it
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("repo.Worktree failed: %v", err)
	}
	err = os.WriteFile(filepath.Join(remoteRepoDir, "dummy.txt"), []byte("hello"), 0644)
	if err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}
	_, err = w.Add("dummy.txt")
	if err != nil {
		t.Fatalf("w.Add failed: %v", err)
	}
	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("w.Commit failed: %v", err)
	}

	// Create a new workspace
	ws, err := NewWorkspace()
	if err != nil {
		t.Fatalf("NewWorkspace() failed: %v", err)
	}
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
	if err == nil {
		t.Errorf("PrepareModules(%v) got nil, want error", modules)
	} else if wantErr := "failed to resolve version"; !strings.Contains(err.Error(), wantErr) {
		t.Errorf("PrepareModules(%v) got error %v, want error containing %q", modules, err, wantErr)
	}
}

func TestWorkspace_SetupGoWorkspace(t *testing.T) {
	ws, err := NewWorkspace()
	if err != nil {
		t.Fatalf("NewWorkspace() failed: %v", err)
	}
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
		if err != nil {
			t.Fatalf("os.MkdirAll(%s) failed: %v", modulePath, err)
		}
		goModContent := []byte("module example.com/onix/" + m.DirName + "\n\ngo 1.21.0\n")
		err = os.WriteFile(filepath.Join(modulePath, "go.mod"), goModContent, 0644)
		if err != nil {
			t.Fatalf("os.WriteFile(%s) failed: %v", filepath.Join(modulePath, "go.mod"), err)
		}
	}

	goVersion := "1.21.0"
	// Use a mock runner to avoid executing 'go' command
	ws.runner = &MockCommandRunner{}
	err = ws.SetupGoWorkspace(modules, goVersion)
	if err != nil {
		t.Fatalf("ws.SetupGoWorkspace failed: %v", err)
	}

	// Check if go.work was created
	goWorkPath := filepath.Join(ws.Path(), "go.work")
	_, err = os.Stat(goWorkPath)
	if err != nil {
		t.Errorf("os.Stat(%s) failed: %v, go.work file should exist", goWorkPath, err)
	}

	// Check go.work content
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) failed: %v", goWorkPath, err)
	}
	expectedContent := "go 1.21.0\n\nuse (\n\t\"./module-a\"\n\t\"./module-b\"\n)\n"
	if string(content) != expectedContent {
		t.Errorf("go.work content got %q, want %q", string(content), expectedContent)
	}
}

func TestWorkspace_SetupGoWorkspace_NoGoMod(t *testing.T) {
	ws, err := NewWorkspace()
	if err != nil {
		t.Fatalf("NewWorkspace() failed: %v", err)
	}
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
	if err != nil {
		t.Fatalf("os.MkdirAll(%s) failed: %v", modulePath, err)
	}

	// Simulate failure in go work sync
	ws.runner = &MockCommandRunner{
		ShouldError: fmt.Errorf("go work sync failed"),
	}

	goVersion := "1.21.0"
	err = ws.SetupGoWorkspace(modules, goVersion)
	if err == nil {
		t.Errorf("SetupGoWorkspace(%v, %s) got nil, want error", modules, goVersion)
	} else if wantErr := "failed to sync workspace dependencies"; !strings.Contains(err.Error(), wantErr) {
		t.Errorf("SetupGoWorkspace(%v, %s) got error %v, want error containing %q", modules, goVersion, err, wantErr)
	}
}

func TestWorkspace_PrepareModules_CopyError(t *testing.T) {
	ws, _ := NewWorkspace()
	defer ws.Close()

	// Mock runner that returns an error
	ws.runner = &MockCommandRunner{ShouldError: fmt.Errorf("cp failed")}

	modules := []Module{
		{Name: "fail-module", Path: "/invalid/path", DirName: "fail"},
	}

	err := ws.PrepareModules(modules)
	if err == nil {
		t.Errorf("PrepareModules(%v) got nil, want error", modules)
	} else if wantErr := "failed to copy local module"; !strings.Contains(err.Error(), wantErr) {
		t.Errorf("PrepareModules(%v) got error %v, want error containing %q", modules, err, wantErr)
	}
}

func TestWorkspace_SetupGoWorkspace_ModTidyError(t *testing.T) {
	ws, _ := NewWorkspace()
	defer ws.Close()

	modules := []Module{{DirName: "module-a"}}
	modulePath := filepath.Join(ws.Path(), "module-a")
	os.MkdirAll(modulePath, 0755)
	os.WriteFile(filepath.Join(modulePath, "go.mod"), []byte("module a"), 0644)

	// Configure mock to PASS "go work sync" but FAIL "go mod tidy"
	ws.runner = &MockCommandRunner{
		FailOnArgs:  []string{"go", "mod", "tidy"}, // Look for these specific args
		ShouldError: fmt.Errorf("mock tidy failure"),
	}

	err := ws.SetupGoWorkspace(modules, "1.21.0")
	if err == nil {
		t.Errorf("SetupGoWorkspace got nil, want error")
	} else if wantErr := "failed to run go mod tidy"; !strings.Contains(err.Error(), wantErr) {
		t.Errorf("SetupGoWorkspace got error %v, want error containing %q", err, wantErr)
	}
}
