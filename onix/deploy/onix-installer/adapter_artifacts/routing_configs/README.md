# Defining Beckn-ONIX Routing Rules

This document explains how to configure the routing rules for the Beckn-ONIX adapter. Routing determines the destination for every message processed by the adapter.

## 1. How Routing Works

The routing mechanism in Beckn-ONIX is a **deterministic lookup process**, not a dynamic, content-based system. When a message is processed, the router identifies the correct destination by matching a pre-defined rule against three key fields from the request:

1.  **`context.domain`**: The Beckn domain of the transaction (e.g., `ONDC:RET10`).
2.  **`context.version`**: The Beckn version of the transaction (e.g., `1.2.0`).
3.  **`context.action or the endpoint`**: The final segment of the request URL path, which represents the Beckn action (e.g., `search`, `on_select`).

If a rule matching this unique combination is found, the router provides the destination details. If no rule matches, the transaction will fail.

---

## 2. Configuration by Module

A critical concept is that the adapter is composed of different transaction modules, and **each module has its own, separate routing configuration file.** This allows for precise control over the message flow for each role (BAP/BPP) and direction (sending/receiving).

### Module Summary

| Module | Role | Purpose | Configuration File |
| :--- | :--- | :--- | :--- |
| `bapTxnCaller` | BAP | **Sending outgoing requests** (e.g., `search`, `init`). | `bapTxnCaller-routing.yaml` |
| `bapTxnReceiver` | BAP | **Receiving incoming callbacks** (e.g., `on_search`). | `bapTxnReceiver-routing.yaml` |
| `bppTxnCaller` | BPP | **Sending outgoing responses/callbacks** (e.g., `on_init`). | `bppTxnCaller-routing.yaml` |
| `bppTxnReceiver` | BPP | **Receiving incoming requests** (e.g., `search`). | `bppTxnReceiver-routing.yaml` |

---

## 3. Routing Rule Schema

All routing files share the same schema. The file must contain a top-level `routingRules` key, which holds a list of individual rules.

```yaml
routingRules:
  # This is a single routing rule
  - domain: "BECKN-DOMAIN"
    version: "BECKN-VERSION"
    targetType: "bpp" | "bap" | "url" | "publisher"
    # The 'target' object is optional but often necessary
    target:
      # Used as the primary for 'url' or as a fallback for 'bpp'/'bap'
      url: https://default-destination.example.com/beckn

      # Used only for 'publisher' targetType
      publisherId: id-of-my-publisher-plugin

    endpoints:
      - "endpoint1"
      - "endpoint2"

```

### Understanding `targetType`

This is the most important field, as it dictates the routing logic.

| `targetType` | Behavior |
| :--- | :--- |
| **`bpp`** | **Protocol-Aware Routing for BAPs.** The router inspects the request's `context` for a `bpp_uri`. <br/>- If `bpp_uri` exists, the message is sent there. <br/>- If `bpp_uri` is **missing**, the router falls back to the URL defined in `target.url`. If `target.url` is also not provided, the routing will fail. |
| **`bap`** | **Protocol-Aware Routing for BPPs.** The router inspects the request's `context` for a `bap_uri`. <br/>- If `bap_uri` exists, the message is sent there. <br/>- If `bap_uri` is **missing**, the router falls back to the URL defined in `target.url`. If `target.url` is also not provided, the routing will fail. |
| **`url`** | **Direct URL Routing.** The router sends the message to the hardcoded URL defined in `target.url`. The request endpoint is appended to the URL's path by default (e.g., `https://host/path` + `search` becomes `https://host/path/search`). |
| **`publisher`**| **Internal Message Forwarding.** The router does not send an HTTP request. Instead, it hands the message to an internal publisher plugin (identified by `target.publisherId`) to be sent to a message queue like RabbitMQ or Kafka. |

---

## 4. Module Configuration Examples

### BAP: Sending Requests (`bapTxnCaller-routing.yaml`)

A BAP's `TxnCaller` sends requests into the network. `search` requests typically go to a Gateway, while subsequent requests go directly to a BPP.

**Goal**: Route `search` to the Gateway, and route `select` based on the `bpp_uri` from the context.

```yaml
routingRules:
  # Rule for initial discovery via Gateway
  - domain: example-domain
    version: 1.0.0
    targetType: url
    target:
      url: https://network-gateway.example.com
    endpoints:
      - search


  # Rule for subsequent, direct-to-BPP calls
  - domain: example-domain
    version: 1.0.0
    targetType: bpp
    endpoints:
      - select
      - init
      - confirm
    # No target.url is needed here, as the bpp_uri from the on_search response
    # MUST be present in the context for these calls.
```

### BPP: Sending Responses (`bppTxnCaller-routing.yaml`)

A BPP's `TxnCaller` sends responses/callbacks back to the BAP that made the original request.

**Goal**: Route all `on_*` responses back to the originating BAP.

```yaml
routingRules:
  - domain: example-domain
    version: 1.0.0
    targetType: bap
    endpoints:
      - on_search
      - on_select
      - on_init
    # No 'target.url' is needed. The router will get the destination
    # from the context.bap_uri, which is required by the Beckn spec.
```

### BAP: Receiving Callbacks (`bapTxnReceiver-routing.yaml`)

A BAP's `TxnReceiver` handles callbacks from BPPs.

**Goal**: Forward all incoming `on_*` callbacks to the BAP's application.

```yaml
routingRules:
  - domain: example-domain
    version: 1.0.0
    targetType: url
    target:
      url: http://my-bap-backend-service:9000/
    endpoints:
      - on_search
      - on_select
      - on_init
      - on_confirm

```

### BPP: Receiving Requests (`bppTxnReceiver-routing.yaml`)

A BPP's `TxnReceiver` is the entry point for requests from the network for BPP. It typically forwards them to the BPP's application/service or message queue.

**Goal**: Forward all requests to the BPP's service. You may send it to a message queue for asynchronous processing as per need.

```yaml
routingRules:
  # Rule for all requests to be processed synchronously
  - domain: example-domain
    version: 1.0.0
    targetType: url
    target:
      url: http://my-bpp-backend-service:8000/api/beckn
    endpoints:
      - search
      - select
      - init


  # Rule for async processing
  - domain: example-domain
    version: 1.0.0
    targetType: publisher
    target:
      publisherId: onix-adapter-topic
    endpoints:
      - confirm

```
