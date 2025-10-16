// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package onixctl

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCmd creates a new cobra command for testing and resets the global flag variables.
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	// Reset global flag variables before each test
	cfgFile = ""
	registry = ""
	output = ""
	zipFileName = ""
	gsPath = ""

	// Define flags for the test command to mirror the real command's flags
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "test-config.yaml", "config file")
	cmd.PersistentFlags().StringVar(&registry, "registry", "", "Container registry")
	cmd.PersistentFlags().StringVar(&output, "output", "", "Output directory")
	cmd.PersistentFlags().StringVar(&zipFileName, "zipFileName", "", "Zip file name")
	cmd.PersistentFlags().StringVar(&gsPath, "gsPath", "", "GCS path")

	return cmd
}

func TestRunOrchestrator_ConfigLoadFail(t *testing.T) {

	cfgFile = "non-existent-config.yaml"

	cmd := newTestCmd()

	err := runOrchestrator(cmd)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "failed to load configuration")

}

func TestRootCmd_Run_Error(t *testing.T) {

	// Mock os.Exit to prevent the test from terminating

	var exitCode int

	originalOsExit := OsExit

	OsExit = func(code int) {

		exitCode = code

	}

	defer func() { OsExit = originalOsExit }()

	// Point to a non-existent config file to trigger an error

	RootCmd.SetArgs([]string{"--config", "non-existent-config.yaml"})

	// Capture stderr to check the error message

	oldStderr := os.Stderr

	r, w, _ := os.Pipe()

	os.Stderr = w

	if err := RootCmd.Execute(); err != nil {
		// Handle the error here, perhaps log it or fail the test
		t.Errorf("RootCmd.Execute failed: %v", err)
	}

	// Execute() does not return an error here because the Run function handles it and calls OsExit.

	// We are testing the side effect (OsExit call), not the return value of Execute().

	if err := w.Close(); err != nil {
		// Handle the error here, perhaps log it or fail the test
		t.Errorf("RootCmd.Execute failed: %v", err)
	}

	var buf bytes.Buffer

	if _, err := io.Copy(&buf, r); err != nil {
		// Handle the error here, perhaps log it or fail the test
		t.Errorf("RootCmd.Execute failed: %v", err)
	}

	os.Stderr = oldStderr

	assert.Equal(t, 1, exitCode, "OsExit should be called with 1 on error")

	assert.True(t, strings.HasPrefix(buf.String(), "Error:"), "Error message should be printed to stderr")

}
