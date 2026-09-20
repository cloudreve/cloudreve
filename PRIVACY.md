# Privacy Policy — Cloudreve Mobile

**Last updated: 2026-09-20**

Cloudreve Mobile is an open-source client for self-hosted Cloudreve servers. It is designed so that your data stays between your device and the server you choose.

## What the app does

- Connects to a **Cloudreve server URL that you provide**. All file transfers, authentication, and API calls go directly between your device and that server.
- Stores your server address and session credentials **locally on your device** (Android DataStore / app-private storage).
- Uploads files, photos, or folders **only when you initiate it** (manual upload, share sheet, or a sync/camera-backup feature you explicitly enable).
- Optionally reads media files and folders you select for upload or sync features.

## What the app does not do

- No analytics, no telemetry, no crash reporting to us.
- No advertising, no tracking SDKs, no third-party data sharing.
- No data is sent to the developers of this app. There is no developer-operated backend — your configured server is the only remote endpoint.

## Data safety summary

| Category | Collected | Shared |
|---|---|---|
| Files and photos | Only files you choose to upload, sent to your own server | Never |
| Credentials | Stored on-device only | Never |
| Personal info | None | None |

## Encryption in transit

Traffic is encrypted when your server is configured with HTTPS. If you configure a plain-HTTP server address, traffic is not encrypted — this is under your control as the server operator.

## Data deletion

Uninstalling the app removes all locally stored credentials and settings. Data on your server is governed by your own server configuration.

## Source code

The full source is auditable: https://github.com/Dvorinka/cloudreve

## Contact

Open an issue at https://github.com/Dvorinka/cloudreve/issues
