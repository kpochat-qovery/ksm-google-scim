package scim

import (
	"encoding/base64"
	"errors"
	"os"
	"strconv"
	"strings"
)

// LoadScimParametersFromEnv loads SCIM configuration from environment variables
// instead of Keeper Secrets Manager.
//
// Required environment variables:
//   - GOOGLE_CREDENTIALS: GCP service account credentials JSON (can be base64 encoded)
//   - GOOGLE_ADMIN_ACCOUNT: Google Workspace admin account email
//   - SCIM_GROUPS: Comma or newline separated list of Google groups/users to sync
//   - SCIM_URL: SCIM endpoint URL
//   - SCIM_TOKEN: SCIM bearer token
//
// Optional environment variables:
//   - SCIM_VERBOSE: Enable verbose logging (true/false/1/0)
//   - SCIM_DESTRUCTIVE: Deletion behavior (-1=safe mode, 0=partial, >0=full)
func LoadScimParametersFromEnv() (ka *ScimEndpointParameters, gcp *GoogleEndpointParameters, err error) {
	// Load Google credentials
	var credentials []byte
	credentialsStr := os.Getenv("GOOGLE_CREDENTIALS")
	if len(credentialsStr) == 0 {
		err = errors.New("environment variable \"GOOGLE_CREDENTIALS\" is not set")
		return
	}

	// Try to decode as base64 first, if that fails, use as-is
	if decoded, err2 := base64.StdEncoding.DecodeString(credentialsStr); err2 == nil {
		credentials = decoded
	} else {
		// If not base64, assume it's the raw JSON
		credentials = []byte(credentialsStr)
	}

	// Validate that credentials look like JSON
	credStr := strings.TrimSpace(string(credentials))
	if !strings.HasPrefix(credStr, "{") {
		err = errors.New("GOOGLE_CREDENTIALS does not appear to be valid JSON")
		return
	}

	// Load Google admin account
	adminAccount := os.Getenv("GOOGLE_ADMIN_ACCOUNT")
	if len(adminAccount) == 0 {
		err = errors.New("environment variable \"GOOGLE_ADMIN_ACCOUNT\" is not set")
		return
	}

	// Load SCIM groups
	scimGroupsStr := os.Getenv("SCIM_GROUPS")
	if len(scimGroupsStr) == 0 {
		err = errors.New("environment variable \"SCIM_GROUPS\" is not set")
		return
	}
	scimGroups := parseScimGroupsFromString(scimGroupsStr)
	if len(scimGroups) == 0 {
		err = errors.New("\"SCIM_GROUPS\" environment variable does not contain any valid groups")
		return
	}

	// Load SCIM URL
	scimUrl := os.Getenv("SCIM_URL")
	if len(scimUrl) == 0 {
		err = errors.New("environment variable \"SCIM_URL\" is not set")
		return
	}

	// Load SCIM token
	scimToken := os.Getenv("SCIM_TOKEN")
	if len(scimToken) == 0 {
		err = errors.New("environment variable \"SCIM_TOKEN\" is not set")
		return
	}

	// Build Google endpoint parameters
	gcp = &GoogleEndpointParameters{
		AdminAccount: adminAccount,
		Credentials:  credentials,
		ScimGroups:   scimGroups,
	}

	// Build SCIM endpoint parameters
	ka = &ScimEndpointParameters{
		Url:   scimUrl,
		Token: scimToken,
	}

	// Load optional verbose flag
	if verboseStr := os.Getenv("SCIM_VERBOSE"); len(verboseStr) > 0 {
		if bv, ok := toBoolean(verboseStr); ok {
			ka.Verbose = bv
		}
	}

	// Load optional destructive flag
	if destructiveStr := os.Getenv("SCIM_DESTRUCTIVE"); len(destructiveStr) > 0 {
		if iv, err2 := strconv.Atoi(destructiveStr); err2 == nil {
			ka.Destructive = int32(iv)
		} else {
			ka.Destructive = -1
		}
	}

	return
}

// parseScimGroupsFromString parses a comma or newline separated list of groups
func parseScimGroupsFromString(groupsStr string) []string {
	var groups []string
	groupsStr = strings.TrimSpace(groupsStr)

	// Split by newlines first
	lines := strings.Split(groupsStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		// Split each line by comma
		parts := strings.Split(line, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if len(part) > 0 {
				groups = append(groups, part)
			}
		}
	}
	return groups
}

// IsEnvConfigAvailable checks if the required environment variables for
// environment-based configuration are present.
func IsEnvConfigAvailable() bool {
	requiredVars := []string{
		"GOOGLE_CREDENTIALS",
		"GOOGLE_ADMIN_ACCOUNT",
		"SCIM_GROUPS",
		"SCIM_URL",
		"SCIM_TOKEN",
	}
	for _, varName := range requiredVars {
		if len(os.Getenv(varName)) == 0 {
			return false
		}
	}
	return true
}

// GetConfigSourceDescription returns a description of which configuration
// source will be used based on available environment variables.
func GetConfigSourceDescription() string {
	if IsEnvConfigAvailable() {
		return "Using environment variable configuration"
	}
	if len(os.Getenv("KSM_CONFIG_BASE64")) > 0 {
		return "Using Keeper Secrets Manager configuration"
	}
	return "No valid configuration source found"
}
