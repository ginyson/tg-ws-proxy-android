# TgWsProxy SNI fork

This fork is based on Regstar2/tg-ws-proxy-android 1.11.8. The application is
installed separately as **TgWsProxy SNI** (`io.github.malordin.tgwsproxy`). Its
update feed points to this fork's GitHub Releases.

## Cloudflare route

Cloudflare proxy connections use the configured base domain as TLS SNI while
retaining `kwsN.<base-domain>` as the HTTP Host and `/apiws` as the path. Both
MTProto and SOCKS5 use this behavior; the route diagnostics use the same names.
Direct Telegram WebSocket and Worker routes retain their existing TLS names.

This addresses the hostname-dependent TLS timeouts observed on mobile data on
2026-11-16. The change is automatic: do not enter the base domain into the
frontend's Fake TLS masking-domain field.

Initial independent device probes succeeded with SNI `cakeisalie.co.uk` and HTTP
Host `kws2.cakeisalie.co.uk` / `kws4.cakeisalie.co.uk`. Probes also succeeded for
`pclead.co.uk` and `stopblocking.co.uk`. HTTP 101 alone does not establish that
Telegram messages and media work; test the installed APK on the affected network.

## Build on Linux x86-64

Requirements: JDK 17 or 21, Go 1.25+, Android SDK platform 35, build tools 34.0.0,
and Android NDK 27.2.12479018.

```sh
export ANDROID_HOME="$HOME/Android/Sdk"
export ANDROID_SDK_ROOT="$ANDROID_HOME"
sdkmanager 'platforms;android-35' 'build-tools;34.0.0' 'ndk;27.2.12479018'
(cd native/tgwsproxy && go test ./...)
./gradlew :app:testDebugUnitTest :app:assembleDebug
```

The debug APK is `app/build/outputs/apk/debug/app-debug.apk`. For a release APK,
set `KEYSTORE_FILE`, `KEYSTORE_PASSWORD`, `KEY_PASSWORD`, and `KEY_ALIAS` to your
own signing credentials and run `./gradlew :app:assembleRelease`. Keep the same
private keystore for subsequent updates; do not commit it or its passwords.

## First run

1. Stop the original TgWsProxy before starting this fork: both default to port
   1443. Alternatively, assign different ports.
2. Start TgWsProxy SNI and use its **Apply in Telegram** action to configure the
   desired Telegram client. A fresh installation has a different MTProto secret.
3. On the affected mobile network, check that Telegram connects and loads both
   messages and media. A running local listener alone does not prove connectivity.
