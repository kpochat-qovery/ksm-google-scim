# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based SCIM synchronization tool that syncs Google Workspace Users/Groups with Keeper Enterprise Users/Teams. It can run as either a standalone CLI application or as a Google Cloud Function (HTTP or PubSub triggered).

The project replicates the `keeper scim push --source=google` Commander CLI command and shares configuration settings with it.

## Development Commands

### Build and Run
```bash
# Build the standalone CLI application
go build -o ksm-scim ./cmd/main.go

# Run with environment variables (recommended)
# See .env.sample for all available variables
export GOOGLE_CREDENTIALS='{"type":"service_account",...}'
export GOOGLE_ADMIN_ACCOUNT='admin@example.com'
export SCIM_GROUPS='group1@example.com,group2@example.com'
export SCIM_URL='https://keepersecurity.com/api/rest/scim/v2/...'
export SCIM_TOKEN='your-bearer-token'
./ksm-scim

# Or run with KSM config file (legacy method)
# Requires config.base64 file in current dir or home directory
./ksm-scim [optional-record-uid]

# Run with Go
go run ./cmd/main.go
```

### Dependencies
```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### Testing and Linting
```bash
# Run tests (if any exist)
go test ./...

# Run tests with verbose output
go test -v ./...

# Check for code issues
go vet ./...

# Format code
go fmt ./...
```

## Architecture

### Entry Points

1. **CLI Application** (`cmd/main.go`): Standalone command-line tool that reads KSM configuration from `config.base64` file
2. **GCP HTTP Function** (`gcp_function.go:gcpScimSyncHttp`): HTTP-triggered Cloud Function
3. **GCP PubSub Function** (`gcp_function.go:gcpScimSyncPubSub`): PubSub-triggered Cloud Function

All entry points use the same core sync logic via `runScimSync()`.

### Core Components

#### SCIM Package (`scim/`)

The synchronization logic is organized around interfaces and implementations:

- **`ICrmDataSource`** (`scim/scim_data.go`): Interface for external CRM data sources (e.g., Google Workspace)
  - `googleEndpoint` (`scim/google_endpoint.go`): Google Workspace implementation

- **`IScimSync`** (`scim/scim_data.go`): Main synchronization interface
  - `sync` (`scim/sync.go`): Core sync orchestrator that coordinates user/group/membership sync

- **SCIM API Client** (`scim/scim_api.go`): HTTP client for Keeper SCIM endpoints
  - Handles GET, POST, PATCH, DELETE operations
  - Supports pagination for large datasets

#### Sync Flow

The synchronization happens in three phases (see `sync.Sync()` in `scim/sync.go:47`):

1. **Group Sync** (`syncGroups`): Creates, updates, or deletes groups
   - Three-round matching algorithm: by ExternalId, by name (case-insensitive), then position-based
   - Groups are matched, patched if different, and new ones are created

2. **User Sync** (`syncUsers`): Creates, updates, or deletes users
   - Matches users by email (case-insensitive)
   - Updates user attributes (name, active status) if changed
   - Only adds active users; skips inactive users during creation

3. **Membership Sync** (`syncMembership`): Synchronizes group memberships
   - Adds users to groups and removes them from groups
   - Respects "destructive" mode settings

#### Destructive Mode

The sync supports different levels of data deletion (see `sync.destructive` field):

- **`destructive > 0`**: Full destructive mode - deletes all unmatched entities and removes all memberships
- **`destructive == 0`**: Partial destructive mode - only deletes entities with ExternalId (SCIM-controlled)
- **`destructive < 0`**: Safe mode - no deletions (automatically enabled if load errors occur)

#### Configuration

The tool supports two configuration methods. It automatically detects which method to use:

**Method 1: Environment Variables** (`scim/env_config.go:LoadScimParametersFromEnv()`)

The tool first checks if all required environment variables are set. If yes, it uses this method:

- **Required environment variables**:
  - `GOOGLE_CREDENTIALS`: GCP service account credentials JSON (can be base64 encoded or raw JSON)
  - `GOOGLE_ADMIN_ACCOUNT`: Google Workspace admin account email
  - `SCIM_GROUPS`: Comma or newline separated list of Google groups/users to sync
  - `SCIM_URL`: SCIM endpoint URL (must contain `/api/rest/scim/v2/`)
  - `SCIM_TOKEN`: SCIM bearer token

- **Optional environment variables**:
  - `SCIM_VERBOSE`: Enable verbose logging (true/false/1/0)
  - `SCIM_DESTRUCTIVE`: Control deletion behavior (-1, 0, or positive integer)

**Method 2: Keeper Secrets Manager** (`scim/ksm_utils.go:LoadScimParametersFromRecord()`)

If environment variables are not available, falls back to KSM configuration:

- **Required fields in KSM record**:
  - `url`: SCIM endpoint URL (must contain `/api/rest/scim/v2/`)
  - `password`: SCIM bearer token
  - `login`: Google Workspace admin account (subject for JWT)
  - File attachment: `credentials.json` (GCP service account JWT credentials)
  - Custom field: `SCIM Group` - specifies which Google Workspace groups to sync

- **Optional fields**:
  - Custom field `Verbose`: Enable verbose logging
  - Custom field `Destructive`: Control deletion behavior (-1, 0, or positive integer)

The configuration selection logic is in `gcp_function.go:runScimSync()` and `cmd/main.go`.

### Google Workspace Integration

The `googleEndpoint` (`scim/google_endpoint.go`) loads data from Google Workspace:

1. **Resolves "SCIM Group" entries**: Can be group emails, user emails, or group names
2. **Loads all users** from the workspace (paginated, 200 per page)
3. **Expands group memberships**: Recursively includes nested groups
4. **Error handling**: Switches to safe mode if any resolution errors occur

### Cloud Function Deployment

**Configuration Options:**

*Option 1: Direct Environment Variables (Recommended)*
Set these environment variables for direct configuration:
- `GOOGLE_CREDENTIALS`: GCP service account credentials JSON
- `GOOGLE_ADMIN_ACCOUNT`: Google Workspace admin account
- `SCIM_GROUPS`: Comma-separated list of groups
- `SCIM_URL`: SCIM endpoint URL
- `SCIM_TOKEN`: SCIM bearer token
- `SCIM_VERBOSE`: (Optional) Enable verbose logging
- `SCIM_DESTRUCTIVE`: (Optional) Deletion behavior

*Option 2: Keeper Secrets Manager (Legacy)*
- `KSM_CONFIG_BASE64`: Base64-encoded KSM configuration
- `KSM_RECORD_UID`: (Optional) UID to filter specific SCIM record

**Entry Points:**
- HTTP: `GcpScimSyncHttp`
- PubSub: `GcpScimSyncPubSub`

The function automatically selects the configuration method based on which environment variables are present.

## Important Notes

- The project uses Go 1.21 and the Functions Framework for GCP
- All SCIM API operations use bearer token authentication
- Pagination is handled automatically (500 items per page for SCIM, 200 for Google API)
- User matching is case-insensitive for emails
- Group matching tries multiple strategies (ExternalId, name, position)
- The sync is designed to be idempotent - running it multiple times produces the same result
