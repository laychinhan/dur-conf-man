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

## 6. API Endpoints Documentation

The Configuration Management API provides the following endpoints for managing versioned configurations:

### Base URL
```
http://localhost:8080
```

### 1. Create Configuration
**POST** `/api/v1/configs`

Creates a new configuration with version 1.

**Request Body:**
```json
{
  "name": "feature-toggle-new",
  "data": {
    "max_limit": 500,
    "enabled": true
  }
}
```

**Example cURL:**
```bash
curl -X POST http://localhost:8080/api/v1/configs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "feature-toggle-new",
    "data": {
      "max_limit": 500,
      "enabled": true
    }
  }'
```

**Success Response (201):**
```json
{
  "success": true,
  "message": "Configuration created successfully",
  "data": {
    "name": "feature-toggle-new",
    "version": 1,
    "created_at": "2025-09-15T10:30:00Z"
  }
}
```

**Error Responses:**
- **400 Bad Request**: Invalid JSON or missing required fields
- **409 Conflict**: Configuration with the same name already exists
- **422 Unprocessable Entity**: Data validation failed

---

### 2. Get Latest Configuration
**GET** `/api/v1/configs/{name}`

Retrieves the latest version of a configuration.

**Path Parameters:**
- `name` (string): Configuration name

**Example cURL:**
```bash
curl -X GET http://localhost:8080/api/v1/configs/feature-toggle-new
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Configuration retrieved successfully",
  "data": {
    "name": "feature-toggle-new",
    "version": 3,
    "data": {
      "max_limit": 500,
      "enabled": true
    },
    "created_at": "2025-09-15T10:30:00Z",
    "updated_at": "2025-09-15T11:45:00Z"
  }
}
```

**Error Responses:**
- **404 Not Found**: Configuration does not exist

---

### 3. Update Configuration
**PUT** `/api/v1/configs/{name}`

Updates an existing configuration and increments the version number.

**Path Parameters:**
- `name` (string): Configuration name

**Request Body:**
```json
{
  "data": {
    "max_limit": 800,
    "enabled": false
  }
}
```

**Example cURL:**
```bash
curl -X PUT http://localhost:8080/api/v1/configs/feature-toggle-new \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "max_limit": 800,
      "enabled": false
    }
  }'
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Configuration updated successfully",
  "data": {
    "name": "feature-toggle-new",
    "version": 4,
    "previous_version": 3,
    "updated_at": "2025-09-15T12:00:00Z"
  }
}
```

**Error Responses:**
- **400 Bad Request**: Invalid JSON or missing data field
- **404 Not Found**: Configuration does not exist
- **422 Unprocessable Entity**: Data validation failed

---

### 4. List Configuration Versions
**GET** `/api/v1/configs/{name}/versions`

Returns a list of all version numbers and their creation timestamps for a configuration.

**Path Parameters:**
- `name` (string): Configuration name

**Example cURL:**
```bash
curl -X GET http://localhost:8080/api/v1/configs/feature-toggle-new/versions
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Configuration versions retrieved successfully",
  "data": {
    "name": "feature-toggle-new",
    "versions": [
      {
        "version": 1,
        "created_at": "2025-09-15T10:30:00Z"
      },
      {
        "version": 2,
        "created_at": "2025-09-15T10:45:00Z"
      },
      {
        "version": 3,
        "created_at": "2025-09-15T11:45:00Z"
      },
      {
        "version": 4,
        "created_at": "2025-09-15T12:00:00Z"
      }
    ]
  }
}
```

**Error Responses:**
- **404 Not Found**: Configuration does not exist

---

### 5. Get Specific Configuration Version
**GET** `/api/v1/configs/{name}/versions/{version}`

Retrieves a specific version of a configuration.

**Path Parameters:**
- `name` (string): Configuration name
- `version` (integer): Version number

**Example cURL:**
```bash
curl -X GET http://localhost:8080/api/v1/configs/feature-toggle-new/versions/2
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Configuration version retrieved successfully",
  "data": {
    "name": "feature-toggle-new",
    "version": 2,
    "data": {
      "max_limit": 300,
      "enabled": true
    },
    "created_at": "2025-09-15T10:45:00Z"
  }
}
```

**Error Responses:**
- **400 Bad Request**: Invalid version number
- **404 Not Found**: Configuration or version does not exist

---

### 6. Rollback Configuration
**POST** `/api/v1/configs/{name}/rollback`

Reverts the configuration to a specified version and increments the current version.

**Path Parameters:**
- `name` (string): Configuration name

**Request Body:**
```json
{
  "target_version": 1
}
```

**Example cURL:**
```bash
curl -X POST http://localhost:8080/api/v1/configs/feature-toggle-new/rollback \
  -H "Content-Type: application/json" \
  -d '{
    "target_version": 1
  }'
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Configuration rolled back successfully",
  "data": {
    "name": "feature-toggle-new",
    "new_version": 5,
    "rolled_back_to_version": 1,
    "rolled_back_at": "2025-09-15T12:15:00Z"
  }
}
```

**Error Responses:**
- **400 Bad Request**: Invalid target version or missing target_version field
- **404 Not Found**: Configuration or target version does not exist

---

### Common Response Format

All API responses follow this format:

**Success Response:**
```json
{
  "success": true,
  "message": "Operation description",
  "data": { /* Response data */ }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": { /* Additional error details */ }
  }
}
```

### HTTP Status Codes

- **200 OK**: Request successful
- **201 Created**: Resource created successfully
- **400 Bad Request**: Invalid request format or parameters
- **404 Not Found**: Resource not found
- **409 Conflict**: Resource already exists
- **422 Unprocessable Entity**: Validation failed
- **500 Internal Server Error**: Server error

---

## 7. Future Improvements

- Schema should be dynamic instead of hardcoded.
- Add authentication/authorization for config access, at least basic authentication.

## 8. Running with Docker

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
