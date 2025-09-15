# Configuration Manager

## 1. Setup & Running Instructions

### Prerequisites

- Go 1.20+
- SQLite3 (for local database)

### Installation

1. Clone the repository:
   ```sh
   git clone <repo-url>
   cd config-manager
   ```
2. Install dependencies:
   ```sh
   go mod tidy
   ```
3. Run database migrations:
   ```sh
   sqlite3 data/config.db < migrations/000001_initial_schema.up.sql
   ```
4. Start the server:
   ```sh
   go run cmd/server/main.go
   ```
   Or build and run:
   ```sh
   go build -o bin/config-server cmd/server/main.go
   ./bin/config-server
   ```

## 2. Accessing API Documentation (Swagger)

API documentation is auto-generated using [Swaggo](https://github.com/swaggo/swag).

1. Generate docs:
   ```sh
   make api-docs
   ```
   This will create `docs/swagger.yaml` and `docs/swagger.json`.
2. Serve Swagger UI:
    - The server exposes Swagger UI at: `http://localhost:8080/swagger/index.html`
    - Or use [Swagger Editor](https://editor.swagger.io/) and import `docs/swagger.yaml`.

## 3. Schema Explanation

### Database Schema

- **configurations**: Stores configuration metadata (name, current version, timestamps)
- **versions**: Stores each version of configuration data (version number, JSON data, timestamps)

#### Table: configurations

| Column          | Type    | Description             |
|-----------------|---------|-------------------------|
| name            | TEXT    | Unique config name (PK) |
| current_version | INTEGER | Latest version number   |
| created_at      | TEXT    | Creation timestamp      |
| updated_at      | TEXT    | Last update timestamp   |

#### Table: versions

| Column             | Type    | Description                   |
|--------------------|---------|-------------------------------|
| id                 | INTEGER | Version row ID (PK)           |
| configuration_name | TEXT    | Foreign key to configurations |
| version_number     | INTEGER | Version number                |
| json_data          | TEXT    | Configuration data (JSON)     |
| created_at         | TEXT    | Version creation timestamp    |

### Configuration Data Schema

- Each configuration's `data` field must match the expected schema, e.g.:
  ```json
  {
    "max_limit": 1000,
    "enabled": true
  }
  ```
- Schema validation is enforced by the service layer.

## 4. Design Decisions & Trade-offs

### Why Echo Framework?

- **Performance**: Echo is lightweight and fast for REST APIs.
- **Developer Experience**: Simple routing, middleware, and request/response handling.
- **Swagger Integration**: Easy to use with Swaggo for API docs.
- **Extensibility**: Supports middleware, validation, and error handling out of the box.

### Why SQLite?

- **Persistence**: Data survives server restarts, unlike in-memory solutions.
- **Simplicity**: No external DB server required; file-based, easy to set up.
- **Transactions**: Supports ACID transactions for safe updates and rollbacks.
- **Scalability**: Suitable for small/medium projects and local development.

#### Alternatives Considered

- **File-based (JSON)**: Simpler, but lacks concurrency safety, versioning, and transactional integrity.
- **In-memory (Map)**: Fast, but data is lost on restart and not suitable for production.
- **Other DBs**: Overkill for a lightweight config manager; SQLite is sufficient and portable.

### Trade-offs

- **SQLite vs File-based**: Chose SQLite for reliability, atomic updates, and easy querying/versioning.
- **Echo vs net/http**: Echo provides more features and better DX for REST APIs.
- **Swagger via Swaggo**: Enables automatic, up-to-date API docs for consumers and developers.

## 5. Makefile Commands

The following commands are available via `make`:

| Command         | Usage                  | Description                                           |
|-----------------|------------------------|-------------------------------------------------------|
| format          | `make format`          | Formats code using `bin/format.sh`.                   |
| tidy            | `make tidy`            | Runs `go mod tidy` to clean up go.mod/go.sum.         |
| lint            | `make lint`            | Runs `golangci-lint` on the codebase.                 |
| check.import    | `make check.import`    | Checks import statements using `bin/check-import.sh`. |
| lint.cleancache | `make lint.cleancache` | Cleans golangci-lint cache.                           |
| pretty          | `make pretty`          | Runs tidy, format, and lint.                          |
| mod.download    | `make mod.download`    | Downloads Go module dependencies.                     |
| test            | `make test`            | Run tests.                                            |
| vendor          | `make vendor`          | Vendors Go dependencies.                              |
| api-docs        | `make api-docs`        | Generates Swagger API documentation.                  |
| build.docker    | `make build.docker`    | Builds Docker image tagged with the current git hash. |

## 6. Future Improvements

- Schema should be dynamic instead of hardcoded.
- Add authentication/authorization for config access, at least basic authentication.

## 5. Running with Docker

You can run the Configuration Manager using Docker. Follow these steps:

### Step 1: Build the Docker image

Run the following command in the project root:
```sh
docker build -t config-manager .
```
This will build a Docker image named `config-manager` using the provided Dockerfile.

### Step 2: Run the Docker container

Start the container with:
```sh
docker run -d \
  -p 8080:8080 \
  -e PORT=8080 \
  -e DB_PATH=/app/data/config.db \
  -v $(pwd)/data:/app/data \
  --name config-manager \
  config-manager
```
- The API will be available at `http://localhost:8080`.
- Swagger UI will be available at `http://localhost:8080/swagger/index.html`.
- The SQLite database will be persisted in the `data/` directory on your host.

### Step 3: Environment Variables

- `PORT`: Port to expose the API (default: 8080)
- `DB_PATH`: Path to the SQLite DB file (default: `./data/config.db` inside the container)

### Step 4: Notes
- Bruno collections is provided inside the `bruno` directory for local development and testing.
- The container does **not** support hot-reload (intended for production use).
- The image is not published to any registry; build locally as needed.
---
