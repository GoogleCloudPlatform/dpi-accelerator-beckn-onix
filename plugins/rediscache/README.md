# ONIX Redis Cache Plugin

The ONIX Redis Cache Plugin provides a Redis-based implementation for Onix. It enables efficient caching of data, improving performance and reducing redundant queries.

This plugin implements the Cache and CacheProvider interface defined by the ONIX plugin framework(see here [`https://github.com/beckn/beckn-onix/tree/beckn-onix-v1.0-develop/pkg/plugin/definition`](https://github.com/beckn/beckn-onix/tree/beckn-onix-v1.0-develop/pkg/plugin/definition)), enabling seamless integration with other ONIX modules.

## Features

* **Redis Backend:** Utilizes Redis for high-performance data caching.
* **Data Storage and Retrieval:** Provides methods for setting, getting, and deleting cached data.
* **Time-to-Live (TTL):** Supports setting TTL for cached data.
* **Cache Clearing:** Allows clearing all data from the cache.
* **ONIX Integration:** Fully compliant with the ONIX Plugin Framework, implementing the Cache interface.

## Integration

To integrate the ONIX Redis Cache Plugin into your ONIX application, you will need to perform two steps:

**Step 1: Add the plugin to your plugin configuration file.**

Include the plugin's details in your application's plugin configuration file. Here's an example:

```yaml
plugins:
  rediscache: # Plugin ID
    src: <YOUR_GITHUB_REPO_URL>
    version: v0.0.1 # Managed via git tags.
    path: plugins/rediscache/cmd
```
**Step 2: Configure the desired handler to use the rediscache plugin.**

In the configuration for the handler that requires caching, add the cache section, specifying the plugin ID and its configuration. Here's an example:

```yaml
cache:
  id: rediscache
  config:
    addr: "localhost:6379"
    password: "" # Optional
```

## Configuration

The plugin requires the following configuration.

#### Configuration Keys:

* **addr:** The Redis server address (e.g., localhost:6379).
* **password:** (Optional) The Redis server password.