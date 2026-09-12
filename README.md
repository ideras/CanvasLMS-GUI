# CanvasLMS GUI

A cross-platform desktop client for **Canvas LMS** that helps instructors and administrators manage course information without working directly with the Canvas API.

Built with Go, Wails, Svelte, and Vite.

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
- [Wails v2](https://wails.io/docs/gettingstarted/installation/)
- The platform dependencies required by Wails for your operating system

## Development

Install the frontend dependencies:

```bash
cd frontend
npm install
cd ..
```

Start the application in development mode:

```bash
make dev
```

Alternatively, run:

```bash
wails dev -tags webkit2_41
```

## Build

Create a production build with:

```bash
make build
```

The application binary is written to `build/bin/`.

## Canvas configuration

On first launch, enter your Canvas instance URL and an API token. Keep credentials local—do not add tokens or the generated `config.json` file to version control.

## Testing

Run the Go test suite with:

```bash
go test ./...
```

## Tech stack

- **Backend:** Go and Wails v2
- **Frontend:** Svelte and Vite
- **Local storage:** SQLite
- **Document conversion:** `github.com/ideras/md-to-pdf`
