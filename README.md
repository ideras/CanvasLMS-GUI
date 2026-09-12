# CanvasLMS GUI

A cross-platform desktop client for **Canvas LMS** that helps instructors and administrators manage course information without working directly with the Canvas API.

Built with Go, Wails v3, Svelte, and Vite.

## Features

- Connect to a Canvas LMS instance using an API token
- Browse and select courses, then view course items and statistics
- Manage assignments and assignment groups
- Create, edit, and delete course announcements
- Upload and download announcement attachments
- View students and submissions
- Export student rosters and scores to CSV
- Import grades from CSV files with progress reporting and cancellation
- Convert Markdown documents to PDF

## Requirements

- Go 1.25 or later
- Node.js and npm
- [Wails v3 CLI](https://v3.wails.io/getting-started/installation/) — the project is built against the pinned version:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.20
```

- The platform dependencies required by Wails v3 for your operating system. On Linux the default build uses the GTK4 + WebKitGTK 6.0 stack (`libgtk-4-1`, `libwebkitgtk-6.0-4` on Ubuntu 24.04+/Debian 13+). A legacy GTK3 + WebKit2GTK 4.1 fallback exists via the `gtk3` build tag (supported through v3.0.x).

Verify the toolchain at any time with `wails3 doctor`.

## Development

Install dependencies and start the application in development mode:

```bash
make dev
```

Equivalent direct invocation:

```bash
wails3 task dev
```

Dev mode regenerates Go bindings into `frontend/bindings/`, builds the frontend, and launches the desktop window.

## Build

Create a production build with:

```bash
make build
```

The application binary is written to `bin/canvaslms-gui` (root `bin/`). Linux packages (deb/rpm/AppImage) are produced with `wails3 task package`.

### Regenerating frontend bindings

Bindings are regenerated automatically by the build/dev tasks. To regenerate them manually:

```bash
wails3 generate bindings -clean=true -time-type=Date -d frontend/bindings
```

Do not hand-edit files under `frontend/bindings/` — they are generated. If a build fails after changing exported `App` methods, regenerate and commit the updated bindings.

## Canvas configuration

On first launch, enter your Canvas instance URL and an API token. The settings dialog persists them to `config.json` in the working directory (or next to the executable).

To configure the app manually, copy the example file and fill in your own values:

```bash
cp config.example.json config.json
```

| Field | Type | Description |
| --- | --- | --- |
| `canvas_base_url` | string | Base URL of your Canvas instance |
| `api_token` | string | Canvas API access token |
| `last_course_id` | number | Last selected course; restored on launch (`0` for none) |
| `course_set` | number[] | Saved set of course IDs |

Credentials may also be supplied through environment variables, which take precedence over `config.json`:

- `CANVAS_BASE_URL`
- `CANVAS_API_TOKEN`

Keep credentials local—do not add tokens or the generated `config.json` file to version control. Generate a token under *Account → Settings → New Access Token*.

## Testing

Run the Go test suite with:

```bash
go test ./...
go test -race ./internal/...
```

Frontend and desktop behavior (dialogs, event delivery, window lifecycle) is validated with real desktop smoke checks; browser-only Vite preview is not sufficient proof of native integration.

## CI

`.github/workflows/build.yml` builds deployable artifacts on every push to `main`, version tag, or manual dispatch:

| Job | Runner | Output |
| --- | --- | --- |
| Test | `ubuntu-24.04` | `go vet` + `go test -race` |
| Linux | `ubuntu-24.04` | binary + `deb` + `rpm` (GTK4/WebKitGTK 6.0) |
| Windows | `ubuntu-24.04` | cross-compiled `.exe` (CGO-free; GUI not validated in CI) |
| macOS | `macos-14` | universal (arm64 + amd64) ad-hoc signed `.app` zip |

macOS requires a macOS runner (the Apple SDK cannot exist on Linux); Windows cross-compiles with `CGO_ENABLED=0` per the pinned v3 Taskfile. Artifacts are downloadable from the workflow run page for manual smoke testing on each OS.

## Tech stack

- **Backend:** Go and Wails v3 (v3.0.0-beta.20)
- **Frontend:** Svelte and Vite
- **Local storage:** SQLite
- **Document conversion:** `github.com/ideras/md-to-pdf`
