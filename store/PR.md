Add Headroom Compression & Stats

Repository: https://github.com/frankyw/cliproxyapi-headroom
Release: v0.4.0

The plugin compresses eligible tool-result text through Headroom's marker-free compression endpoint before CLIProxyAPI provider execution. It adds a plugin menu page with persistent CPA-specific savings, latency and history, plus a separate view of Headroom's liveness, readiness, health, statistics and durable history endpoints. CPA Manager Plus and the CPA-hosted panel require no modifications. Dashboard data is public to anyone who can reach the manager address; it contains aggregate statistics and recent request metadata, never prompt or tool content.

Release assets required before submission:
- headroom_0.4.0_linux_amd64.zip, containing only headroom.so at its root
- checksums.txt with a SHA-256 entry for that ZIP

Requires a separately running Headroom service. Tested with CLIProxyAPI v7.3.4 and Headroom v0.37.0 on Linux amd64. Other architectures are not included in this initial release.

Validation: Go race tests, protocol-preservation and failure tests, public dashboard data and authenticated management APIs, persistent counters across restart, real Headroom compression and streaming/non-streaming host integration.

The upstream change should modify only registry.json by appending store/registry-entry.json from the plugin repository. Do not submit until the release assets are published and verified.
