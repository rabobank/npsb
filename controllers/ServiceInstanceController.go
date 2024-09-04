package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"net/http"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	"github.com/rabobank/npsb/conf"
	"github.com/rabobank/npsb/model"
	"github.com/rabobank/npsb/util"
)

//var instanceOperations = make(map[string]string) // key: serviceInstanceId, value: InProgress/Succeeded/Failed

func Catalog(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("get service broker catalog from %s...\n", r.RemoteAddr)
	util.WriteHttpResponse(w, http.StatusOK, conf.Catalog)
}

func CreateOrUpdateServiceInstance(w http.ResponseWriter, r *http.Request) {
	var err error
	serviceInstanceId := mux.Vars(r)["service_instance_guid"]
	var serviceInstance model.ServiceInstance
	err = util.ProvisionObjectFromRequest(r, &serviceInstance)
	if err != nil {
		util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: err.Error(), InstanceUsable: false, UpdateRepeatable: false})
		return
	}

	var serviceInstanceParms model.ServiceInstanceParameters
	if serviceInstanceParms, err = validateInstanceParameters(serviceInstance); err != nil {
		util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: err.Error(), InstanceUsable: false, UpdateRepeatable: false})
		return
	}

	labels := make(map[string]*string)
	labels[conf.LabelNameType] = &serviceInstanceParms.Type
	labels[conf.LabelNameName] = &serviceInstanceParms.Name
	labels[conf.LabelNameDesc] = &serviceInstanceParms.Description
	labels[conf.LabelNameScope] = &serviceInstanceParms.Scope
	labels[conf.LabelNameSource] = &serviceInstanceParms.Source
	metadata := resource.Metadata{Labels: labels}

	serviceInstanceUpdate := resource.ServiceInstanceManagedUpdate{Metadata: &resource.Metadata{Labels: labels}}

	go func() {
		time.Sleep(3 * time.Second)
		if _, si, err := conf.CfClient.ServiceInstances.UpdateManaged(conf.CfCtx, serviceInstanceId, &serviceInstanceUpdate); err != nil {
			fmt.Printf("failed to update service instance %s: %s\n", serviceInstanceId, err)
		} else {
			labelsToPrint := ""
			for _, labelName := range conf.AllLabelNames {
				if labelValue, found := si.Metadata.Labels[labelName]; found && labelValue != nil && *labelValue != "" {
					labelsToPrint = fmt.Sprintf("%s %s=%s", labelsToPrint, labelName, *labelValue)
				}
			}
			fmt.Printf("service instance %s (%s) updated with labels %s\n", serviceInstanceId, si.Name, labelsToPrint)
		}
	}()

	// If we respond with StatusAccepted, the CC will poll the last_operation endpoint, but the above routine cannot update the instance, it gets (CF-AsyncServiceInstanceOperationInProgress|60016):
	// So, we are cheating here and respond with StatusOk, and the CC will not poll the last_operation endpoint, and we take the small risk that the instance is not (properly) updated by the above routine.
	util.WriteHttpResponse(w, http.StatusOK, model.CreateServiceInstanceResponse{ServiceId: serviceInstance.ServiceId, PlanId: serviceInstance.PlanId, Metadata: &metadata})
	return
}

func DeleteServiceInstance(w http.ResponseWriter, r *http.Request) {
	serviceInstanceId := mux.Vars(r)["service_instance_guid"]
	response := model.DeleteServiceInstanceResponse{Result: fmt.Sprintf("Service instance %s deleted", serviceInstanceId)}
	util.WriteHttpResponse(w, http.StatusOK, response)
}

//
//func GetServiceInstanceLastOperation(w http.ResponseWriter, r *http.Request) {
//	serviceInstanceId := mux.Vars(r)["service_instance_guid"]
//	fmt.Printf("get service instance LastOperation for %s...\n", serviceInstanceId)
//	if operation, found := instanceOperations[serviceInstanceId]; !found {
//		response := &model.LastOperation{State: model.StatusFailed, Description: fmt.Sprintf("Service instance %s not found", serviceInstanceId)}
//		util.WriteHttpResponse(w, http.StatusOK, response)
//	} else {
//		if operation == model.StatusInProgress {
//			response := &model.LastOperation{State: operation, Description: fmt.Sprintf("Service instance %s is being processed", serviceInstanceId)}
//			util.WriteHttpResponse(w, http.StatusOK, response)
//		}
//		if operation == model.StatusSucceeded {
//			response := &model.LastOperation{State: operation, Description: fmt.Sprintf("Service instance %s has been successfully created", serviceInstanceId)}
//			util.WriteHttpResponse(w, http.StatusOK, response)
//		}
//	}
//}

func validateInstanceParameters(serviceInstance model.ServiceInstance) (serviceInstanceParms model.ServiceInstanceParameters, err error) {
	parameterValueRegex := regexp.MustCompile("^[a-zA-Z0-9._-]{1,64}$")
	if serviceInstance.Parameters == nil {
		return serviceInstanceParms, fmt.Errorf("parameters are missing")
	}
	body, _ := json.Marshal(serviceInstance.Parameters)
	if err := json.Unmarshal(body, &serviceInstanceParms); err != nil {
		return serviceInstanceParms, fmt.Errorf("failed to unmarshal parameters: %s", err)
	}
	if serviceInstanceParms.Type == "" {
		return serviceInstanceParms, fmt.Errorf("parameter \"type\" is missing")
	}
	if serviceInstanceParms.Type != "source" && serviceInstanceParms.Type != "target" {
		return serviceInstanceParms, fmt.Errorf("parameter \"type\" is invalid, should be \"source\" or \"target\"")
	}
	if serviceInstanceParms.Type == "source" {
		if serviceInstanceParms.Name == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"name\" is missing")
		}
		if !parameterValueRegex.MatchString(serviceInstanceParms.Name) {
			return serviceInstanceParms, fmt.Errorf("parameter \"name\" is invalid, should match regex %s", parameterValueRegex.String())
		}
		if serviceInstanceParms.Description == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"description\" is missing")
		}
		if !parameterValueRegex.MatchString(serviceInstanceParms.Description) {
			return serviceInstanceParms, fmt.Errorf("parameter \"description\" is invalid, should match regex %s", parameterValueRegex.String())
		}
		if serviceInstanceParms.Scope == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"scope\" is missing")
		}
		if serviceInstanceParms.Scope == "" || (serviceInstanceParms.Scope != "global" && serviceInstanceParms.Scope != "local") {
			return serviceInstanceParms, fmt.Errorf("parameter \"scope\" is missing or invalid, should be \"global\" or \"local\"")
		}
	}
	if serviceInstanceParms.Type == "target" {
		if serviceInstanceParms.Source == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"source\" is missing")
		}
		if !parameterValueRegex.MatchString(serviceInstanceParms.Source) {
			return serviceInstanceParms, fmt.Errorf("parameter \"source\" is invalid, should match regex %s", parameterValueRegex.String())
		}
	}
	return serviceInstanceParms, nil
}
