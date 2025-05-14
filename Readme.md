# GoAvatar

GoAvatar is a lightweight Go service that dynamically generates avatar images (PNG or SVG).

## 🚀 Features

- Output as PNG or SVG
- Auto-generated initials from names
- GitHub username support (auto-fetch name)
- Supports custom **text color** via `color` query param (named or hex)
- Docker & Docker Compose compatible

---

## 📦 Running the App

### ▶️ With Docker Compose

```bash
docker compose up --build
```

This will build and run the GoAvatar service on port `8080`.

---

### ▶️ Manual Run (without Docker)

```bash
go build ./src
./goavatar
```

Or for quick dev run:

```bash
go run ./src
```

Or via shell script:

```bash
chmod +x run.sh
./run.sh
```

---

## 🔧 API Endpoint

```http
GET /avatar/{name,email,any}
GET (for github) /avatar/github/{github_username}
```

### Query Parameters

| Param      | Description                               | Example Value     |
|------------|-------------------------------------------|-------------------|
| `type`     | Output type (`svg` or `png`)              | `type=svg`        |
| `initials` | Custom initials override (auto by default)| `initials=JD`     |
| `w`        | Width/height (square, max: 1080, default: 120)          | `w=300`           |
| `color`    | **Text color** (only black/white)              | `color=white`   |

---

## 🖼️ Example Usage

```http
GET /avatar/octocat?type=svg&color=black&initials=auto
```


```http
GET /avatar/github/onurartan
```
