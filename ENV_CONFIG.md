# Environment Variable Configuration Guide

This document describes how to use environment variables to configure the SCIM sync tool, as an alternative to Keeper Secrets Manager.

## Overview

The SCIM sync tool now supports two configuration methods:

1. **Environment Variables** (New, Recommended) - Configure directly via environment variables
2. **Keeper Secrets Manager** (Legacy) - Load configuration from KSM records

The tool automatically detects which method to use. If all required environment variables are set, it will use them. Otherwise, it falls back to KSM configuration.

## When to Use Environment Variables

Environment variables are recommended when:
- Deploying to environments that support native environment variable management (Kubernetes, Docker, AWS Lambda, etc.)
- You want simpler configuration without KSM setup
- Running locally for development and testing
- You prefer declarative configuration over secret management systems

## Required Environment Variables

All five of these variables must be set for environment variable configuration to be used:

### `GOOGLE_CREDENTIALS`
Google Cloud Platform service account credentials in JSON format.

**Format Options:**
- Raw JSON string: `{"type":"service_account","project_id":"...","private_key":"..."}`
- Base64-encoded JSON: The tool automatically detects and decodes base64 strings

**Example:**
```bash
export GOOGLE_CREDENTIALS='{"type":"service_account","project_id":"my-project",...}'
```

Or with base64 encoding:
```bash
export GOOGLE_CREDENTIALS=$(cat credentials.json | base64)
```

### `GOOGLE_ADMIN_ACCOUNT`
The Google Workspace administrator email account that has domain-wide delegation enabled for the service account.

**Example:**
```bash
export GOOGLE_ADMIN_ACCOUNT='admin@example.com'
```

### `SCIM_GROUPS`
Comma or newline separated list of Google Workspace groups or users to synchronize.

**Supported Formats:**
- Group email addresses: `all-users@example.com`
- User email addresses: `specific-user@example.com`
- Group names (case-sensitive): `Engineering`
- Mixed formats: `group@example.com,AnotherGroup,user@example.com`

**Examples:**
```bash
# Single group
export SCIM_GROUPS='all-users@example.com'

# Multiple groups (comma-separated)
export SCIM_GROUPS='engineering@example.com,sales@example.com'

# Multiple groups (newline-separated)
export SCIM_GROUPS='engineering@example.com
sales@example.com
marketing@example.com'

# Mixed formats
export SCIM_GROUPS='Engineering,all-users@example.com,admin@example.com'
```

### `SCIM_URL`
The Keeper SCIM endpoint URL. Must contain `/api/rest/scim/v2/` in the path.

**Example:**
```bash
export SCIM_URL='https://keepersecurity.com/api/rest/scim/v2/abc123def456'
```

### `SCIM_TOKEN`
The bearer token for authenticating with the Keeper SCIM API.

**Example:**
```bash
export SCIM_TOKEN='your-secret-bearer-token-here'
```

## Optional Environment Variables

### `SCIM_VERBOSE`
Enable verbose logging to see detailed sync operations.

**Accepted Values:** `true`, `false`, `1`, `0`, `ok`

**Default:** `false`

**Example:**
```bash
export SCIM_VERBOSE=true
```

### `SCIM_DESTRUCTIVE`
Controls how the sync handles deletions of users and groups.

**Values:**
- `-1` or any non-numeric value: **Safe Mode** - No deletions are performed
- `0`: **Partial Destructive** - Only delete entities that have an externalId (SCIM-controlled)
- Positive number: **Full Destructive** - Delete all unmatched entities

**Default:** `0` (automatically becomes `-1` if load errors occur)

**Example:**
```bash
export SCIM_DESTRUCTIVE=0
```

**Important Notes:**
- Safe mode (`-1`) is automatically enabled if the tool encounters errors loading Google Workspace data
- Partial mode (`0`) only removes users/groups that were previously created via SCIM sync
- Full destructive mode (`>0`) removes all users/groups not found in Google Workspace, regardless of how they were created

## Usage Examples

### Local Development

Create a `.env` file (copy from `.env.sample`):

```bash
cp .env.sample .env
# Edit .env with your values
```

Load and run:
```bash
source .env
go run ./cmd/main.go
```

### Docker

```dockerfile
FROM golang:1.21 as builder
WORKDIR /app
COPY . .
RUN go build -o ksm-scim ./cmd/main.go

FROM debian:bookworm-slim
COPY --from=builder /app/ksm-scim /usr/local/bin/
ENV GOOGLE_CREDENTIALS=""
ENV GOOGLE_ADMIN_ACCOUNT=""
ENV SCIM_GROUPS=""
ENV SCIM_URL=""
ENV SCIM_TOKEN=""
CMD ["/usr/local/bin/ksm-scim"]
```

Run with:
```bash
docker run \
  -e GOOGLE_CREDENTIALS='{"type":"service_account",...}' \
  -e GOOGLE_ADMIN_ACCOUNT='admin@example.com' \
  -e SCIM_GROUPS='all-users@example.com' \
  -e SCIM_URL='https://keepersecurity.com/api/rest/scim/v2/...' \
  -e SCIM_TOKEN='your-token' \
  ksm-scim
```

