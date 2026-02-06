# Lumo API (private, self‑hosted)

A **privacy‑first** wrapper that lets you talk to Proton’s Lumo assistant from any
programming language. It runs a headless Chromium instance locally, logs in once,
and exposes a tiny OpenAI‑style HTTP endpoint:

```bash
POST http://localhost:8080/v1/prompt \
{ \
  "prompt": "Explain the difference between static and dynamic analysis.", \
  "webSearch": false \
}
```

## Why this exists

- Proton does **not** provide a public Lumo API (as of Feb 2026).
- The community wrapper simulates the UI with Playwright, preserving the
  zero‑access encryption guarantees because the browser talks directly to
  `lumo.proton.me` over TLS.
- The service is **self‑hosted**, so you keep full control over logs, rate limits,
  and any additional tooling (MCP servers, CI pipelines, etc.).

## Prerequisites

| Item                               | Version                |
| ---------------------------------- | ---------------------- |
| Go                                 | ≥ 1.22                 |
| Docker (optional)                  | any recent engine      |
| Chromium (installed by Playwright) | handled automatically  |
| Proton account                     | needed for `auth.json` |

## First‑time login (creates `auth.json`)

```bash
git clone https://github.com/guil/lumo-api.git
cd lumo-api
go run ./cmd/server
```

The server will detect that auth.json is missing, open a headless browser
pointing at `https://lumo.proton.me/login`, and pause:

```text
⚡ No auth.json found – opening Lumo login page.
🔐 After you log in, press ENTER to continue...
```

Complete the login (2‑FA if enabled) in the displayed window, then press Enter.
The script saves auth.json for all subsequent runs.

> Security tip: keep auth.json readable only by the service user (chmod 600 auth.json).
> It contains session cookies that grant access to your Lumo account.

## Running the API

### Locally (no Docker)

```bash
make run
# or:
go run ./cmd/server
```

The endpoint will be reachable at `http://localhost:8080/v1/promptl`.

### With Docker

```bash
make docker-build
make docker-run
```

Make sure to mount auth.json into the container (read‑only)
as shown in the Makefile target.

## Example client (Python)

```
import requests, json

payload = {
    "prompt": "Write a Go function that reverses a slice of ints.",
    "webSearch": False
}
resp = requests.post("http://localhost:8080/v1/prompt", json=payload)
print(json.loads(resp.text)["answer"])
```

