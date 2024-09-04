package model

import "github.com/cloudfoundry/go-cfclient/v3/resource"

type ServiceInstance struct {
	ServiceId        string                 `json:"service_id"`
	PlanId           string                 `json:"plan_id"`
	OrganizationGuid string                 `json:"organization_guid"`
	SpaceGuid        string                 `json:"space_guid"`
	Context          *Context               `json:"context"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
}

type CreateServiceInstanceResponse struct {
	ServiceId    string     `json:"service_id"`
	PlanId       string     `json:"plan_id"`
	DashboardUrl string     `json:"dashboard_url"`
	Operation    *Operation `json:"operation,omitempty"`
	//Operation *Operation `json:"last_operation,omitempty"`
	Metadata *resource.Metadata `json:"metadata,omitempty"`
}

type DeleteServiceInstanceResponse struct {
	Result string `json:"result,omitempty"`
}

type ServiceInstanceParameters struct {
	Type        string `json:"type"`                  // source or destination
	Name        string `json:"name,omitempty"`        // only valid for type=source
	Description string `json:"description,omitempty"` // only valid for type=source
	Scope       string `json:"scope,omitempty"`       // only valid for type=source
	Source      string `json:"source,omitempty"`      // only valid for type=destination
}
