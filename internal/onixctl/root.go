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
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	registry    string
	output      string
	zipFileName string
	gsPath      string
	// OsExit is a function that can be mocked in tests.
	OsExit = os.Exit
)

// rootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "onixctl",
	Short: "A unified build and deploy orchestrator for onix.",
	Long: `onixctl is a command-line tool to build Go plugins and Docker images
from a YAML configuration file, ensuring dependency consistency across modules.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runOrchestrator(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			OsExit(1)
		}
	},
}

func init() {
	// Correctly reference the onix directory from where the binary is expected to be run
	defaultCfgPath := "configs/source.yaml"
	if _, err := os.Stat(defaultCfgPath); os.IsNotExist(err) {
		// If not found, assume we might be running from within the onix directory
		defaultCfgPath = "configs/source.yaml"
	}
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultCfgPath, "config file")
	RootCmd.PersistentFlags().StringVar(&registry, "registry", "", "Container registry to push images to")
	RootCmd.PersistentFlags().StringVar(&output, "output", "", "Output directory for artifacts")
	RootCmd.PersistentFlags().StringVar(&zipFileName, "zipFileName", "", "Name of the zipped plugin bundle")
	RootCmd.PersistentFlags().StringVar(&gsPath, "gsPath", "", "GCS path to upload the plugin bundle to")
}

// runOrchestrator is the main logic function of the application.
func runOrchestrator(cmd *cobra.Command) error {
	// 1. Load Config
	config, err := LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override config with flags if they are set
	if cmd.Flags().Changed("registry") {
		config.Registry = registry
	}
	if cmd.Flags().Changed("output") {
		config.Output = output
	}
	if cmd.Flags().Changed("zipFileName") {
		config.ZipFileName = zipFileName
	}
	if cmd.Flags().Changed("gsPath") {
		config.GSPath = gsPath
	}

	fmt.Println("✅ Configuration loaded successfully.")
	fmt.Printf("Go version: %s\n", config.GoVersion)

	// 2. Create Workspace
	fmt.Println("⚙️  Creating temporary workspace...")
	workspace, err := NewWorkspace()
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}
	// uncomment to delete the temporary workspace
	// defer func() {
	// 	fmt.Println("🧹 Cleaning up workspace...")
	// 	if err := workspace.Close(); err != nil {
	// 		fmt.Fprintf(os.Stderr, "Warning: failed to clean up workspace: %v\n", err)
	// 	}
	// }()
	fmt.Printf("Workspace created at: %s\n", workspace.Path())

	// 3. Prepare Modules
	fmt.Println("⚙️  Preparing modules...")
	if err := workspace.PrepareModules(config.Modules); err != nil {
		return fmt.Errorf("failed to prepare modules: %w", err)
	}

	// 4. Setup Go Workspace
	if err := workspace.SetupGoWorkspace(config.Modules, config.GoVersion); err != nil {
		return fmt.Errorf("failed to set up Go workspace: %w", err)
	}

	// 5. Build Artifacts
	fmt.Println("⚙️  Building artifacts...")
	builder, err := NewBuilder(config, workspace.Path())
	if err != nil {
		return fmt.Errorf("failed to initialize builder: %w", err)
	}
	if err := builder.Build(); err != nil {
		return fmt.Errorf("build process failed: %w", err)
	}

	// 6. Publish Artifacts
	fmt.Println("⚙️  Publishing artifacts...")
	publisher := NewPublisher(config)
	if err := publisher.Publish(); err != nil {
		// Publishing is not considered a critical failure
		fmt.Fprintf(os.Stderr, "Warning: failed to publish artifacts: %v\n", err)
	}

	fmt.Println("✅ Build and deploy process completed successfully.")
	return nil
}
