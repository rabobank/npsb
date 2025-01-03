package controllers

import (
	"encoding/json"
	"errors"
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
	if serviceInstanceParms.Type == conf.LabelValueTypeSrc {
		labels[conf.LabelNameName] = &serviceInstanceParms.Name
	} else {
		labels[conf.LabelNameSourceName] = &serviceInstanceParms.SourceName
		labels[conf.LabelNameSourceSpace] = &serviceInstanceParms.SourceSpace
		labels[conf.LabelNameSourceOrg] = &serviceInstanceParms.SourceOrg
	}
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
	util.WriteHttpResponse(w, http.StatusCreated, model.CreateServiceInstanceResponse{ServiceId: serviceInstance.ServiceId, PlanId: serviceInstance.PlanId})
	return
}

func DeleteServiceInstance(w http.ResponseWriter, r *http.Request) {
	_ = r // prevent compiler warning
	util.WriteHttpResponse(w, http.StatusOK, model.DeleteServiceInstanceResponse{})
}

func validateInstanceParameters(serviceInstance model.ServiceInstance) (serviceInstanceParms model.ServiceInstanceParameters, err error) {
	parameterValueRegex := regexp.MustCompile("^[a-zA-Z0-9._-]{1,64}$")
	const (
		ParmType     = "type"
		ParmName     = "name"
		ParmDesc     = "description"
		ParmSrcName  = "sourceName"
		ParmSrcSpace = "sourceSpace"
		ParmSrcOrg   = "sourceOrg"
	)

	if serviceInstance.Parameters == nil {
		return serviceInstanceParms, fmt.Errorf("parameters are missing")
	}
	body, _ := json.Marshal(serviceInstance.Parameters)
	if err = json.Unmarshal(body, &serviceInstanceParms); err != nil {
		return serviceInstanceParms, fmt.Errorf("failed to unmarshal parameters: %s", err)
	}
	if serviceInstanceParms.Type == "" {
		return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is missing", ParmType)
	}
	if serviceInstanceParms.Type != conf.LabelValueTypeSrc && serviceInstanceParms.Type != conf.LabelValueTypeDest {
		return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is invalid, should be \"%s\" or \"%s\"", ParmType, conf.LabelValueTypeSrc, conf.LabelValueTypeDest)
	}
	if serviceInstanceParms.Type == "source" {
		if serviceInstanceParms.Name == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is missing", ParmName)
		}
		if !parameterValueRegex.MatchString(serviceInstanceParms.Name) {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is invalid, should match regex %s", ParmName, parameterValueRegex.String())
		}
		if len(serviceInstanceParms.Description) > 128 {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is invalid, maximum length is 128, you have %d", ParmDesc, len(serviceInstanceParms.Description))
		}
		if instanceWithNameExists(serviceInstanceParms.Name, serviceInstance) {
			return serviceInstanceParms, fmt.Errorf("a network-policies service with label \"%s\"=\"%s\" is already taken", conf.LabelNameName, serviceInstanceParms.Name)
		}
	}

	if serviceInstanceParms.Type == "destination" {
		if serviceInstanceParms.SourceName == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is missing", ParmSrcName)
		}
		if !parameterValueRegex.MatchString(serviceInstanceParms.SourceName) {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is invalid, should match regex %s", ParmSrcName, parameterValueRegex.String())
		}
		if serviceInstanceParms.SourceSpace == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is missing", ParmSrcSpace)
		}
		if !parameterValueRegex.MatchString(serviceInstanceParms.SourceSpace) {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is invalid, should match regex %s", ParmSrcSpace, parameterValueRegex.String())
		}
		if serviceInstanceParms.SourceOrg == "" {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is missing", ParmSrcOrg)
		}
		if !parameterValueRegex.MatchString(serviceInstanceParms.SourceOrg) {
			return serviceInstanceParms, fmt.Errorf("parameter \"%s\" is invalid, should match regex %s", ParmSrcOrg, parameterValueRegex.String())
		}
		if serviceInstanceParms.SourceSpace == serviceInstance.Context.SpaceName && serviceInstanceParms.SourceOrg == serviceInstance.Context.OrganizationName {
			return serviceInstanceParms, fmt.Errorf("you cannot use a source that is in the same org/space (%s/%s) as the target, for those cases use the standard \"cf add-network-policy\" commands", serviceInstanceParms.SourceOrg, serviceInstanceParms.SourceSpace)
		}

		// check if the source org/space exists:
		var orgGuid string
		orgListOptions := client.OrganizationListOptions{Names: client.Filter{Values: []string{serviceInstanceParms.SourceOrg}}}
		if org, err := conf.CfClient.Organizations.Single(conf.CfCtx, &orgListOptions); err != nil {
			errorMsg := fmt.Sprintf("failed to get org with name %s: %s\n", serviceInstanceParms.SourceOrg, err)
			fmt.Println(errorMsg)
			return serviceInstanceParms, errors.New(errorMsg)
		} else {
			orgGuid = org.GUID
		}
		spaceListOptions := client.SpaceListOptions{Names: client.Filter{Values: []string{serviceInstanceParms.SourceSpace}}, OrganizationGUIDs: client.Filter{Values: []string{orgGuid}}}
		if _, err = conf.CfClient.Spaces.Single(conf.CfCtx, &spaceListOptions); err != nil {
			errorMsg := fmt.Sprintf("failed to get space with name %s in org with name %s: %s\n", serviceInstanceParms.SourceSpace, serviceInstanceParms.SourceOrg, err)
			fmt.Println(errorMsg)
			return serviceInstanceParms, errors.New(errorMsg)
		}
	}
	return serviceInstanceParms, nil
}

// instanceWithNameExists checks if a service instance with the given "Name" label (and the network policies service name) in the current space already exists. If errors occur, we return true so the caller fails
func instanceWithNameExists(instanceLabelName string, serviceInstance model.ServiceInstance) bool {
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

	instanceListOptions := client.ServiceInstanceListOptions{ServicePlanGUIDs: client.Filter{Values: []string{servicePlanGuid}}, SpaceGUIDs: client.Filter{Values: []string{serviceInstance.Context.SpaceGuid}}}
	if spaceInstances, err := conf.CfClient.ServiceInstances.ListAll(conf.CfCtx, &instanceListOptions); err != nil {
		fmt.Printf("failed to list service instances in space %s: %s\n", serviceInstance.Context.SpaceName, err)
		return false
	} else {
		if len(spaceInstances) > 0 {
			for _, spaceInstance := range spaceInstances {
				if spaceInstance.Metadata.Labels[conf.LabelNameName] != nil && *spaceInstance.Metadata.Labels[conf.LabelNameName] == instanceLabelName {
					fmt.Printf("a service instance with label %s=%s already exists with name=%s, instance_guid=%s\n", conf.LabelNameName, instanceLabelName, spaceInstance.Name, spaceInstance.GUID)
					return true
				}
			}
		}
	}
	return false
}