### Kubernetes

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: scim-config
type: Opaque
stringData:
  GOOGLE_CREDENTIALS: |
    {"type":"service_account",...}
  GOOGLE_ADMIN_ACCOUNT: admin@example.com
  SCIM_GROUPS: all-users@example.com,engineering@example.com
  SCIM_URL: https://keepersecurity.com/api/rest/scim/v2/...
  SCIM_TOKEN: your-secret-token
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: scim-sync
spec:
  schedule: "15 * * * *"  # Every hour at :15
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: scim-sync
            image: your-registry/ksm-scim:latest
            envFrom:
            - secretRef:
                name: scim-config
          restartPolicy: OnFailure
```

### Google Cloud Functions

When deploying to GCP, you can now use direct environment variables instead of the legacy `.env.yaml` format:

```bash
gcloud functions deploy ScimSync \
  --gen2 \
  --runtime=go121 \
  --entry-point=GcpScimSyncHttp \
  --trigger-http \
  --no-allow-unauthenticated \
  --set-env-vars "GOOGLE_CREDENTIALS=$(cat credentials.json | base64)" \
  --set-env-vars "GOOGLE_ADMIN_ACCOUNT=admin@example.com" \
  --set-env-vars "SCIM_GROUPS=all-users@example.com" \
  --set-env-vars "SCIM_URL=https://keepersecurity.com/api/rest/scim/v2/..." \
  --set-env-vars "SCIM_TOKEN=your-token"
```

Or use Secret Manager for sensitive values:
```bash
gcloud functions deploy ScimSync \
  --gen2 \
  --runtime=go121 \
  --entry-point=GcpScimSyncHttp \
  --trigger-http \
  --no-allow-unauthenticated \
  --set-secrets "GOOGLE_CREDENTIALS=google-creds:latest" \
  --set-secrets "SCIM_TOKEN=scim-token:latest" \
  --set-env-vars "GOOGLE_ADMIN_ACCOUNT=admin@example.com" \
  --set-env-vars "SCIM_GROUPS=all-users@example.com" \
  --set-env-vars "SCIM_URL=https://keepersecurity.com/api/rest/scim/v2/..."
```

## Migration from KSM Configuration

If you're currently using Keeper Secrets Manager configuration, here's how to migrate:

1. Extract values from your KSM record:
   - URL field → `SCIM_URL`
   - Password field → `SCIM_TOKEN`
   - Login field → `GOOGLE_ADMIN_ACCOUNT`
   - credentials.json file → `GOOGLE_CREDENTIALS`
   - "SCIM Group" custom field → `SCIM_GROUPS`
   - "Verbose" custom field → `SCIM_VERBOSE`
   - "Destructive" custom field → `SCIM_DESTRUCTIVE`

2. Set the environment variables in your deployment environment

3. Remove or unset `KSM_CONFIG_BASE64` and `KSM_RECORD_UID` (the tool will automatically use environment variables when they're available)

## Troubleshooting

### Tool still using KSM configuration

Make sure ALL five required environment variables are set. If even one is missing, the tool falls back to KSM configuration.

Check which configuration is being used by looking at the log output:
```
Loading configuration from environment variables
```
or
```
Loading configuration from Keeper Secrets Manager
```

### Invalid JSON error for GOOGLE_CREDENTIALS

Ensure the JSON is properly formatted and quoted:
```bash
# Good
export GOOGLE_CREDENTIALS='{"type":"service_account",...}'

# Bad - missing quotes
export GOOGLE_CREDENTIALS={"type":"service_account",...}
```

Or use base64 encoding to avoid shell escaping issues:
```bash
export GOOGLE_CREDENTIALS=$(cat credentials.json | base64)
```

### No groups found error

Verify that `SCIM_GROUPS` contains valid group emails, user emails, or exact group names (case-sensitive for names).

Enable verbose logging to see resolution details:
```bash
export SCIM_VERBOSE=true
```

## Security Best Practices

1. **Never commit credentials to version control**
   - Add `.env` to `.gitignore`
   - Use secret management systems in production

2. **Use base64 encoding for complex JSON**
   - Avoids shell escaping issues
   - Safer for CI/CD pipelines

3. **Rotate tokens regularly**
   - Update `SCIM_TOKEN` periodically
   - Rotate service account keys for `GOOGLE_CREDENTIALS`

4. **Use secret management systems in production**
   - Google Secret Manager for GCP
   - AWS Secrets Manager for AWS
   - Kubernetes Secrets for K8s
   - HashiCorp Vault for multi-cloud

5. **Limit service account permissions**
   - Only grant necessary Google Workspace scopes
   - Use separate service accounts per environment

## Implementation Details

For developers working on this codebase:

- Configuration loading logic: `scim/env_config.go`
- Configuration detection: `scim.IsEnvConfigAvailable()`
- Entry point selection: `gcp_function.go:runScimSync()` and `cmd/main.go`
- KSM fallback: Automatic if env vars not complete
