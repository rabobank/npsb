package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"github.com/gorilla/mux"
	"github.com/rabobank/npsb/conf"
	"github.com/rabobank/npsb/model"
	"github.com/rabobank/npsb/util"
	"net/http"
	"strconv"
)

const (
	ActionBind   = "create"
	ActionUnbind = "delete"
)

func CreateServiceBinding(w http.ResponseWriter, r *http.Request) {
	var err error
	serviceInstanceGuid := mux.Vars(r)["service_instance_guid"]
	serviceBindingGuid := mux.Vars(r)["service_binding_guid"]
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
	portStr := strconv.Itoa(serviceBindingParms.Port)
	labels[conf.LabelNamePort] = &portStr

	serviceBindingUpdate := resource.ServiceCredentialBindingUpdate{Metadata: &resource.Metadata{Labels: labels}}

	// update the service binding with the labels
	if *labels[conf.LabelNamePort] != "" {
		if _, err := conf.CfClient.ServiceCredentialBindings.Update(conf.CfCtx, serviceBindingGuid, &serviceBindingUpdate); err != nil {
			fmt.Printf("failed to update service binding %s: %s\n", serviceBindingGuid, err)
			util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("failed to update service binding %s: %s", serviceBindingGuid, err), InstanceUsable: false, UpdateRepeatable: false})
			return
		}
	}

	if serviceInstance, err := conf.CfClient.ServiceInstances.Get(conf.CfCtx, serviceInstanceGuid); err != nil {
		fmt.Printf("failed to get service instance %s: %s\n", serviceBinding.ServiceInstanceId, err)
		util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("failed to get service instance %s: %s", serviceBinding.ServiceInstanceId, err), InstanceUsable: false, UpdateRepeatable: false})
	} else {
		if serviceInstance == nil || serviceInstance.Metadata == nil || serviceInstance.Metadata.Labels == nil {
			fmt.Printf("service instance (metadata.labels) for id %s not found\n", serviceBinding.ServiceInstanceId)
			util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("service instance (metadata.labels) for id %s not found", serviceBinding.ServiceInstanceId), InstanceUsable: false, UpdateRepeatable: false})
		} else {
			port, _ := strconv.Atoi(portStr)
			if numProcessedPolicies, err := createOrDeletePolicies(ActionBind, serviceInstance, serviceBinding.AppGuid, port); err != nil {
				util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("failed to create policies for service instance %s: %s", serviceBinding.ServiceInstanceId, err), InstanceUsable: false, UpdateRepeatable: false})
			} else {
				util.WriteHttpResponse(w, http.StatusOK, model.CreateServiceBindingResponse{Result: fmt.Sprintf("bind completed, created %d policies", numProcessedPolicies)})
			}
		}
	}
}

// DeleteServiceBinding - Deletes the service binding and the associated network policies
func DeleteServiceBinding(w http.ResponseWriter, r *http.Request) {
	serviceInstanceGuid := mux.Vars(r)["service_instance_guid"]
	serviceBindingGuid := mux.Vars(r)["service_binding_guid"]

	if serviceCredentialBinding, err := conf.CfClient.ServiceCredentialBindings.Get(conf.CfCtx, serviceBindingGuid); err != nil {
		util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("failed to get service binding %s: %s", serviceBindingGuid, err), InstanceUsable: false, UpdateRepeatable: false})
	} else {
		if serviceInstance, err := conf.CfClient.ServiceInstances.Get(conf.CfCtx, serviceInstanceGuid); err != nil {
			fmt.Printf("failed to get service instance %s: %s\n", serviceCredentialBinding.Relationships.ServiceInstance.Data.GUID, err)
			util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("failed to get service instance %s: %s", serviceCredentialBinding.Relationships.ServiceInstance.Data.GUID, err), InstanceUsable: false, UpdateRepeatable: false})
		} else {
			if serviceInstance == nil || serviceInstance.Metadata == nil || serviceInstance.Metadata.Labels == nil {
				fmt.Printf("service instance (metadata.labels) for id %s not found\n", serviceCredentialBinding.Relationships.ServiceInstance.Data.GUID)
				util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("service instance (metadata.labels) for id %s not found", serviceCredentialBinding.Relationships.ServiceInstance.Data.GUID), InstanceUsable: false, UpdateRepeatable: false})
			} else {
				port, _ := strconv.Atoi(*serviceCredentialBinding.Metadata.Labels[conf.LabelNamePort])
				if numProcessedPolicies, err := createOrDeletePolicies(ActionUnbind, serviceInstance, serviceCredentialBinding.Relationships.App.Data.GUID, port); err != nil {
					util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("failed to create policies for service instance %s: %s", serviceCredentialBinding.Relationships.ServiceInstance.Data.GUID, err), InstanceUsable: false, UpdateRepeatable: false})
				} else {
					util.WriteHttpResponse(w, http.StatusOK, model.CreateServiceBindingResponse{Result: fmt.Sprintf("bind completed, created %d policies", numProcessedPolicies)})
				}
			}
		}
	}
}

