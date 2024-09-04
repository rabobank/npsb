package model

type ServiceBinding struct {
	ServiceId         string                 `json:"service_id"`
	PlanId            string                 `json:"plan_id"`
	AppGuid           string                 `json:"app_guid"`
	ServiceInstanceId string                 `json:"service_instance_id"`
	BindResource      *BindResource          `json:"bind_resource"`
	Context           *Context               `json:"context"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
}

type BindResource struct {
	AppGuid   string `json:"app_guid"`
	SpaceGuid string `json:"space_guid"`
}

type CreateServiceBindingResponse struct {
	Result string `json:"result"`
}

type DeleteServiceBindingResponse struct {
	Result string `json:"result"`
}

type ServiceBindingParameters struct {
	Port string `json:"port"`
}
