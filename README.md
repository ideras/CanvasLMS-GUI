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

## Uploading grades

- Type/paste a CSV path or use Browse. Balanced quotes and surrounding whitespace
  are trimmed on paste/proceed. CSVs must be UTF-8 (a UTF-8 BOM is accepted).
- Required columns are `student_id` (or `canvas_id`) and `grade`. IDs are **Canvas
  user IDs**, not SIS/login IDs or email. Duplicate IDs/columns and non-finite
  grades are rejected. Attachment paths resolve relative to the CSV directory.
- Each attempt fetches the live, paginated student roster. Unmatched rows can be
  exported to CSV, ignored, or cancelled. Ignored rows never prepare/upload files
  or submit grades. An all-unmatched import performs no Canvas writes.
- All matched students' referenced files are checked for readability before any
  upload. File, conversion and CSV errors offer Retry/Cancel. Retry re-reads the
  current path and roster; it does not merely resume the old CSV data.
- Confirmed uploads are reused within the app session for the same target,
  absolute path and content hash. Changed files are uploaded again. This does
  **not** cover app restarts, files deleted remotely, or a lost response after
  Canvas accepted an upload; those cases can leave duplicate/orphaned files.
  Unchanged Markdown conversions are also reused when both source and generated
  PDF hashes match. Linked resources inside Markdown are not part of this key;
  edit the source or remove its generated PDF to force re-conversion.
- File uploads retain their completed-file progress bar. Grade submission uses
  a status spinner and elapsed time: Canvas's bulk-grading implementation does
  not maintain reliable intermediate `completion` values. Monitoring polls
  immediately, then backs off from 2 to 30 seconds, with a 15-minute timeout and
  at most five consecutive transient failures. Authentication failures stop
  monitoring immediately.
- Continue in background hides the wizard without stopping its worker. The
  status-bar indicator reopens it; completion/failure notifications and the
  wizard retain skipped students and Canvas-reported failures/result warnings.
  Only one upload runs at a time. Closing an active upload warns before exit;
  shutdown cancels local work and waits at most three seconds. Stopping monitoring
  or closing the app **does not cancel a job already submitted to Canvas**.
- After submission starts, a failed/uncertain outcome does not offer whole-upload
  Retry, since that could apply grades/comments twice. Check Canvas first.
  Progress `results` vary by deployment: known structured errors and upstream
  missing-user messages are identified; unrecognized results remain visible as
  warnings rather than being silently dropped.

## Testing

Run the Go test suite with:

```bash
go test ./...
go test -race ./...
```

Frontend helper tests (Node's built-in runner; no extra dependencies) and build:

```bash
npm --prefix frontend test
npm --prefix frontend run build
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