// createOrDeletePolicies - Creates or deletes (indicated by the action parameter) network policies for the given source or destination (determined by the presence of the name or source label) service instances,
//
//	returns the number of policies created or deleted and an optional error
func createOrDeletePolicies(action string, serviceInstance *resource.ServiceInstance, appGuid string, port int) (numProcessed int, err error) {
	var srcPolicies []model.NetworkPolicyLabels
	var destPolicies []model.NetworkPolicyLabels
	// get the policies for the source service instance
	if serviceInstance.Metadata.Labels[conf.LabelNameType] != nil && *serviceInstance.Metadata.Labels[conf.LabelNameType] == conf.LabelValueTypeSrc {
		if srcPolicies, err = policies4Source(*serviceInstance.Metadata.Labels[conf.LabelNameName], appGuid); err != nil {
			fmt.Printf("failed to get policies for source service instance id %s: %s\n", serviceInstance.GUID, err)
			return 0, err
		} else {
			for ix, policy := range srcPolicies {
				fmt.Printf("%s policy %d for source service instance id %s: %s\n", action, ix, serviceInstance.GUID, policy)
			}
		}
	}
	// get the policies for the destination service instance
	if serviceInstance.Metadata.Labels[conf.LabelNameType] != nil && *serviceInstance.Metadata.Labels[conf.LabelNameType] == conf.LabelValueTypeDest {
		if destPolicies, err = policies4Destination(*serviceInstance.Metadata.Labels[conf.LabelNameSource], appGuid, port); err != nil {
			fmt.Printf("failed to get policies for destination service instance id %s: %s\n", serviceInstance.GUID, err)
			return 0, err
		} else {
			for ix, policy := range destPolicies {
				fmt.Printf("%s policy %d for destination service instance id %s: %s\n", action, ix, serviceInstance.GUID, policy)
			}
		}
	}
	return
}

// validateBindingParameters - Validates the parameters of the service binding, returns the parameters or an error. The only allowed parameter is port, it must be between 1024 and 65535
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
	if serviceBindingParms.Port <= minvalue || serviceBindingParms.Port >= maxValue {
		return serviceBindingParms, fmt.Errorf("parameter \"port\" is invalid, should be an integer between 1024 and 65535")
	}
	return serviceBindingParms, nil
}

