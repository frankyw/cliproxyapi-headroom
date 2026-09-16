# Headroom for CLIProxyAPI

Native CLIProxyAPI plugin that compresses tool-result text through Headroom and adds an embedded statistics page to CPA Manager Plus and compatible CPA-hosted panels. No manager modifications are required.

## Features

- OpenAI Chat, Responses/Codex, Anthropic and Gemini tool-output compression.
- Preserves instructions, user messages, tool IDs, images and provider routing.
- Marker-free Headroom compression with original-request fallback on failure.
- Persistent CPA-only savings, latency, model breakdowns and 90-day hourly history.
- Separate Headroom service view reading `/livez`, `/readyz`, `/health`, `/stats` and `/stats-history`.
- Authenticated statistics APIs, live refresh, responsive charts and JSON export.

## Request flow

```text
Client → CLIProxyAPI → Headroom /v1/compress → CLIProxyAPI → selected provider
```

Headroom returns compressed content; it does not forward the generation request. CLIProxyAPI retains authentication, OAuth, routing, translation and streaming.

## Requirements

Tested with CLIProxyAPI v7.3.4 and Headroom v0.37.0 on Linux amd64. The initial release supports Linux amd64 with Debian Bookworm-compatible glibc. Headroom must run separately and support `/v1/compress` with `config.mode: lossy_inline`. The earlier compression-only plugin was also exercised with the existing deployment originally labelled 0.33.0; the live service now reports 0.37.0.

## Install

Download `headroom_0.3.0_linux_amd64.zip` and `checksums.txt` from [Releases](https://github.com/frankyw/cliproxyapi-headroom/releases). Verify the checksum, extract `headroom.so`, and install it as `plugins/linux/amd64/headroom-v0.3.0.so` in CLIProxyAPI's persistent plugin directory. Back up your config and retain other plugin settings when merging:

```yaml
plugins:
  enabled: true
  configs:
    headroom:
      enabled: true
      priority: 10
      endpoint: http://headroom:8787/v1/compress
      stats_path: plugins/data/headroom/stats.json
      timeout_ms: 10000
      min_chars: 512
      target_ratio: 0.5
      token_env: ""
      service_url: ""
```

Restart CLIProxyAPI, then refresh the manager and open **Headroom Stats** in its sidebar. Both Docker containers must share a network. Headroom requires `HEADROOM_COMPRESS_ALLOW_REMOTE=1` for compression requests from another container. If authentication is configured, set `token_env` to the name of a Headroom-token environment variable available to CLIProxyAPI; client and provider credentials are never forwarded.

`service_url` optionally overrides the service root used for health/statistics. Empty uses the origin of `endpoint`. Use it if Headroom is hosted under a URL prefix. Keep these operator-configured endpoints on trusted infrastructure.

## Dashboard authentication

The static page is public, but all statistics endpoints require the existing management credential. On the same origin, the page can reuse a remembered CPA Manager session. If the manager did not remember its credential, the page asks for the same manager key once; it keeps that key only in page memory. In Plus mode use the Plus admin key; on the CPA-hosted panel use the CPA management key.

- Page: `/v0/resource/plugins/headroom/stats`
- CPA-only data: `/v0/management/plugins/headroom/stats`
- Headroom service data: `/v0/management/plugins/headroom/service-stats`

Custom reverse proxies must pass both `/v0/resource/plugins/*` and `/v0/management/plugins/*` to the corresponding manager/CPA service.

## Statistics and interpretation

CPA-only counters begin when v0.2+ is enabled. They record tool-output reductions accepted by this plugin before provider execution, not successful/billed provider requests. Token estimates are recorded only when Headroom's returned text is fully accepted and its token counters are valid. Partially accepted reductions retain byte metrics without claiming token savings. Requests without eligible text, unchanged responses and fallback failures are counted separately.

Snapshots are written atomically every two seconds and on orderly plugin shutdown. Abrupt termination may lose the last two seconds. `stats_path` must reside on persistent storage; empty means memory-only. The file stores aggregate counts, bounded model labels, 90 days of hourly history and the latest 100 metadata-only events. No prompts, tool contents, request headers or API keys are stored. Lifetime totals remain after hourly retention expires. A corrupt file is not silently overwritten; configuration fails so the operator can restore it.

The service section separately reads all five Headroom endpoints in parallel, with a five-second deadline and ten-second cache. It displays health, aggregate request/token statistics, display-session/lifetime savings and durable history. These totals include other Headroom clients and older traffic; they are never added to the CPA-only counters. Endpoint details are filtered to omit configuration, internal URLs and project labels. They are not a verbatim dump of every field. The service view includes up to 720 hourly and 90 daily buckets, with its chart showing the latest 30 daily buckets.

## Compression scope and limitations

Eligible text includes Chat `tool`/legacy `function` content, Anthropic `tool_result` text, Responses `function_call_output`, and string leaves within Gemini `functionResponse.response` objects. Outputs shorter than `min_chars` bytes pass through. Gemini numeric fields and strings directly inside arrays are not compressed.

The plugin uses marker-free `lossy_inline` mode and rejects CCR hashes. It does not provide retrieval tools, contextual cross-message optimization or provider prefix-cache tracking. Compression is lossy; savings and answer quality depend on workload. The target ratio is not guaranteed. WebSocket behavior depends on the host invoking its interception hook and has not been independently tested.

Compression errors/timeouts preserve the original request. Bodies over 32 MiB bypass compression. Pending compression calls may continue until their deadline after a client disconnects because the native RPC interface does not expose the host request context. Metrics are not a billing ledger.

## Build and verify

Requires Docker and Python 3:

```sh
sh scripts/build.sh
HEADROOM_TEST_URL=http://headroom:8787/v1/compress sh scripts/test-live.sh
```

The build uses Go 1.26 with a C compiler and `-buildmode=c-shared`, not Go's compiler-specific plugin format. Override release metadata with `REPOSITORY_URL` and `PLUGIN_AUTHOR`; GitHub Actions populates them automatically.

On Linux with Docker, Python 3 and PyYAML, `python3 tests/integration.py` runs an isolated CLIProxyAPI container and mock provider with Headroom at `127.0.0.1:8787`, ports 18318/18319, and no production credentials. It verifies compression, streaming, authenticated stats, page registration, all five service endpoints and restart persistence. Work files remain in ignored `work/`.

## Releases and plugin store

Push a tag matching `plugin.go`, currently `v0.3.0`. GitHub Actions runs race tests and produces:

- `headroom_0.3.0_linux_amd64.zip`, containing `headroom.so` at its root
- `checksums.txt`, in SHA-256 format

See `store/registry-entry.json` and `store/PR.md` for the prepared official store entry. Store inclusion requires an upstream PR; publishing this repository does not itself list the plugin.

## Disable / rollback

Set `plugins.configs.headroom.enabled: false` and restart CPA. For a version rollback, stop CPA, remove the newer versioned library, restore the previous library/config, then start CPA. Never overwrite a loaded native library. Retain the statistics file if you want to preserve history.

MIT licensed. C ABI glue is adapted from CLIProxyAPI's MIT example; the original notice is retained in LICENSE.
