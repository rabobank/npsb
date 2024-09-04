package model

type CfApiEndpoints struct {
	Links struct {
		Uaa struct {
			Href string `json:"href"`
		} `json:"uaa"`
	} `json:"links"`
}

type CfServiceInstancePermissions struct {
	Read   bool `json:"read"`
	Manage bool `json:"manage"`
}
