# MasterDataMate

[日本語 README](./README.ja.md)

MasterDataMate is a schema-driven master data editor for teams that keep
table-like master data in Git. It stores canonical data as YAML, validates
records against table-local schemas, supports generation-based dataset layers,
and can export merged data through backend-specific adapters.

## Features

- Edit master data through a React/Vite frontend.
- Keep schemas and records in reviewable YAML files.
- Load and save project-local `masterdata/` workspaces.
- Open project-local custom editors for domain-specific workflows, such as image-backed enemy tuning or grid-based map editing.
- Ask the in-app AI assistant questions about the current data and use it while reviewing or editing tables.
- Generate source files or other text artifacts from validated master data with Pongo2 templates.
- Build a single Go web server binary with embedded frontend assets.
- Build a Wails desktop host that shares the Go service layer.
- Use an npm wrapper for locally built or prebuilt native binaries.

## Screenshots

### Table editing

![Table editing screen](./docs/screenshots/table-editing.jpg)

### Custom editors

Custom editors can provide focused, domain-specific interfaces on top of the
same schema-backed master data. For example, a project can expose an enemy
status editor with image upload and stat visualization, or a maze editor for
painting walls, paths, start points, goals, and enemy placements.

![Enemy status custom editor](./docs/screenshots/custom-editor.png)

![Maze grid custom editor](./docs/screenshots/custom-editor2.png)

### AI assistant

The AI assistant panel stays available from the editing workspace so users can
ask questions about visible data, compare records, and request changes while
keeping the table view in context.

![AI assistant panel](./docs/screenshots/ai-assistant.png)

## Requirements

- Node.js and npm
- Go
- Wails tooling, only when building the desktop distribution

## Install

Install JavaScript dependencies:

```bash
npm ci
```

## Development

Run the Vite development server:

```bash
npm run dev
```

Run the Node/Hono development server:

```bash
npm start
```

Run the Go web server against a workspace:

```bash
npm run start:go -- --workspace .
```

The sample workspace lives under `masterdata/`.

## Build

Build the frontend:

```bash
npm run build
```

Build the React frontend and package it into a Go server binary:

```bash
npm run build:go
./dist-native/masterdatamate --workspace .
```

The Go server embeds `dist` with `go:embed`, serves `/api/*`, serves static Vite assets, and falls back to `index.html` for SPA routes.

## Template Generation

`masterdatamate generate` renders project-defined Pongo2 templates into source
files or other generated text artifacts. Template jobs live in
`masterdata/generate_definitions.yaml`, and larger templates can be kept under
`masterdata/generate_templates/`.

The sample workspace at [examples/templates](./examples/templates) generates
SQL DDL and Go source files from weather-service master data. See its
[template definitions](./examples/templates/masterdata/generate_definitions.yaml)
and sample templates such as
[go/types.go.pongo2](./examples/templates/masterdata/generate_templates/go/types.go.pongo2)
and
[sql/schema.sql.pongo2](./examples/templates/masterdata/generate_templates/sql/schema.sql.pongo2).

Validate the sample without writing files:

```bash
masterdatamate generate --workspace examples/templates --check-only --json
```

Generate the configured files into the sample's `generated/` directory:

```bash
masterdatamate generate --workspace examples/templates --force-overwrite
```

## npm Wrapper

For local development after `npm run build:go`:

```bash
npx masterdatamate --workspace .
```

Published packages can place platform-specific binaries under `prebuilds/<platform>-<arch>/masterdatamate`.

## Desktop Build

Build the Wails desktop entrypoint:

```bash
npm run build:desktop
```

On macOS, create an `.app` bundle:

```bash
npm run package:desktop:mac
```

`wails.json` defines the frontend build hooks for Wails packaging. The desktop host reuses the same Go host layer and Vite frontend bundle as the web server.

## Make Targets

The repository also includes a `Makefile`:

```bash
make install
make backend
make desktop
make package-mac
make test
make check
make run
```

`make run` starts the Go web server with `WORKSPACE`, `HOST`, and `PORT`
variables, for example:

```bash
make run WORKSPACE=. HOST=127.0.0.1 PORT=8787
```

## Project Layout

- `src/`: React frontend
- `server/`: Node/Hono development server
- `cmd/masterdatamate/`: Go web server entrypoint
- `internal/host/`: shared Go host services
- `masterdata/`: sample schemas and generation data
- `specs/`: product and implementation specifications
- `bin/masterdatamate.js`: npm wrapper for native binaries

## License

MasterDataMate is licensed under the GNU Affero General Public License v3.0.
See [LICENSE](./LICENSE).
