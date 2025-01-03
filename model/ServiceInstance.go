package model

import "github.com/cloudfoundry/go-cfclient/v3/resource"

type ServiceInstance struct {
	ServiceId  string                 `json:"service_id"`
	PlanId     string                 `json:"plan_id"`
	Context    InstanceContext        `json:"context,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type InstanceContext struct {
	Platform         string `json:"platform,omitempty"`
	OrganizationGuid string `json:"organization_guid,omitempty"`
	SpaceGuid        string `json:"space_guid,omitempty"`
	OrganizationName string `json:"organization_name,omitempty"`
	SpaceName        string `json:"space_name,omitempty"`
	InstanceName     string `json:"instance_name,omitempty"`
}
type CreateServiceInstanceResponse struct {
	ServiceId string             `json:"service_id"`
	PlanId    string             `json:"plan_id"`
	Metadata  *resource.Metadata `json:"metadata,omitempty"`
}

type DeleteServiceInstanceResponse struct {
	Result string `json:"result,omitempty"`
}

type ServiceInstanceParameters struct {
	Type        string `json:"type"`                  // source or destination
	Name        string `json:"name,omitempty"`        // only valid for type=source
	Description string `json:"description,omitempty"` // only valid for type=source
	SourceName  string `json:"sourceName,omitempty"`  // only valid for type=destination
	SourceSpace string `json:"sourceSpace,omitempty"` // only valid for type=destination
	SourceOrg   string `json:"sourceOrg,omitempty"`   // only valid for type=destination
}
