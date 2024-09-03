package model

import "time"

// Context The context inside the ServiceInstance and ServiceBinding request
type Context struct {
	Platform     string `json:"platform"`
	OrgName      string `json:"organization_name"`
	SpaceName    string `json:"space_name"`
	InstanceName string `json:"instance_name"`
}

type Operation struct {
	State       string `json:"state"`
	Description string `json:"description"`
}

type Catalog struct {
	Services []Service `json:"services"`
}

type Service struct {
	Name            string        `json:"name"`
	Id              string        `json:"id"`
	Description     string        `json:"description"`
	Bindable        bool          `json:"bindable"`
	MaxPollInterval int           `json:"maximum_polling_duration"`
	PlanUpdateable  bool          `json:"plan_updateable,omitempty"`
	Tags            []string      `json:"tags,omitempty"`
	Requires        []string      `json:"requires,omitempty"`
	Metadata        interface{}   `json:"metadata,omitempty"`
	Plans           []ServicePlan `json:"plans"`
	DashboardClient interface{}   `json:"dashboard_client"`
}

type ServicePlan struct {
	Name        string      `json:"name"`
	Id          string      `json:"id"`
	Description string      `json:"description"`
	Metadata    interface{} `json:"metadata,omitempty"`
	Free        bool        `json:"free,omitempty"`
}

type BrokerError struct {
	Error            string `json:"error"`
	Description      string `json:"description"`
	InstanceUsable   bool   `json:"instance_usable"`
	UpdateRepeatable bool   `json:"update_repeatable"`
}

type JSON map[string]interface{}

// CredhubCredentials - The structure that is returned when querying credhub for the npsb credentials (containing the broker password and the npsb cf client secret)
type CredhubCredentials struct {
	Data []struct {
		Type             string      `json:"type"`
		VersionCreatedAt time.Time   `json:"version_created_at"`
		ID               string      `json:"id"`
		Name             string      `json:"name"`
		Metadata         interface{} `json:"metadata"`
		Value            struct {
			CsbBrokerPassword string `json:"BROKER_PASSWORD"`
			CsbClientSecret   string `json:"CLIENT_SECRET"`
		} `json:"value"`
	} `json:"data"`
}

//const (
//	StatusSucceeded  = "succeeded"
//	StatusFailed     = "failed"
//	StatusInProgress = "in progress"
//)

//type LastOperation struct {
//	State       string `json:"state"`
//	Description string `json:"description"`
//}
