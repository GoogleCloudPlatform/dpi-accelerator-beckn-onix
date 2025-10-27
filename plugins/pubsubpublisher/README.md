# ONIX Google Cloud Pub/Sub Publisher Plugin

The ONIX Google Cloud Pub/Sub Publisher Plugin provides a Google Cloud Pub/Sub-based implementation for publishing messages within the ONIX platform. This plugin allows ONIX modules to publish messages to specified Pub/Sub topics, enabling asynchronous communication and event-driven architectures.

This plugin implements the Publisher and PublisherProvider interface defined by the ONIX plugin framework(see here [`https://github.com/beckn/beckn-onix/tree/beckn-onix-v1.0-develop/pkg/plugin/definition`](https://github.com/beckn/beckn-onix/tree/beckn-onix-v1.0-develop/pkg/plugin/definition)), enabling seamless integration with other ONIX modules.

## Features

* **Google Cloud Pub/Sub Integration:** Uses Google Cloud Pub/Sub as the message broker.
* **Topic Management:** Supports publishing to multiple configured Pub/Sub topics.
* **Message Publishing:** Provides a method to publish byte messages to specified topics.
* **ONIX Integration:** Designed for seamless integration with the ONIX Plugin Framework.

## Integration

To integrate the ONIX Google Cloud Pub/Sub Publisher Plugin into your ONIX application, you will need to perform two steps:

**Step 1: Add the plugin to your plugin configuration file.**

Include the plugin's details in your application's plugin configuration file. Here's an example:

```yaml
plugins:
  pubsubpublisher: # Plugin ID
    src: <YOUR_GITHUB_REPO_URL>
    version: v0.0.1 # Managed via git tags.
    path: plugins/pubsubpublisher/cmd
```
**Step 2: Configure the desired handler to use the pubsubpublisher plugin.**

In the configuration for the handler that requires message publishing, add the publisher section, specifying the plugin ID and its configuration. Here's an example:

```yaml
publisher:
  id: pubsubpublisher
  config:
    project: your-gcp-project-id
    topics: "topic1,topic2"
```

## Configuration

The plugin requires the following configuration.

#### Configuration Keys:

* **project:** The Google Cloud Project ID.
* **topics:** A comma-separated string of Pub/Sub topic IDs.