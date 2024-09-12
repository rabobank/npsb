package model

type ServiceBinding struct {
	AppGuid           string                 `json:"app_guid"`
	ServiceInstanceId string                 `json:"service_instance_id"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
}

type CreateServiceBindingResponse struct {
	Result string `json:"result"`
}

type DeleteServiceBindingResponse struct {
	Result string `json:"result"`
}

type ServiceBindingParameters struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}
