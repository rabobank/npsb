package conf

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/config"
	"os"
	"strconv"
	"strings"

	"github.com/rabobank/npsb/httpHelper"
	"github.com/rabobank/npsb/model"
)

var (
	//  NPSB  :  Network Policy Service Broker
	debugStr           = os.Getenv("DEBUG")
	Debug              = false
	httpTimeoutStr     = os.Getenv("HTTP_TIMEOUT")
	HttpTimeoutDefault = 10
	ClientId           = os.Getenv("CLIENT_ID")
	ClientSecret       string // will be resolved from config in credhub path CredsPath
	CredhubURL         = os.Getenv("CREDHUB_URL")

	Catalog    model.Catalog
	ListenPort int

	BrokerUser           = os.Getenv("BROKER_USER")
	BrokerPassword       string // will be resolved from config in credhub path CredsPath
	CatalogDir           = os.Getenv("CATALOG_DIR")
	ListenPortStr        = os.Getenv("LISTEN_PORT")
	CfApiURL             = os.Getenv("CFAPI_URL")
	UaaApiURL            = os.Getenv("UAA_URL")
	SkipSslValidationStr = os.Getenv("SKIP_SSL_VALIDATION")
	SkipSslValidation    bool
	CredsPath            = os.Getenv("CREDS_PATH") // something like /brokers/npsb/credentials

	CfClient      *client.Client
	CfConfig      *config.Config
	CfCtx         = context.Background()
	AllLabelNames = []string{"rabobank.com/npsb.type", "rabobank.com/npbs.source.name", "rabobank.com/npsb.source.description", "rabobank.com/npsb.source.scope", "rabobank.com/npsb.target.source"}
)

const (
	BasicAuthRealm  = "NPSB Network Policy Service Broker"
	LabelNameType   = "rabobank.com/npsb.type"
	LabelNameName   = "rabobank.com/npbs.source.name"
	LabelNameDesc   = "rabobank.com/npsb.source.description"
	LabelNameScope  = "rabobank.com/npsb.source.scope"
	LabelNameSource = "rabobank.com/npsb.target.source"
)

// EnvironmentComplete - Check for required environment variables and exit if not all are there.
func EnvironmentComplete() {
	app, e := cfenv.Current()
	if e != nil {
		fmt.Printf("Not running in a CF environment")
	}

	envComplete := true
	if debugStr == "true" {
		Debug = true
	}
	if CredhubURL == "" {
		CredhubURL = "https://credhub.service.cf.internal:8844"
	}
	if ClientId == "" {
		envComplete = false
		fmt.Println("missing envvar: CLIENT_ID")
	}
	if BrokerUser == "" {
		envComplete = false
		fmt.Println("missing envvar: BROKER_USER")
	}
	if CatalogDir == "" {
		CatalogDir = "./catalog"
	}
	if ListenPortStr == "" {
		ListenPort = 8080
	} else {
		var err error
		ListenPort, err = strconv.Atoi(ListenPortStr)
		if err != nil {
			fmt.Printf("failed reading envvar LISTEN_PORT, err: %s\n", err)
			envComplete = false
		}
	}
	if CfApiURL == "" {
		if app != nil {
			fmt.Printf("CF API Url not provided, defaulting to cf environment url : %s\n", app.CFAPI)
			CfApiURL = app.CFAPI
			fmt.Println("CF API endpoint:", CfApiURL)
		} else {
			envComplete = false
			fmt.Println("missing envvar: CFAPI_URL")
		}
	}

	if UaaApiURL == "" {
		fmt.Println("UAA url not provided. Inferring it from CF API")
		if content, e := httpHelper.Request(CfApiURL).Accepting("application/json").Get(); e != nil {
			fmt.Println("Unable to get CF API endpoints:", e)
			envComplete = false
		} else {
			var endpoints model.CfApiEndpoints
			if e = json.Unmarshal(content, &endpoints); e != nil {
				fmt.Println("Unable to unmarshal CF API endpoints:", e)
				envComplete = false
			} else {
				UaaApiURL = endpoints.Links.Uaa.Href
				fmt.Println("UAA endpoint:", UaaApiURL)
			}
		}
	}

	if strings.EqualFold(SkipSslValidationStr, "true") {
		SkipSslValidation = true
	}

	if !envComplete {
		fmt.Println("one or more required environment variables missing, aborting...")
		os.Exit(8)
	}
}
