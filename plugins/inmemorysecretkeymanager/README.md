The ONIX In-Memory Secrets Key Manager Plugin provides a highly secure and efficient solution for managing cryptographic keys within the ONIX ecosystem. It leverages Google Cloud's Secret Manager for persistent, robust storage of private keys, while implementing a two-tiered caching strategy to optimize performance and enhance security.

This plugin implements the KeyManager and KeyManagerProvider interfaces defined by the ONIX plugin framework (see here), enabling seamless integration with other ONIX modules.

Security-First Caching
To prevent sensitive private keys from being exposed in a shared or persistent cache (like Redis), this plugin uses a two-level caching system:

In-Memory Cache (for Private Keys): All private keys (signing and encryption) are cached exclusively in a secure, local in-memory store within the application's process. This cache is thread-safe and has a configurable Time-To-Live (TTL) to ensure keys are periodically re-validated against the source of truth (Google Secret Manager).

Distributed Cache (for Public Network Keys): Non-sensitive public keys of other network participants are stored in the provided distributed cache (e.g., Redis). This minimizes redundant calls to the Beckn network registry, improving lookup performance without compromising security.

Features
Key Generation: Generates Ed25519 key pairs for signing and X25519 key pairs for encryption.

Secure Key Storage: Stores private keys securely in Google Cloud's Secret Manager.

Secure In-Memory Caching: Caches private keys locally with a configurable TTL for high performance and enhanced security.

Network Key Caching: Uses the provided distributed cache to store public keys, reducing redundant network lookups.

ONIX Integration: Fully compliant with the ONIX Plugin Framework for seamless integration.

Integration
To integrate the ONIX In-Memory Secrets Key Manager Plugin into your ONIX application, follow these two steps:

Step 1: Add the plugin to your plugin configuration file.

Include the plugin's details in your application's plugin configuration file.

plugins:
  inmemorysecretkeymanager: # Plugin ID
    src: <YOUR_GITHUB_REPO_URL>
    version: v0.0.1 # Managed via git tags.
    path: plugins/inmemorysecretkeymanager/cmd

Step 2: Configure the desired handler to use the plugin.

In the configuration for the handler that requires key management, add the keyManager section, specifying the plugin ID and its configuration.

keyManager:
  id: inmemorysecretkeymanager
  config:
    projectID: your-gcp-project-id
    privateKeyCacheTTLSeconds: 15 # e.g., 15 Seconds
    publicKeyCacheTTLSeconds: 3600  # e.g., 1 hour

Configuration
The plugin requires the following configuration keys:

projectID: (Required) Your Google Cloud Project ID to access Secret Manager.

privateKeyCacheTTLSeconds: (Optional) The time-to-live in seconds for private keys in the secure in-memory cache. Defaults to 15 (15 Seconds).

publicKeyCacheTTLSeconds: (Optional) The time-to-live in seconds for public network keys in the distributed cache. Defaults to 3600 (1 hour).