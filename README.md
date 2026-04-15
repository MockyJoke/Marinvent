# Marinvent Chart API

A Go-based API for accessing and exporting Jeppesen terminal charts from TCL files.

This project is a clone of [StarNumber12046/Marinvent](https://github.com/StarNumber12046/Marinvent), and this work would not exist without their original reverse-engineering and implementation effort. Many thanks to them for building the foundation.

This fork primarily updates the project so it can run on a Linux amd64 machine.

## Requirements

- **Linux amd64 or Windows**
- **Go 1.25+** (for build)
- **Python 3.10+** (for pdf cleanup)
- **Jeppesen data files**:
  - `charts.bin` - Chart archive used to extract TCL files
  - `charts.dbf` - Chart database
  - `ctypes.dbf` - Chart type definitions
  - `airports.dbf` - Airport database
  - `vfrchrts.dbf` - Optional VFR chart database

On Linux, the required Wine-side DLLs, fonts, and config files are already
provided in the repository/image. The only external data users need to supply is
the live chart data.

## Installation

### A video version is available [on YouTube](https://youtu.be/UUESsbvshbc)

### Linux amd64

1. Place your live chart data under `win/data/Charts/`, including `charts.bin`.
2. Extract TCL files with:

   ```bash
   mkdir -p TCLs
   cd TCLs
   python3 ../jdmtool/chartview.py -x ../win/data/Charts/charts.bin
   ```

3. Build and run the server:

   ```bash
   go build -o marinvent-api ./cmd/server/
   ./marinvent-api
   ```

### Windows

The original upstream Windows-oriented workflow still applies: provide the
Jeppesen DLLs/fonts/config files plus the chart DBFs and extracted TCLs, then
run the server binary. A GUI client like
[BetterJepp](https://github.com/StarNumber12046/BetterJepp) can be used on top.

## Quick Start

### 1. Clone and Build

```bash
# Build CLI and API server
go build -o marinvent ./cmd/cli/
go build -o marinvent-api ./cmd/server/
```

### 2. Directory Structure

```
Marinvent/
├── marinvent             # CLI binary (Linux build)
├── marinvent-api         # API server binary (Linux build)
├── win_deps/             # Baked Windows DLL/font dependencies for Linux or Docker
│   ├── lib/
│   └── fonts/
├── win/
│   └── data/Charts/      # Live Jeppesen chart data (charts.bin, DBFs, etc.)
├── TCLs/                 # Extracted TCL chart files (optional outside Docker)
├── cmd/
│   ├── cli/              # CLI source
│   └── server/           # API server source
└── internal/
    ├── api/              # HTTP handlers
    ├── charts/           # Chart catalog
    ├── dbf/              # DBF parsing
    └── export/            # Export functionality
```

### 3. Run the API Server

```bash
# Default configuration
./marinvent-api

# Linux custom configuration
./marinvent-api -port 9000 -host 127.0.0.1 \
  -charts "win/data/Charts/charts.dbf" \
  -types "win/data/Charts/ctypes.dbf" \
  -airports "win/data/Charts/airports.dbf" \
  -tcls "TCLs"
```

### 4. Use the API

```bash
# Health check
curl http://localhost:8080/health

# Get OpenAPI spec
curl http://localhost:8080/openapi.json

# List all KJFK charts
curl http://localhost:8080/api/v1/charts/KJFK

# Filter KJFK by type name (RNAV, ILS, etc.)
curl "http://localhost:8080/api/v1/charts/KJFK?type=RNAV"

# Filter KJFK by type code
curl "http://localhost:8080/api/v1/charts/KJFK?type=1L"

# Filter KJFK by procedure name
curl "http://localhost:8080/api/v1/charts/KJFK?search=RWY+30"

# Combine type and search
curl "http://localhost:8080/api/v1/charts/KJFK?type=RNAV&search=RWY+30"

# List chart types
curl http://localhost:8080/api/v1/chart-types

# Export chart to PDF
curl -o chart.pdf "http://localhost:8080/api/v1/charts/KJFK/export/KJFK225"
```

## Docker

The container image includes Wine, the built Windows helper executables,
Jeppesen fonts/config files, the required Jeppesen DLLs, CUPS PDF printing,
and automatic chart extraction.
The **only** path you need to mount is the live Jeppesen chart data directory at:

```text
/app/data/Charts
```

That mounted directory should contain at least:

- `charts.bin`
- `charts.dbf`
- `ctypes.dbf`
- `Airports.dbf`
- `vfrchrts.dbf` (optional)

The Docker builder compiles `tcl2emf.exe` and `georef_tool.exe`, and the
required Wine-side dependencies are already included in the repository.

```bash
docker build -f docker/Dockerfile -t marinvent .
```

Run it by mounting the changing chart-data directory into the single expected
volume path:

```bash
docker run --rm -p 8080:8080 \
  -v /path/to/Charts:/app/data/Charts \
  marinvent
```

The repository also includes a GitHub Actions workflow that publishes the image
to GHCR as:

```text
ghcr.io/<github_username>/marinvent/marinvent-webserver
```

Tagging strategy:

- `main` for pushes to the default branch
- `sha-<commit>` for every published build
- `vX.Y.Z`, `X.Y`, and `X` for release tags like `v1.2.3`
- `latest` only for stable release tags, not for branch pushes or pre-releases

On startup the container will:

1. start CUPS with a PDF printer
2. initialize Wine
3. extract `.tcl` files into `/app/data/Charts/TCLs`
4. start the API server on port `8080`

The extracted `TCLs/` directory is treated as a cache. The entrypoint stores a
checksum of `charts.bin` in the mounted directory and only reuses `TCLs/` when
the checksum matches. If `charts.bin` changes, the container rebuilds the cache
in a temporary directory and swaps it into place, so incomplete or stale TCLs
are not reused.

## Environment Variables

The API server respects these environment variables:

| Variable     | Default                                                    | Description                    |
| ------------ | ---------------------------------------------------------- | ------------------------------ |
| `PORT`       | `8080`                                                     | HTTP server port               |
| `HOST`       | `0.0.0.0`                                                  | HTTP server host               |
| `CHARTS_DBF` | Linux: `win/data/Charts/charts.dbf`; Windows: `C:\ProgramData\Jeppesen\Common\TerminalCharts\charts.dbf` | Charts DBF path |
| `VFR_CHARTS_DBF` | Linux: `win/data/Charts/vfrchrts.dbf`; Windows: `C:\ProgramData\Jeppesen\Common\TerminalCharts\vfrchrts.dbf` | VFR charts DBF path |
| `TYPES_DBF`  | Linux: `win/data/Charts/ctypes.dbf`; Windows: `C:\ProgramData\Jeppesen\Common\TerminalCharts\ctypes.dbf` | Chart types DBF path |
| `AIRPORTS_DBF` | Linux: `win/data/Charts/airports.dbf`; Windows: `C:\ProgramData\Jeppesen\Common\TerminalCharts\Airports.dbf` | Airports DBF path |
| `TCL_DIR`    | `TCLs`                                                     | Directory containing TCL files |

## CLI Usage

```bash
# List all charts for an ICAO
./marinvent -icao KJFK

# Search charts
./marinvent -search RNAV

# Filter by type name
./marinvent -type ILS

# List all chart types
./marinvent -list-types

# Export charts
./marinvent -icao KJFK -export output/
```

## API Endpoints

| Method | Endpoint                                  | Description                       |
| ------ | ----------------------------------------- | --------------------------------- |
| `GET`  | `/health`                                 | Health check                      |
| `GET`  | `/swagger/index.html`                     | Swagger UI (interactive API docs) |
| `GET`  | `/swagger.json`                           | OpenAPI 3.0 specification         |
| `GET`  | `/api/v1/charts/{icao}`                   | List charts for ICAO              |
| `GET`  | `/api/v1/charts/{icao}/export/{filename}` | Export chart to PDF               |
| `GET`  | `/api/v1/chart-types`                     | List all chart types              |

### Query Parameters for `/api/v1/charts/{icao}`

| Parameter | Type   | Description                                                      |
| --------- | ------ | ---------------------------------------------------------------- |
| `type`    | string | Filter by chart type (code like `1L` or name like `RNAV`, `ILS`) |
| `search`  | string | Search in procedure name (PROC_ID)                               |

## Type Lookup

The `type` parameter supports both:

- **Raw codes**: `1L`, `AP`, `01`, etc.
- **Human-readable names**: `RNAV`, `ILS`, `VOR`, `AIRPORT`, etc.

When using a name, the API searches both the `TYPE` and `CATEGORY` fields in `ctypes.dbf` and returns all matching chart types.

Examples:

- `?type=RNAV` → matches codes `1L`, `1C`, etc. (all RNAV types)
- `?type=ILS` → matches codes `01`, `1K`, `2A`, etc. (all ILS types)
- `?type=AP` → matches code `AP` (Airport)

## Data Sources

This tool works with Jeppesen terminal chart data. The DBF files are typically located at:

- `C:\ProgramData\Jeppesen\Common\TerminalCharts\charts.dbf`
- `C:\ProgramData\Jeppesen\Common\TerminalCharts\ctypes.dbf`

On Linux, the live chart data is typically staged under `win/data/Charts/` or
mounted into `/app/data/Charts` in Docker. TCL files are extracted from
`charts.bin`.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        API Server                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │  Gin     │  │ Charts   │  │   DBF    │  │ Export   │  │
│  │  Router  │──│ Service  │──│ Parser   │──│ Service  │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
└─────────────────────────────────────────────────────────────┘
         │                    │                   │
         ▼                    ▼                   ▼
     HTTP Client         DBF Files          tcl2emf.exe
                                           (Wine-backed on Linux)
```

## License

This is reverse-engineered software for educational purposes. Use in accordance with Jeppesen's terms of service.
