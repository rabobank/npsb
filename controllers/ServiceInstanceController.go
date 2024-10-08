package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"net/http"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	"github.com/rabobank/npsb/conf"
	"github.com/rabobank/npsb/model"
	"github.com/rabobank/npsb/util"
)

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
	labels[conf.LabelNameScope] = &serviceInstanceParms.Scope
	labels[conf.LabelNameSource] = &serviceInstanceParms.Source
	annotations := make(map[string]*string)
	annotations[conf.AnnotationNameDesc] = &serviceInstanceParms.Description

	serviceInstanceUpdate := resource.ServiceInstanceManagedUpdate{Metadata: &resource.Metadata{Labels: labels, Annotations: annotations}}

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
	util.WriteHttpResponse(w, http.StatusCreated, model.CreateServiceInstanceResponse{})
	return
}

func DeleteServiceInstance(w http.ResponseWriter, r *http.Request) {
	_ = r // prevent compiler warning
	util.WriteHttpResponse(w, http.StatusOK, model.DeleteServiceInstanceResponse{})
}

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
		return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is missing", conf.LabelNameType)
	}
	if serviceInstanceParms.Type != conf.LabelValueTypeSrc && serviceInstanceParms.Type != conf.LabelValueTypeDest {
		return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is invalid, should be \"%s\" or \"%s\"", conf.LabelNameType, conf.LabelValueTypeSrc, conf.LabelValueTypeDest)
	}
	if serviceInstanceParms.Type == "source" {
		if serviceInstanceParms.Name == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is missing", conf.LabelNameName)
		}
		if !parameterValueRegex.MatchString(serviceInstanceParms.Name) {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is invalid, should match regex %s", conf.LabelNameName, parameterValueRegex.String())
		}
		if len(serviceInstanceParms.Description) > 128 {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is invalid, maximum length is 128, you have %d", conf.AnnotationNameDesc, len(serviceInstanceParms.Description))
		}
		if serviceInstanceParms.Scope == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is missing", conf.LabelNameScope)
		}
		if serviceInstanceParms.Scope == "" || (serviceInstanceParms.Scope != conf.LabelValueScopeGlobal && serviceInstanceParms.Scope != conf.LabelValueScopeLocal) {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is missing or invalid, should be \"%s\" or \"%s\"", conf.LabelNameScope, conf.LabelValueScopeGlobal, conf.LabelValueScopeLocal)
		}
		if instanceWithNameExists(serviceInstanceParms.Name) {
			return serviceInstanceParms, fmt.Errorf("a network-policies service with label \"%s\"=\"%s\" is already taken", conf.LabelNameName, serviceInstanceParms.Name)
		}
	}

	if serviceInstanceParms.Type == "destination" {
		if serviceInstanceParms.Source == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is missing", conf.LabelNameSource)
		}
		if !parameterValueRegex.MatchString(serviceInstanceParms.Source) {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is invalid, should match regex %s", conf.LabelNameSource, parameterValueRegex.String())
		}
	}
	return serviceInstanceParms, nil
}

// instanceWithNameExists checks if a service instance with the given name (and the network policies service name) already exists. If errors occur, we return true so the caller fails
func instanceWithNameExists(srcName string) bool {
	// get the plan first
	var servicePlanGuid string
	serviceName := conf.Catalog.Services[0].Name
	planName := conf.Catalog.Services[0].Plans[0].Name
	planListOptions := client.ServicePlanListOptions{ListOptions: &client.ListOptions{}, Names: client.Filter{Values: []string{planName}}, ServiceOfferingNames: client.Filter{Values: []string{serviceName}}}
	if plans, err := conf.CfClient.ServicePlans.ListAll(conf.CfCtx, &planListOptions); err != nil {
		fmt.Printf("failed to list service plans: %s\n", err)
		return true
	} else {
		if len(plans) == 0 {
			fmt.Printf("no service plan found service \"%s\" plan \"%s\"\n", serviceName, planName)
			return true
		}
		servicePlanGuid = plans[0].GUID
	}

	labelSelector := client.LabelSelector{}
	labelSelector.EqualTo(conf.LabelNameName, srcName)
	instanceListOptions := client.ServiceInstanceListOptions{ServicePlanGUIDs: client.Filter{Values: []string{servicePlanGuid}}, ListOptions: &client.ListOptions{LabelSel: labelSelector}}
	if instances, err := conf.CfClient.ServiceInstances.ListAll(conf.CfCtx, &instanceListOptions); err != nil {
		fmt.Printf("failed to list service instances with label %s=%s: %s\n", conf.LabelNameName, srcName, err)
		return false
	} else {
		if len(instances) > 0 {
			for _, instance := range instances {
				fmt.Printf("a service instance with label %s=%s already exists with name=%s, instance_guid=%s, space_guid=%s\n", conf.LabelNameName, srcName, instance.Name, instance.GUID, instance.Relationships.Space.Data.GUID)
			}
			return true
		}
	}
	return false
}
