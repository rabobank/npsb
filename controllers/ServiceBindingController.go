package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"github.com/gorilla/mux"
	"github.com/rabobank/npsb/conf"
	"github.com/rabobank/npsb/model"
	"github.com/rabobank/npsb/util"
	"net/http"
	"strconv"
)

type Credentials string

func (c Credentials) ServiceInstanceId(serviceInstanceId string) string {
	return fmt.Sprintf(string(c), serviceInstanceId)
}

func CreateServiceBinding(w http.ResponseWriter, r *http.Request) {
	var err error
	serviceBindingId := mux.Vars(r)["service_binding_guid"]
	var serviceBinding model.ServiceBinding
	err = util.ProvisionObjectFromRequest(r, &serviceBinding)
	if err != nil {
		util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: err.Error(), InstanceUsable: false, UpdateRepeatable: false})
		return
	}

	var serviceBindingParms model.ServiceBindingParameters
	if serviceBindingParms, err = validateBindingParameters(serviceBinding); err != nil {
		util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: err.Error(), InstanceUsable: false, UpdateRepeatable: false})
		return
	}

	labels := make(map[string]*string)
	labels[conf.LabelNamePort] = &serviceBindingParms.Port

	serviceBindingUpdate := resource.ServiceCredentialBindingUpdate{Metadata: &resource.Metadata{Labels: labels}}

	if sb, err := conf.CfClient.ServiceCredentialBindings.Update(conf.CfCtx, serviceBindingId, &serviceBindingUpdate); err != nil {
		fmt.Printf("failed to update service binding %s: %s\n", serviceBindingId, err)
	} else {
		labelsToPrint := ""
		for _, labelName := range conf.AllLabelNames {
			if labelValue, found := sb.Metadata.Labels[labelName]; found && labelValue != nil && *labelValue != "" {
				labelsToPrint = fmt.Sprintf("%s %s=%s", labelsToPrint, labelName, *labelValue)
			}
		}
		fmt.Printf("service binding %s updated with labels %s\n", serviceBindingId, labelsToPrint)
	}
	util.WriteHttpResponse(w, http.StatusOK, model.CreateServiceBindingResponse{Result: "bind completed"})
	return
}

func DeleteServiceBinding(w http.ResponseWriter, r *http.Request) {
	serviceInstanceId := mux.Vars(r)["service_instance_guid"]
	serviceBindingId := mux.Vars(r)["service_binding_guid"]
	fmt.Printf("delete service binding id %s for service instance id %s...\n", serviceBindingId, serviceInstanceId)
	util.WriteHttpResponse(w, http.StatusOK, model.DeleteServiceBindingResponse{Result: "unbind completed"})
}

func validateBindingParameters(serviceBinding model.ServiceBinding) (serviceBindingParms model.ServiceBindingParameters, err error) {
	minvalue := 1024
	maxValue := 65535
	if serviceBinding.Parameters == nil {
		return serviceBindingParms, nil
	}
	body, _ := json.Marshal(serviceBinding.Parameters)
	if err = json.Unmarshal(body, &serviceBindingParms); err != nil {
		return serviceBindingParms, fmt.Errorf("failed to unmarshal parameters: %s", err)
	}
	if serviceBindingParms.Port != "" {
		if i, err := strconv.Atoi(serviceBindingParms.Port); err != nil {
			return serviceBindingParms, fmt.Errorf("parameter \"port\" is invalid, should be an integer between 1024 and 65535")
		} else if i <= minvalue || i >= maxValue {
			return serviceBindingParms, fmt.Errorf("parameter \"port\" is invalid, should be an integer between 1024 and 65535")
		}
	}
	return serviceBindingParms, nil
}
