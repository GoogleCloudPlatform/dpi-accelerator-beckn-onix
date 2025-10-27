# ONIX Secrets Key Manager Plugin

The ONIX Secrets Key Manager Plugin provides a secure and efficient solution for managing cryptographic keys within the ONIX ecosystem. It leverages **Google Cloud's Secret Manager** for robust storage of private keys, ensuring the integrity and availability of signing (Ed25519) and encryption (X25519) keys. It also incorporates a caching mechanism to optimize performance by minimizing redundant calls to the Beckn network registry.

This plugin implements the `KeyManager` and `KeyManagerProvider` interface defined by the ONIX plugin framework  (see here [`https://github.com/beckn/beckn-onix/tree/beckn-onix-v1.0-develop/pkg/plugin/definition`](https://github.com/beckn/beckn-onix/tree/beckn-onix-v1.0-develop/pkg/plugin/definition)), enabling seamless integration with other ONIX modules.

## Features

* **Key Generation:** Generates Ed25519 key pairs for signing and X25519 key pairs for encryption.
* **Secure Key Storage:** Stores private keys securely in Google Cloud's Secret Manager.
* **Caching**: Uses the provided cache to improve performance and reduce redundant queries to network.
* **ONIX Integration:** Fully compliant with the ONIX Plugin Framework, ensuring seamless integration and lifecycle management.

## Integration

To integrate the ONIX Secrets Key Manager Plugin into your ONIX application, you will need to perform two steps:

**Step 1: Add the plugin to your plugin configuration file.**

Include the plugin's details in your application's plugin configuration file. Here's an example:

```yaml
plugins:
  secretskeymanager: # Plugin ID
    src: <YOUR_GITHUB_REPO_URL>
    version: v0.0.1 # Managed via git tags.
    path: plugins/secretskeymanager/cmd
```
**Step 2: Configure the desired handler to use the secretskeymanager plugin.**

In the configuration for the handler that requires key management, add the keyManager section, specifying the plugin ID and its configuration. Here's an example:

```yaml
keyManager:
  id: secretskeymanager
  config:
    projectID: your-gcp-project-id
```

## Configuration

The plugin requires the following configuration.

#### Configuration Keys:

* **projectID:** Google Cloud Project ID to access Secret Manager.

