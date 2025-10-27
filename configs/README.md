# Onix Configuration README

This document provides a detailed explanation of the configuration values for all Onix services.

## Table of Contents

- [Registry Service (`registry.yaml`)](#registry-service-registryyaml)
- [Gateway Service (`gateway.yaml`)](#gateway-service-gatewayyaml)
- [Subscriber Service (`subscriber.yaml`)](#subscriber-service-subscriberyaml)
- [Registry Admin Service (`registry-admin.yaml`)](#registry-admin-service-registry-adminyaml)
- [Beckn Adapter (`adapter.yaml` and routing files)](#beckn-adapter-adapteryaml-and-routing-files)

---

## Registry Service (`registry.yaml`)

The `registry` service is responsible for managing subscriptions and looking up network participants.

**log**: This section configures the logging for the application.

| Key      | Type   | Description                                                                      |
| :------- | :----- | :--------------------------------------------------------------------------------------------------------------------------------------- |
| `level`  | String | The logging level. Possible values: `FATAL`, `ERROR`, `WARN`, `INFO`, `DEBUG`, `OFF`. |
| `target` | String | Where to write the logs. Possible values: `STDOUT` (for containers) or `FILE` (writes to `app.log`).                                   |

Code Reference: `internal/log/log.go`

**timeouts**: Configures the HTTP server's timeouts to manage client connections and graceful shutdown.

| Key        | Type     | Description                                                                          |
| :--------- | :------- | :----------------------------------------------------------------------------------- |
| `read`     | Duration | The maximum duration for reading the entire request, including the body. This prevents slow clients from holding connections open. |
| `write`    | Duration | The maximum duration before timing out writes of the response. This is useful for ensuring responses are sent promptly. |
| `idle`     | Duration | The maximum amount of time to wait for the next request when keep-alives are enabled.|
| `shutdown` | Duration | The duration to wait for graceful server shutdown.                                   |

Code Reference: `cmd/registry/main.go`

**server**: This section configures the HTTP server.

| Key    | Type   | Description                                                              |
| :----- | :----- | :-------------------------------------- |
| `host` | String | The host on which the server will listen. `0.0.0.0` listens on all available interfaces. |
| `port` | Int    | The port on which the server will listen (e.g., `8080`).                 |

Code Reference: `cmd/registry/main.go`

**db**: This section configures the database connection.

| Key               | Type     | Description                                                                                   |
| :---------------- | :------- | :-------------------------------------------------------------------------------------------- |
| `user`            | String   | The service account email used to authenticate with the Cloud SQL database.                   |
| `name`            | String   | The name of the database to connect to.                                                       |
| `connectionName`  | String   | The Cloud SQL instance connection name in the format `<PROJECT_ID:REGION:INSTANCE_ID>`.       |
| `maxOpenConns`    | Int      | The maximum number of open connections to the database. `0` means no limit.                   |
| `maxIdleConns`    | Int      | The maximum number of connections in the idle connection pool. `0` means no idle connections. |
| `connMaxIdleTime` | Duration | The maximum amount of time a connection may be idle before being closed. `0` means no limit.  |
| `connMaxLifetime` | Duration | The maximum amount of time a connection may be reused before being closed. `0` means no limit.|

Code Reference: `internal/repository/registry.go`

**event**: This section configures the event publisher.

| Key         | Type   | Description                                           |
| :---------- | :----- | :---------------------------------------------------- |
| `projectID` | String | The Google Cloud project ID for Pub/Sub.              |
| `topicID`   | String | The Pub/Sub topic ID to publish events to.            |

Code Reference: `internal/event/publisher.go`

---

## Gateway Service (`gateway.yaml`)

The `gateway` service acts as a proxy between network participants.

**log**: This section configures the logging for the application.

| Key      | Type   | Description                                                                      |
| :------- | :----- | :------------------------------------------------------------------------------- |
| `level`  | String | The logging level. Can be one of `FATAL`, `ERROR`, `WARN`, `INFO`, `DEBUG`, `OFF`. |
| `target` | String | Where to write the logs. Can be `STDOUT` or `FILE`. If `FILE`, it writes to `app.log`. |

Code Reference: `internal/log/log.go`

**timeouts**: Configures the HTTP server's timeouts to manage client connections and graceful shutdown.

| Key        | Type     | Description                                                                          |
| :--------- | :------- | :----------------------------------------------------------------------------------- |
| `read`     | Duration | The maximum duration for reading the entire request, including the body. This prevents slow clients from holding connections open. |
| `write`    | Duration | The maximum duration before timing out writes of the response. This is useful for ensuring responses are sent promptly. |
| `idle`     | Duration | The maximum amount of time to wait for the next request when keep-alives are enabled.|
| `shutdown` | Duration | The duration to wait for graceful server shutdown.                                   |

Code Reference: `cmd/gateway/main.go`

**server**: This section configures the HTTP server.

| Key    | Type   | Description                                                              |
| :----- | :----- | :-------------------------------------- |
| `host` | String | The host on which the server will listen. `0.0.0.0` listens on all available interfaces. |
| `port` | Int    | The port on which the server will listen (e.g., `8080`).                 |

Code Reference: `cmd/gateway/main.go`

**projectID**: The Google Cloud project ID.

| Key         | Type   | Description                      |
| :---------- | :----- | :------------------------------- |
| `projectID` | String | The Google Cloud project ID.     |

**registry**: This section configures the client for the registry service.

| Key                 | Type     | Description                                                      |
| :------------------ | :------- | :--------------------------------------------------------------- |
| `baseURL`           | String   | The base URL of the registry service (e.g., `http://registry:8080`). |
| `timeout`           | Duration | The timeout for each individual HTTP request attempt.            |
| `maxIdleConns`      | Int      | The maximum number of idle connections in the pool.              |
| `maxIdleConnsPerHost` | Int      | The maximum number of idle connections to keep per host.         |
| `maxConnsPerHost`   | Int      | The maximum number of connections per host. `0` means no limit.  |
| `idleConnTimeout`   | Duration | The maximum amount of time an idle connection will wait before being closed. |

Code Reference: `internal/client/registry.go`

**redisAddr**: The address of the Redis server for caching.

| Key         | Type   | Description                               |
| :---------- | :----- | :---------------------------------------- |
| `redisAddr` | String | The address of the Redis server for caching. |

**keyManagerCacheTTL**: This section configures the TTL for the key manager cache.

| Key                  | Type | Description                                                                                                                  |
| :------------------- | :--- | :--------------------------------------------------------------------------------------------------------------------------- |
| `privateKeysSeconds` | Int  | The Time-To-Live (TTL) in seconds for cached private keys. After this duration, the key will be fetched again from the source. |
| `publicKeysSeconds`  | Int  | The Time-To-Live (TTL) in seconds for cached public keys. After this duration, the key will be fetched again from the source.  |

Code Reference: `plugins/inmemorysecretkeymanager/inmemorysecretkeymanager.go`

**maxConcurrentFanoutTasks**: The maximum number of concurrent fanout tasks.

| Key                        | Type | Description                               |
| :------------------------- | :--- | :---------------------------------------- |
| `maxConcurrentFanoutTasks` | Int  | The maximum number of concurrent fanout tasks. |

**taskQueueWorkersCount**: The number of workers for the channel task queue.

| Key                     | Type | Description                                                                                             |
| :---------------------- | :--- | :------------------------------------------------------------------------------------------------------ |
| `taskQueueWorkersCount` | Int  | The number of worker goroutines that process tasks from the internal channel-based queue.                 |

**taskQueueBufferSize**: The buffer size of the channel task queue.

| Key                   | Type | Description                                                                                             |
| :-------------------- | :--- | :------------------------------------------------------------------------------------------------------ |
| `taskQueueBufferSize` | Int  | The buffer size of the internal channel task queue. A larger size can handle more burst traffic.          |

**subscriberID**: The subscriber ID of the gateway.

| Key            | Type   | Description                                                                                             |
| :------------- | :----- | :------------------------------------------------------------------------------------------------------ |
| `subscriberID` | String | The unique identifier for the gateway itself, as it is registered in the Beckn network.                   |

**httpClientRetry**: This section configures the retryable HTTP client.

| Key                 | Type     | Description                                       |
| :------------------ | :------- | :------------------------------------------------ |
| `retryMax`          | Int      | The maximum number of retries for a failed request. `0` means no retries. |
| `waitMin`           | Duration | The minimum time to wait before the first retry.  |
| `waitMax`           | Duration | The maximum time to wait before a retry.          |
| `timeout`           | Duration | The timeout for each individual HTTP request attempt. |
| `maxIdleConns`      | Int      | The maximum number of idle connections in the pool. |
| `maxIdleConnsPerHost` | Int      | The maximum number of idle connections to keep per host. |
| `maxConnsPerHost`   | Int      | The maximum number of connections per host. `0` means no limit. |
| `idleConnTimeout`   | Duration | The maximum amount of time an idle connection will wait before being closed. |

Code Reference: `internal/service/proxy.go`

---

## Subscriber Service (`subscriber.yaml`)

The `subscriber` service is a sample implementation of a network participant.

**log**: This section configures the logging for the application.

| Key      | Type   | Description                                                                      |
| :------- | :----- | :------------------------------------------------------------------------------- |
| `level`  | String | The logging level. Can be one of `FATAL`, `ERROR`, `WARN`, `INFO`, `DEBUG`, `OFF`. |
| `target` | String | Where to write the logs. Can be `STDOUT` or `FILE`. If `FILE`, it writes to `app.log`. |

Code Reference: `internal/log/log.go`

**timeouts**: Configures the HTTP server's timeouts to manage client connections and graceful shutdown.

| Key        | Type     | Description                                                                          |
| :--------- | :------- | :----------------------------------------------------------------------------------- |
| `read`     | Duration | The maximum duration for reading the entire request, including the body. This prevents slow clients from holding connections open. |
| `write`    | Duration | The maximum duration before timing out writes of the response. This is useful for ensuring responses are sent promptly. |
| `idle`     | Duration | The maximum amount of time to wait for the next request when keep-alives are enabled.|
| `shutdown` | Duration | The duration to wait for graceful server shutdown.                                   |

Code Reference: `cmd/subscriber/main.go`

**server**: This section configures the HTTP server.

| Key    | Type   | Description                                                              |
| :----- | :----- | :-------------------------------------- |
| `host` | String | The host on which the server will listen. `0.0.0.0` listens on all available interfaces. |
| `port` | Int    | The port on which the server will listen (e.g., `8080`).                 |

Code Reference: `cmd/subscriber/main.go`

**projectID**: The Google Cloud project ID.

| Key         | Type   | Description                      |
| :---------- | :----- | :------------------------------- |
| `projectID` | String | The Google Cloud project ID.     |

**registry**: This section configures the client for the registry service.

| Key                 | Type     | Description                                                      |
| :------------------ | :------- | :--------------------------------------------------------------- |
| `baseURL`           | String   | The base URL of the registry service (e.g., `http://registry:8080`). |
| `timeout`           | Duration | The timeout for each individual HTTP request attempt.            |
| `maxIdleConns`      | Int      | The maximum number of idle connections in the pool.              |
| `maxIdleConnsPerHost` | Int      | The maximum number of idle connections to keep per host.         |
| `maxConnsPerHost`   | Int      | The maximum number of connections per host. `0` means no limit.  |
| `idleConnTimeout`   | Duration | The maximum amount of time an idle connection will wait before being closed. |


Code Reference: `internal/client/registry.go`

**redisAddr**: The address of the Redis server for caching.

| Key         | Type   | Description                               |
| :---------- | :----- | :---------------------------------------- |
| `redisAddr` | String | The address of the Redis server for caching. |

**regID**: The registry's ID.

| Key     | Type   | Description        |
| :------ | :----- | :----------------- |
| `regID` | String | The registry's ID. |

**keyManagerCacheTTL**: This section configures the TTL for the key manager cache.

| Key                  | Type | Description                           |
| :------------------- | :--- | :------------------------------------ |
| `privateKeysSeconds` | Int  |  The Time-To-Live (TTL) in seconds for cached private keys. After this duration, the key will be fetched again from the source.  |
| `publicKeysSeconds`  | Int  | The Time-To-Live (TTL) in seconds for cached public keys. After this duration, the key will be fetched again from the source.   |

Code Reference: `plugins/inmemorysecretkeymanager/inmemorysecretkeymanager.go`

**regKeyID**: The registry's key ID.

| Key        | Type   | Description                               |
| :--------- | :----- | :---------------------------------------- |
| `regKeyID` | String | The registry's key ID. |

**event**: This section configures the event publisher.

| Key         | Type   | Description                                           |
| :---------- | :----- | :---------------------------------------------------- |
| `projectID` | String | The Google Cloud project ID for Pub/Sub.              |
| `topicID`   | String | The Pub/Sub topic ID to publish events to.            |

Code Reference: `internal/event/publisher.go`

---

## Registry Admin Service (`registry-admin.yaml`)

The `registry-admin` service provides administrative functions for the registry.

**log**: This section configures the logging for the application.

| Key      | Type   | Description                                                                      |
| :------- | :----- | :------------------------------------------------------------------------------- |
| `level`  | String | The logging level. Can be one of `FATAL`, `ERROR`, `WARN`, `INFO`, `DEBUG`, `OFF`. |
| `target` | String | Where to write the logs. Can be `STDOUT` or `FILE`. If `FILE`, it writes to `app.log`. |

Code Reference: `internal/log/log.go`

**timeouts**: Configures the HTTP server's timeouts to manage client connections and graceful shutdown.

| Key        | Type     | Description                                                                          |
| :--------- | :------- | :----------------------------------------------------------------------------------- |
| `read`     | Duration | The maximum duration for reading the entire request, including the body. This prevents slow clients from holding connections open. |
| `write`    | Duration | The maximum duration before timing out writes of the response. This is useful for ensuring responses are sent promptly. |
| `idle`     | Duration | The maximum amount of time to wait for the next request when keep-alives are enabled.|
| `shutdown` | Duration | The duration to wait for graceful server shutdown.                                   |

Code Reference: `cmd/admin/main.go`

**server**: This section configures the HTTP server.

| Key    | Type   | Description                                                              |
| :----- | :----- | :-------------------------------------- |
| `host` | String | The host on which the server will listen. `0.0.0.0` listens on all available interfaces. |
| `port` | Int    | The port on which the server will listen (e.g., `8080`).                 |

Code Reference: `cmd/admin/main.go`

**db**: This section configures the database connection.

| Key               | Type     | Description                                                                                   |
| :---------------- | :------- | :-------------------------------------------------------------------------------------------- |
| `user`            | String   | The service account email used to authenticate with the Cloud SQL database.                   |
| `name`            | String   | The name of the database to connect to.                                                       |
| `connectionName`  | String   | The Cloud SQL instance connection name in the format `<PROJECT_ID:REGION:INSTANCE_ID>`.       |
| `maxOpenConns`    | Int      | The maximum number of open connections to the database. `0` means no limit.                   |
| `maxIdleConns`    | Int      | The maximum number of connections in the idle connection pool. `0` means no idle connections. |
| `connMaxIdleTime` | Duration | The maximum amount of time a connection may be idle before being closed. `0` means no limit.  |
| `connMaxLifetime` | Duration | The maximum amount of time a connection may be reused before being closed. `0` means no limit.|

Code Reference: `internal/repository/registry.go`

**npClient**: This section configures the client for Network Participants.

| Key       | Type     | Description                                     |
| :-------- | :------- | :---------------------------------------------- |
| `timeout` | Duration | The timeout for each individual HTTP request attempt. |

Code Reference: `internal/client/np.go`

**admin**: This section configures the admin service.

| Key                 | Type | Description                               |
| :------------------ | :--- | :---------------------------------------- |
| `operationRetryMax` | Int  | The maximum number of retries for an operation. |

Code Reference: `internal/service/admin.go`

**event**: This section configures the event publisher.

| Key         | Type   | Description                                           |
| :---------- | :----- | :---------------------------------------------------- |
| `projectID` | String | The Google Cloud project ID for Pub/Sub.              |
| `topicID`   | String | The Pub/Sub topic ID to publish events to.            |

Code Reference: `internal/event/publisher.go`

**setup**: This section configures the registry's self-registration.

| Key            | Type   | Description                                                                       |
| :------------- | :----- | :-------------------------------------------------------------------------------- |
| `keyID`        | String | The unique key ID for the registry's own keyset, used for signing and encryption. |
| `subscriberID` | String | The unique identifier for the registry itself within the Beckn network.             |
| `url`          | String | The base URL of the registry service.                         |
| `domain`       | String | The domain the registry belongs to (e.g., `beckn_network`).                       |

Code Reference: `internal/service/setup.go`

---

## Beckn Adapter (`adapter.yaml` and routing files)

The Beckn adapter is a highly configurable, plugin-based component that facilitates communication between Beckn Application Platforms (BAPs) and Beckn Provider Platforms (BPPs). Its behavior is defined by a main YAML file (`adapter.yaml`) and associated routing files.

### `adapter.yaml`: Main Configuration

This file defines the core settings, modules, and processing pipelines for the adapter.

#### Top-Level Configuration

| Key | Type | Description |
| :--- | :--- | :--- |
| `appName` | String | The name of the application (e.g., `onix`). |
| `log` | Object | Configures logging for the adapter. See log section below. |
| `http` | Object | Configures the HTTP server for the adapter. See http section below. |
| `pluginManager` | Object | Configures the plugin manager. See pluginManager section below. |
| `modules` | Array | A list of modules to be loaded by the adapter. See modules section below. |

---

#### `log` Section

Configures the structured logging for the adapter.

| Key | Type | Description |
| :--- | :--- | :--- |
| `level` | String | The logging level (e.g., `debug`, `info`, `warn`, `error`). |
| `destinations` | Array | A list of logging destinations. Each destination is an object with a `type` (e.g., `stdout`, `file`). |
| `contextKeys` | Array | A list of context keys (e.g., `transaction_id`, `message_id`, `subscriber_id`, `module_id`) to be included in every log entry for better traceability. |

---

#### `http` Section

Configures the adapter's built-in HTTP server.

| Key | Type | Description |
| :--- | :--- | :--- |
| `port` | Int | The port on which the server will listen (e.g., `8080`). |
| `timeout` | Object | Configures server timeouts (`read`, `write`, `idle`) to manage connections effectively. |

---

#### `pluginManager` Section

Configures how the adapter loads plugins.

| Key | Type | Description |
| :--- | :--- | :--- |
| `root` | String | The root directory where compiled plugin (`.so`) files are located (e.g., `./plugins`). |
| `remoteRoot` | String | The remote root for plugins, typically a GCS path for production deployments.  (e.g., `/mnt/gcs/plugins/plugins_bundle.zip`)|

---

#### `modules` Section

This is the core of the adapter configuration, defining the different processing modules. Each module in the list represents a distinct request-handling pipeline.

**Module Structure:**

| Key | Type | Description |
| :--- | :--- | :--- |
| `name` | String | A unique name for the module (e.g., `bapTxnReceiver`, `bapTxnCaller`, `bppTxnReceiver`, `bppTxnCaller`). |
| `path` | String | The HTTP path prefix that this module will handle (e.g., `/bap/receiver/`). |
| `handler` | Object | Defines the processing logic for the module. See handler section below. |

##### `handler` Section

The `handler` section defines the role, plugins, and processing steps for a module.

| Key | Type | Description |
| :--- | :--- | :--- |
| `type` | String | The handler type e.g. `std` |
| `role` | String | The Beckn role of this handler: `bap` (Buyer App) or `bpp` (Provider App). |
| `registryUrl` | String | The URL of the Beckn registry for network lookups. |
| `plugins` | Object | A map of plugin configurations used by this handler. See plugins section below. |
| `steps` | Array | A list of processing steps to be executed. See steps section below. |

##### `plugins` Section

This section defines the specific configuration for each plugin instance used within a handler.

| Plugin | Description | Configuration Keys |
| :--- | :--- | :--- |
| **`keyManager`** | Manages cryptographic keys for signing and validation. | `id`: `<plugin-id>`<br>`config`: `projectID`, `privateKeyCacheTTLSeconds`, `publicKeyCacheTTLSeconds` |
| **`cache`** | Provides caching capabilities (e.g., for responses) to improve performance, typically using Redis. | `id`: `<plugin-id>`<br>`config`: `addr` (Redis address), `password`, `db` |
| **`schemaValidator`** | Validates incoming and outgoing messages against Beckn JSON schemas. | `id`: `<plugin-id>`<br>`config`: `schemaDir` (local directory for schemas) |
| **`signValidator`** | Validates the digital signature of incoming Beckn messages. | `id`: `<plugin-id>`<br> |
| **`signer`** | Applies a digital signature to outgoing Beckn messages. | `id`: `<plugin-id>`<br> |
| **`router`** | Determines where to send a message based on routing rules defined in a separate file. | `id`: `<plugin-id>`<br>`config`: `routingConfig` (path to the routing YAML file) |
| **`publisher`** | Publishes messages to an asynchronous message broker (e.g., RabbitMQ, Pub/Sub) for async processing. | `id`: `<plugin-id>`<br>`config`: `addr`, `username`, `password`, `exchange` (for RabbitMQ) or `projectID`, `topic` (for Pub/Sub) |
| **`middleware`** | A list of plugins that preprocess requests. A common middleware is `reqpreprocessor`. | `id`: `<plugin-id>`<br>`config`: `contextKeys` (e.g., `transaction_id,message_id`), `role` (`bap` or `bpp`) |

##### `steps` Section

The `steps` array defines the pipeline of actions for the handler. Each step corresponds to a plugin's function.

**Common Steps:**

| Step | Description | Associated Plugin |
| :--- | :--- | :--- |
| `validateSign` | Validates the signature of an incoming request. | `signValidator` |
| `addRoute` | Determines the destination for the message using the router plugin. | `router` |
| `validateSchema` | Validates the message body against the relevant Beckn JSON schema. | `schemaValidator` |
| `sign` | Signs an outgoing message before sending it. | `signer` |

**Example Module Configuration:**

Here is an example of a `bapTxnReceiver` module, which receives callbacks at the BAP.

```yaml
  - name: bapTxnReceiver
    path: /bap/receiver/
    handler:
      type: std
      role: bap
      registryUrl: http://localhost:8080/reg # URL for network lookups
      plugins:
        keyManager:
          id: keymanager
          config:
            projectID: beckn-onix-local
            privateKeyCacheTTLSeconds: 15
            publicKeyCacheTTLSeconds: 3600
        cache:
          id: rediscache
          config:
            addr: localhost:6379
        schemaValidator:
          id: schemavalidator
          config:
            schemaDir: ./schemas
        signValidator:
          id: signvalidator
        router:
          id: router
          config:
            routingConfig: ./config/bapTxnReceiver-routing.yaml
        middleware:
          - id: reqpreprocessor
            config:
              contextKeys: transaction_id,message_id
              role: bap
      steps:
        - validateSign
        - addRoute
        - validateSchema
```

---

**Routing Configuration (`*-routing.yaml`)**: These files define the rules for how and where Beckn messages should be sent based on their content.

| Key       | Type   | Description                               |
| :-------- | :----- | :---------------------------------------- |
| `routingRules` | Array | A list of routing rules. |

Each routing rule has the following properties:

| Key | Type | Description |
| :--- | :--- | :--- |
| `domain` | String | The Beckn domain to which the rule applies. |
| `version` | String | The core version of the Beckn protocol (e.g., `1.2.0`). |
| `targetType` | String | The type of the destination. Possible values: `bpp` or `bap` (for network lookup), `url` (for a direct HTTP endpoint), or `publisher` (for an async message queue). |
| `target` | Object | The destination details, which vary based on `targetType`. For `url`, it contains a `url` key. For `publisher`, it contains `topic`. |
| `endpoints` | Array | A list of Beckn actions (e.g., `search`, `select`, `init`) to which this rule applies. |