// policies4Source - Returns the policy labels for the source service instance with the given name and app guid for the app that is being bound. The destination service instance is identified by the label source=srcName
func policies4Source(srcName string, srcAppGuid string) (policyLabels []model.NetworkPolicyLabels, err error) {
	policyLabels = make([]model.NetworkPolicyLabels, 0)
	// find all service instances with label source=srcName
	labelSelector := client.LabelSelector{}
	labelSelector.EqualTo(conf.LabelNameSource, fmt.Sprintf("%s", srcName))
	instanceListOption := client.ServiceInstanceListOptions{ListOptions: &client.ListOptions{LabelSel: labelSelector}}
	if instances, err := conf.CfClient.ServiceInstances.ListAll(conf.CfCtx, &instanceListOption); err != nil {
		fmt.Printf("failed to list service instance with label %s=%s: %s\n", conf.LabelNameSource, srcName, err)
		return nil, err
	} else {
		// can be multiple (many) instances
		if len(instances) < 1 {
			if conf.Debug {
				fmt.Printf("could not find any source service instance with label %s=%s\n", conf.LabelNameName, srcName)
			}
		} else {
			serviceGUIDs := make([]string, 0)
			for _, instance := range instances {
				serviceGUIDs = append(serviceGUIDs, instance.GUID)
			}
			if conf.Debug {
				fmt.Printf("found %d source service instances with label %s=%s\n", len(serviceGUIDs), conf.LabelNameSource, srcName)
			}
			credBindingListOption := client.ServiceCredentialBindingListOptions{ServiceInstanceGUIDs: client.Filter{Values: serviceGUIDs}}
			if bindings, err := conf.CfClient.ServiceCredentialBindings.ListAll(conf.CfCtx, &credBindingListOption); err != nil {
				fmt.Printf("failed to list service bindings for source service instance %s: %s\n", instances[0].GUID, err)
				return nil, err
			} else {
				if len(bindings) < 1 {
					fmt.Printf("could not find any service bindings for %d source service instances with label %s:%s\n", len(serviceGUIDs), conf.LabelNameSource, srcName)
				} else {
					for _, binding := range bindings {
						destPort := 8080
						if binding.Metadata.Labels[conf.LabelNamePort] != nil && *binding.Metadata.Labels[conf.LabelNamePort] != "0" {
							destPort, _ = strconv.Atoi(*binding.Metadata.Labels[conf.LabelNamePort])
						}
						policy := model.NetworkPolicyLabels{Source: binding.Relationships.App.Data.GUID, SourceName: util.Guid2AppName(srcAppGuid), Destination: binding.Relationships.App.Data.GUID, DestinationName: util.Guid2AppName(binding.Relationships.App.Data.GUID), Protocol: "TCP", Port: destPort}
						policyLabels = append(policyLabels, policy)
					}
				}
			}
		}
	}
	return policyLabels, nil
}

// policies4Destination - Returns the policy labels for the destination service instance with the given name and app guid for the app that is being bound. The source service instance is identified by the label name=srcName
func policies4Destination(srcName string, destAppGuid string, port int) (policyLabels []model.NetworkPolicyLabels, err error) {
	policyLabels = make([]model.NetworkPolicyLabels, 0)
	// find all service instances with label name=srcName
	labelSelector := client.LabelSelector{}
	labelSelector.EqualTo(conf.LabelNameName, fmt.Sprintf("%s", srcName))
	instanceListoption := client.ServiceInstanceListOptions{ListOptions: &client.ListOptions{LabelSel: labelSelector}}
	if instances, err := conf.CfClient.ServiceInstances.ListAll(conf.CfCtx, &instanceListoption); err != nil {
		fmt.Printf("failed to list destination service instance with label %s=%s: %s\n", conf.LabelNameName, srcName, err)
		return nil, err
	} else {
		// this should always be a single instance
		if len(instances) < 1 {
			fmt.Printf("could not find any destination service instance with label %s=%s\n", conf.LabelNameName, srcName)
		} else {
			credBindingListOption := client.ServiceCredentialBindingListOptions{ServiceInstanceGUIDs: client.Filter{Values: []string{instances[0].GUID}}}
			if bindings, err := conf.CfClient.ServiceCredentialBindings.ListAll(conf.CfCtx, &credBindingListOption); err != nil {
				fmt.Printf("failed to list service bindings for destination service instance %s: %s\n", instances[0].GUID, err)
				return nil, err
			} else {
				if len(bindings) < 1 {
					fmt.Printf("could not find any service bindings for service instance %s with label %s:%s\n", instances[0].GUID, conf.LabelNameName, srcName)
				} else {
					for _, binding := range bindings {
						destPort := 8080
						if port != 0 {
							destPort = port
						}
						policy := model.NetworkPolicyLabels{Source: binding.Relationships.App.Data.GUID, SourceName: util.Guid2AppName(binding.Relationships.App.Data.GUID), Destination: destAppGuid, DestinationName: util.Guid2AppName(destAppGuid), Protocol: "TCP", Port: destPort}
						policyLabels = append(policyLabels, policy)
					}
				}
			}
		}
	}
	return policyLabels, nil
}
