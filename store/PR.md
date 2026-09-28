Add Headroom Compression & Stats

Repository: https://github.com/frankyw/cliproxyapi-headroom
Release: v0.6.0

The plugin compresses eligible user-message and tool-result text through Headroom before CLIProxyAPI provider execution. User-message compression is configurable and defaults to on. A standalone CPA-hosted dashboard displays persistent CPA-specific savings, latency and history, plus filtered health and statistics from all five Headroom service endpoints.

Only static HTML is served under `/v0/resource/plugins/headroom/stats`. Dynamic CPA and Headroom service data use authenticated `/v0/management/plugins/headroom/stats` and `/v0/management/plugins/headroom/service-stats` routes. The standalone page reuses the manager credential saved on the same origin in `cli-proxy-auth`, including CPAMP server-mode **Remember credential** and direct CPA Manager **Remember password**. It supports the manager’s v1 and v2 storage formats; when no key was saved, it accepts the existing credential for the tab. The menu embeds it in the manager as a same-origin iframe. No manager changes are required.

Release assets: five `headroom_0.6.0_<goos>_<goarch>.zip` archives for Linux amd64/arm64, macOS amd64/arm64, and Windows amd64, each containing only its platform library at the ZIP root; one `checksums.txt` with SHA-256 entries for every archive.

Requires a separately running Headroom service. Linux amd64 was exercised end to end against CLIProxyAPI v7.3.17 and Headroom v0.37.0; the other four platform builds are validated by native GitHub Actions runners.

Validation: Go race tests, protocol-preservation and failure tests, rejected public dynamic routes, authenticated management APIs, persistent counters across restart, real Headroom compression and streaming/non-streaming host integration.

The upstream change should modify only registry.json by appending store/registry-entry.json after release assets are published and verified.
