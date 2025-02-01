# Cloudflare DDNS Renewal

Cloudflare DDNS Renewal is a lightweight Go-based tool for automatically updating Cloudflare DNS A records (including wildcard records) with your current public IP address. This is especially useful for dynamic IP setups where your IP may change periodically.

## Features

- **Automatic IP detection:** Retrieves your current public IP address.
- **DNS record updates:** Updates both the main domain and wildcard A records on Cloudflare.
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

