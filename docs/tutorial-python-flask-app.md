# Tutorial: Using keysync with a Python Flask Application

This tutorial walks through using keysync's **Python client library** to retrieve secrets at runtime in a [Flask](https://flask.palletsprojects.com/) web application. You'll learn how to access secrets stored in the OS keychain directly from Python code — no dependency on the `keysync` binary at runtime.

## Prerequisites

- **keysync CLI** — [installed and initialized](../README.md#installation) with secrets stored
- **Python 3.11+**
- **Flask** — `pip install flask`

The same patterns shown here work with any Python web framework (FastAPI, Django, etc.).

## Step 1: Install the Python client

The Python client is a pure-Python package located at `clients/python/` in the keysync repository. Install it in your project:

```bash
# From the keysync repo root
pip install ./clients/python

# Or point to a local copy
pip install /path/to/keysync/clients/python
```

The client has **zero external dependencies** — it uses `subprocess` to call OS keychain tools on macOS/Linux and `ctypes` (built-in) for the Win32 API on Windows.

## Step 2: Store secrets with the CLI

Before your app can read secrets, store them using the keysync CLI. For this tutorial, create a global secret (available to all projects) and a project-scoped secret:

```bash
# Global secret (shared across all projects)
keysync set API_BASE_URL=https://api.example.com

# Project-scoped secret (overrides global for this project)
keysync set -p my-flask-app DATABASE_URL=postgresql://dbhost:5432/myapp
```

You can also store environment-scoped secrets:

```bash
keysync set -p my-flask-app DATABASE_URL=postgresql://prod-host:5432/proddb --env production
keysync set -p my-flask-app DATABASE_URL=postgresql://staging-host:5432/stagingdb --env staging
```

## Step 3: Write a Flask application

Create a file called `app.py`:

```python
from flask import Flask, jsonify
from keysync import get_secret, list_secrets, SecretNotFoundError

app = Flask(__name__)
PROJECT = "my-flask-app"


@app.route("/")
def index():
    """Display which secrets are available (without revealing values)."""
    try:
        secrets = list_secrets(project=PROJECT)
        names = [s["key"] for s in secrets]
    except Exception as e:
        names = []

    global_secrets = list_secrets(scope="global")
    global_names = [s["key"] for s in global_secrets]

    return jsonify({
        "project_secrets": names,
        "global_secrets": global_names,
    })


@app.route("/config")
def config():
    """Return non-sensitive configuration derived from secrets."""
    try:
        db_url = get_secret("DATABASE_URL", project=PROJECT)
    except SecretNotFoundError:
        return jsonify({"error": "DATABASE_URL not found"}), 500

    try:
        api_url = get_secret("API_BASE_URL")
    except SecretNotFoundError:
        api_url = "https://default.example.com"

    # Derive a config label from the database URL without exposing credentials
    db_label = db_url.split("@")[-1] if "@" in db_url else db_url
    return jsonify({
        "database": db_label,
        "api_base_url": api_url,
    })


if __name__ == "__main__":
    app.run(debug=True)
```

### Key points about the code

- **`get_secret(key, project=PROJECT)`** — looks up a project-scoped secret, falling back to global if not found at the project level. The resolution order is: environment-scoped → project-scoped → global.
- **`get_secret(key)`** — with no project argument, only checks the global scope.
- **`list_secrets(scope="global")`** — returns all global secrets (without values).

## Step 4: Run locally

```bash
python app.py
```

Visit `http://127.0.0.1:5000/` to see the available secret names, and `http://127.0.0.1:5000/config` to see the resolved configuration.

### Platform behavior

The Python client adapts automatically at runtime:

| Platform | Keychain mechanism | Notes |
|----------|-------------------|-------|
| **macOS** | `security find-generic-password` CLI | First access may prompt for Keychain permission |
| **Linux** | `secret-tool lookup` CLI | Requires `libsecret-tools` and an unlocked keyring |
| **Windows** | `CredReadW` Win32 API via `ctypes` | Pure Python, no external dependencies |

## Step 5: Graceful fallback for development

For local development without a keychain (e.g., in a container or CI), wrap the `get_secret` call to fall back to environment variables:

```python
import os
from keysync import get_secret, SecretNotFoundError


def get_Secret_or_env(key, project=None):
    """Try keychain first, fall back to environment variable."""
    try:
        return get_secret(key, project=project)
    except (SecretNotFoundError, Exception):
        return os.environ.get(key, "")
```

This lets developers set `DATABASE_URL=sqlite:///dev.db` in their shell without modifying the application code.

## Step 6: Deploy notes

### Production deployment

When deploying, ensure the keychain is accessible:

- **macOS servers**: The keychain is typically unlocked on desktop login. On headless macOS, unlock the keychain first.
- **Linux servers**: Install `libsecret-tools` and ensure a D-Bus session with an unlocked keyring is available.
- **Containers**: The Python client's fallback mechanism (Step 5) allows env-var override when the keychain isn't available inside a container.

### Using with Docker

```dockerfile
FROM python:3.12-slim

# Install libsecret for keychain access
RUN apt-get update && apt-get install -y libsecret-tools && rm -rf /var/lib/apt/lists/*

# Install the keysync Python client
COPY clients/python /tmp/keysync-python
RUN pip install /tmp/keysync-python && rm -rf /tmp/keysync-python

COPY app.py /app/
WORKDIR /app

CMD ["python", "app.py"]
```

## Summary

In this tutorial you:

1. Installed the keysync Python client library (zero external dependencies)
2. Stored secrets with the keysync CLI at global and project scopes
3. Retrieved secrets at runtime in a Flask application
4. Added graceful fallback to environment variables for development
5. Reviewed deployment considerations for each OS

The Python client gives you native keychain access from any Python application — no dependency on the `keysync` binary at runtime, no plaintext `.env` files in your repository.
