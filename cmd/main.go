package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	ksm "github.com/keeper-security/secrets-manager-go/core"
	"keepersecurity.com/ksm-scim/scim"
)

func main() {
	var err error
	var ka *scim.ScimEndpointParameters
	var gcp *scim.GoogleEndpointParameters

	// Check if environment variable configuration is available
	if scim.IsEnvConfigAvailable() {
		log.Println("Loading configuration from environment variables")
		if ka, gcp, err = scim.LoadScimParametersFromEnv(); err != nil {
			log.Fatal(err)
		}
	} else {
		// Fall back to KSM configuration from file
		log.Println("Loading configuration from Keeper Secrets Manager (config.base64)")
		var filePath = "config.base64"
		if _, err = os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
			var homeDir string
			if homeDir, err = os.UserHomeDir(); err != nil {
				log.Fatal(err)
			}
			filePath = path.Join(homeDir, filePath)
		}
		var data []byte
		if data, err = os.ReadFile(filePath); err != nil {
			log.Fatal(err)
		}
		var config = ksm.NewMemoryKeyValueStorage(string(data))
		var sm = ksm.NewSecretsManager(&ksm.ClientOptions{
			Config: config,
		})
		var filter []string
		if len(os.Args) == 2 {
			filter = append(filter, os.Args[1])
		}

		var records []*ksm.Record
		if records, err = sm.GetSecrets(filter); err != nil {
			log.Fatal(err)
		}

		var scimRecord *ksm.Record
		for _, r := range records {
			if r.Type() != "login" {
				continue
			}
			var webUrl = r.GetFieldValueByType("url")
			if len(webUrl) == 0 {
				continue
			}
			var uri *url.URL
			if uri, err = url.Parse(webUrl); err != nil {
				continue
			}
			if !strings.HasPrefix(uri.Path, "/api/rest/scim/v2/") {
				continue
			}
			var files = r.FindFiles("credentials.json")
			if len(files) == 0 {
				continue
			}
			scimRecord = r
			break
		}
		if scimRecord == nil {
			log.Fatal("SCIM record was not found. Make sure the record is valid and shared to KSM application")
		}

		if ka, gcp, err = scim.LoadScimParametersFromRecord(scimRecord); err != nil {
			log.Println(err)
			return
		}
	}

	var googleEndpoint = scim.NewGoogleEndpoint(gcp.Credentials, gcp.AdminAccount, gcp.ScimGroups)

	var sync = scim.NewScimSync(googleEndpoint, ka.Url, ka.Token)
	sync.SetVerbose(ka.Verbose)
	sync.SetUpdateUsers(ka.UpdateUsers)
	sync.SetDestructive(ka.Destructive)

	if ka.Verbose {
		googleEndpoint.TestConnection()
	}

	var syncStat *scim.SyncStat
	if syncStat, err = sync.Sync(); err != nil {
		log.Fatal(err.Error())
	}
	if len(syncStat.SuccessGroups) > 0 {
		fmt.Printf("Group Success:\n")
		for _, txt := range syncStat.SuccessGroups {
			fmt.Printf("\t%s\n", txt)
		}
	}
	if len(syncStat.FailedGroups) > 0 {
		fmt.Printf("Group Failure:\n")
		for _, txt := range syncStat.FailedGroups {
			fmt.Printf("\t%s\n", txt)
		}
	}
	if len(syncStat.SuccessUsers) > 0 {
		fmt.Printf("User Success:\n")
		for _, txt := range syncStat.SuccessUsers {
			fmt.Printf("\t%s\n", txt)
		}
	}
	if len(syncStat.FailedUsers) > 0 {
		fmt.Printf("User Failure:\n")
		for _, txt := range syncStat.FailedUsers {
			fmt.Printf("\t%s\n", txt)
		}
	}
	if len(syncStat.SuccessMembership) > 0 {
		fmt.Printf("Membership Success:\n")
		for _, txt := range syncStat.SuccessMembership {
			fmt.Printf("\t%s\n", txt)
		}
	}
	if len(syncStat.FailedMembership) > 0 {
		fmt.Printf("Membership Failure:\n")
		for _, txt := range syncStat.FailedMembership {
			fmt.Printf("\t%s\n", txt)
		}
	}
}
