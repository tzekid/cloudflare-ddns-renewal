# Cloudflare DDNS Renewal

Cloudflare DDNS Renewal is a lightweight Go-based tool for automatically updating Cloudflare DNS A records (including wildcard records) with your current public IP address. This is especially useful for dynamic IP setups where your IP may change periodically.

Now supports multiple domains in one run. Provide a list via the `DOMAINS` env var (comma, space, semicolon, newline, or pipe separated). Each domain's root and wildcard A records are updated to point to the same detected IP.

I personally have this running as a daemon every 30 seconds on my server to ensure that my DNS records are always up-to-date. The tool also supports Telegram notifications for successful updates.

## Features

- **Automatic IP detection:** Retrieves your current public IP address.
- **DNS record updates:** Updates both the main domain and wildcard A records on Cloudflare.
- **Multiple domains:** Update any number of zones (e.g. `DOMAINS="plosca.ru sparkdate.love"`).
- **Optimized and static:** Built as a statically linked binary with fast startup.
- **Daemon friendly:** Can easily be run as a systemd timer for continuous updates.

## Prerequisites

- [Go](https://golang.org/dl/) (if you plan to build from source)
- A valid Cloudflare account with appropriate API credentials.
- A Telegram bot token and chat ID (optional) for update notifications.

## Installation

### Building the Binary

Clone the repository and build the binary:

```bash
git clone https://github.com/yourusername/cloudflare-ddns-renewal.git
cd cloudflare-ddns-renewal
go build -trimpath -ldflags="-s -w" -o cloudflare-ddns-renewal .

```

## Configuration

Environment variables:

```dotenv
CLOUDFLARE_EMAIL=you@example.com
CLOUDFLARE_API_KEY=your_api_key
# (Optionally you could adapt code to use CLOUDFLARE_API_TOKEN instead.)

# Optional Telegram notifications
TELEGRAM_BOT_TOKEN=123456:ABCDEF
TELEGRAM_CHAT_ID=12345678

# Multiple domains (any separator: space/comma/semicolon/pipe/newline)
DOMAINS="plosca.ru sparkdate.love"
# Or a single domain
# DOMAIN=plosca.ru
```

Resolution order:
1. `DOMAINS` if set.
2. `DOMAIN` if set.
3. Fallback default `plosca.ru` (for backward compatibility).

## Usage

```bash
./cloudflare-ddns-renewal
```

Example output:

```
Current IP: 203.0.113.42
Processing domains: plosca.ru, sparkdate.love
plosca.ru already up to date (203.0.113.42)
*.plosca.ru already up to date (203.0.113.42)
Finished plosca.ru in 118ms
Updating sparkdate.love from 198.51.100.10 to 203.0.113.42
IP for sparkdate.love updated to 203.0.113.42
Updating *.sparkdate.love from 198.51.100.10 to 203.0.113.42
IP for *.sparkdate.love updated to 203.0.113.42
Finished sparkdate.love in 240ms
```

## Notes

- The program requires the zone (apex) A record and its wildcard A record to already exist; it will not create them automatically.
- Errors for any domain are reported via stdout/stderr and Telegram (if configured); the program exits non‑zero if any domain failed.
- For frequent runs (e.g. every 30s) consider adding simple rate limiting or conditional Telegram messages to avoid noise.

