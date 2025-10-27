# onixctl

`onixctl` is a config drivencommand-line tool for building and pushing ONIX-Adapter and the plugins it consumes. It provides a unified interface for 

1. Sourcing plugins source code from local or github repository.
2. Sourcing adapter  source code from local or github repository.
3. Creating a syncronized dependedency graph for them.
4. Building the plugin bundle and docker service image for Adapter.
5. Optinally push the the Image to a specified artifact registry.
6. Optinally push the plugin bundle to a specified GCS bucket.

## Overview

The tool is split into two main parts:

-   `cmd/onixctl`: This is the entry point of the application. It contains the `main` function that executes the root command.
-   `internal/onixctl`: This is the core of the application. It contains all the logic for parsing the configuration, managing the workspace, building the artifacts, and publishing them.

## How to Use

The `onixctl` tool is configured using a YAML file. By default, it looks for a file named `source.yaml` in the `configs` directory. You can also specify a different configuration file using the `--config` flag.

### Configuration

The configuration file has the following structure:

```yaml
goVersion: "1.24"
modules:
  - name: github.com/example/project
    repo: https://github.com/example/project.git
    version: v1.0.0
    path: .
    plugins:
      myplugin: cmd/myplugin
  - name: app
    path: ./local/app
    images:
      myimage:
        dockerfile: Dockerfile
        tag: v1
```

**Note:** The `name` of each module must match the module name in its `go.mod` file.

### Local Modules

To use a local module, simply omit the `repo` field from the module's configuration. The `path` field should be the path to the module's source code on your local machine.

### Default Values

The following fields have default values if they are not provided in the configuration file:

-   `output`: `./dist`
-   `zipFileName`: `plugins_bundle.zip`
-   `path`: `.`

### Flags

The following flags can be used to override the values in the configuration file:

-   `--registry`: The container registry to push images to.
-   `--output`: The output directory for the build artifacts.
-   `--zipFileName`: The name of the zipped plugin bundle.
-   `--gsPath`: The GCS path to upload the plugin bundle to.

### Example

To run the tool with the default configuration, simply execute the following command from the root of the project:

```bash
./onixctl
```

To run the tool with a custom configuration and override some of the values, you can use the following command:

```bash
./onixctl --config my-config.yaml --registry my-registry.com/project --output ./my-dist
```
