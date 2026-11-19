# TgWsProxy SNI for Android

[Русская документация](README.md) · [Latest APK](https://github.com/malordin/tg-ws-proxy-android/releases/latest) · [GitHub Actions](https://github.com/malordin/tg-ws-proxy-android/actions)

This public fork of Regstar2/tg-ws-proxy-android separates Cloudflare Proxy TLS
SNI (the base domain) from HTTP Host (`kwsN.<domain>`), addressing the
hostname-dependent TLS timeouts observed on mobile data on November 16, 2026.
MTProto, SOCKS5, and Cloudflare diagnostics use the same fix.

**TgWsProxy SNI** installs separately as `io.github.malordin.tgwsproxy`. The default
address is **127.0.0.1:1444**. Opening this version migrates the previous fork's
default 1443 to 1444, preserving other ports. Reapply the proxy link after updating.

**Apply in Telegram** always opens the Android client chooser, as in amurcanov's
user flow. Android lists all handlers of the implicit `tg://` intent, including
Telegram Beta and Plus, without the Android 10+ limit on EXTRA_INITIAL_INTENTS.

Pushes to main and pull requests run tests and produce a debug APK artifact.
Pushing a version tag such as `v1.13.1` runs the signed release workflow and
publishes the APK with its SHA-256 checksum. Signing credentials are stored in
GitHub Actions Secrets, never in Git. Pull request builds cannot access them.

See [build instructions](docs/fork-sni.md), [full documentation](README.md), and
[GPL-3.0 license](LICENSE). Based on
[Regstar2](https://github.com/Regstar2/tg-ws-proxy-android), with the client chooser
approach from [amurcanov](https://github.com/amurcanov/tg-ws-proxy-android) and
upstream proxy work by [Flowseal](https://github.com/Flowseal/tg-ws-proxy).

## November 19 mobile-network workaround

Version 1.13.1 defaults to plain WebSocket on port 80 and provides **Settings → Cloudflare → Cloudflare Proxy without TLS (port 80)**.
Stop the proxy before changing this setting, then start it again. This explicitly
uses plain WebSocket to CF Proxy; Telegram MTProto encryption remains, but the
extra TLS layer is absent. Plain WebSocket is enabled by default; TLS can be explicitly selected.
The local listener remains on port 1444. MTProto, SOCKS5, and CF diagnostics honor
the same setting; Direct and Worker transports are unchanged.

Independent mobile-network probes verified Telegram res_pq nonce responses via
port 80 for multiple domains and DC2/DC4, while TLS to the same proxy domains
was timing out. Future network filtering may require a different route.
