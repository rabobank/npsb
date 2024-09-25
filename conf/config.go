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

	"github.com/rabobank/npsb/model"
)

var (
	//  NPSB  :  Network Policy Service Broker
	debugStr   = os.Getenv("DEBUG")
	Debug      = false
	CredhubURL = os.Getenv("CREDHUB_URL")

	Catalog    model.Catalog
	ListenPort int

	ClientId             = os.Getenv("CLIENT_ID")
	ClientSecret         = os.Getenv("CLIENT_SECRET")
	BrokerUser           = os.Getenv("BROKER_USER")
	BrokerPassword       = os.Getenv("BROKER_PASSWORD")
	CatalogDir           = os.Getenv("CATALOG_DIR")
	ListenPortStr        = os.Getenv("LISTEN_PORT")
	CfApiURL             = os.Getenv("CFAPI_URL")
	UaaApiURL            = os.Getenv("UAA_URL")
	SkipSslValidationStr = os.Getenv("SKIP_SSL_VALIDATION")
	SkipSslValidation    bool
	//CredsPath            = os.Getenv("CREDS_PATH") // something like /brokers/npsb/credentials

	CfClient      *client.Client
	CfConfig      *config.Config
	CfCtx         = context.Background()
	AllLabelNames = []string{LabelNameType, LabelNameName, LabelNameScope, LabelNameSource, LabelNamePort, LabelNameProtocol}
)

const (
	BasicAuthRealm        = "NPSB Network Policy Service Broker"
	LabelNameType         = "npsb.type"
	LabelValueTypeSrc     = "source"
	LabelValueTypeDest    = "destination"
	LabelNameName         = "npsb.source.name"
	AnnotationNameDesc    = "npsb.source.description"
	LabelNameScope        = "npsb.source.scope"
	LabelValueScopeLocal  = "local"
	LabelValueScopeGlobal = "global"
	LabelNameSource       = "npsb.dest.source"
	LabelNamePort         = "npsb.dest.port"
	LabelNameProtocol     = "npsb.dest.protocol"
	LabelValueProtocolTCP = "tcp"
	LabelValueProtocolUDP = "udp"
	ActionBind            = "create"
	ActionUnbind          = "delete"
)

// EnvironmentComplete - Check for required environment variables and exit if not all are there.
func EnvironmentComplete() {
	envComplete := true
	if debugStr == "true" {
		Debug = true
	}
	if CredhubURL == "" {
		CredhubURL = "https://credhub.service.cf.internal:8844"
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

	app, e := cfenv.Current()
	if e != nil {
		fmt.Printf("Not running in a CF environment")
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
		UaaApiURL = strings.Replace(CfApiURL, "api", "uaa", 1)
		fmt.Println("UAA endpoint:", UaaApiURL)
	}

	if strings.EqualFold(SkipSslValidationStr, "true") {
		SkipSslValidation = true
	}

	// try to get the uaa credentials from credhub
	type VcapService struct {
		Credentials struct {
			ClientID       string `json:"clientId,omitempty"`
			ClientSecret   string `json:"clientSecret,omitempty"`
			BrokerUser     string `json:"brokerUser,omitempty"`
			BrokerPassword string `json:"brokerPassword,omitempty"`
		} `json:"credentials"`
		InstanceName string `json:"instance_name"`
	}

	type VcapServices struct {
		Credhub []VcapService `json:"credhub"`
	}

	vcapServicesString := os.Getenv("VCAP_SERVICES")
	if vcapServicesString != "" {
		vcapServices := VcapServices{}
		if err := json.Unmarshal([]byte(vcapServicesString), &vcapServices); err != nil {
			fmt.Printf("could not get npsb-credentials from credhub, error: %s\n", err)
		} else {
			for _, service := range vcapServices.Credhub {
				if service.InstanceName == "npsb-credentials" {
					ClientId = service.Credentials.ClientID
					ClientSecret = service.Credentials.ClientSecret
					BrokerUser = service.Credentials.BrokerUser
					BrokerPassword = service.Credentials.BrokerPassword
					if Debug {
						fmt.Println("got npsb-credentials from credhub")
					}
				}
			}
		}
	}
	if ClientId == "" {
		fmt.Println("missing envvar : CLIENT_ID")
		envComplete = false
	}
	if ClientSecret == "" {
		fmt.Println("missing envvar : CLIENT_SECRET")
		envComplete = false
	}
	if BrokerUser == "" {
		fmt.Println("missing envvar : BROKER_USER")
		envComplete = false
	}
	if BrokerPassword == "" {
		fmt.Println("missing envvar : BROKER_PASSWORD")
		envComplete = false
	}

	if !envComplete {
		fmt.Println("one or more required environment variables missing, aborting...")
		os.Exit(8)
	}
}
