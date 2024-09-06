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

type Credentials string

func (c Credentials) ServiceInstanceId(serviceInstanceId string) string {
	return fmt.Sprintf(string(c), serviceInstanceId)
}

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

	//
	// update the service binding with the labels
	if *labels[conf.LabelNamePort] != "" {
		if _, err := conf.CfClient.ServiceCredentialBindings.Update(conf.CfCtx, serviceBindingGuid, &serviceBindingUpdate); err != nil {
			fmt.Printf("failed to update service binding %s: %s\n", serviceBindingGuid, err)
			util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("failed to update service binding %s: %s", serviceBindingGuid, err), InstanceUsable: false, UpdateRepeatable: false})
			return
		}
	}
	//
	//get the service instance first to see if it is a source or destination instance
	if serviceInstance, err := conf.CfClient.ServiceInstances.Get(conf.CfCtx, serviceInstanceGuid); err != nil {
		fmt.Printf("failed to get service instance %s: %s\n", serviceBinding.ServiceInstanceId, err)
		util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("failed to get service instance %s: %s", serviceBinding.ServiceInstanceId, err), InstanceUsable: false, UpdateRepeatable: false})
		return
	} else {
		if serviceInstance == nil || serviceInstance.Metadata == nil || serviceInstance.Metadata.Labels == nil {
			fmt.Printf("service instance (metadata.labels) for id %s not found\n", serviceBinding.ServiceInstanceId)
			util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("service instance (metadata.labels) for id %s not found", serviceBinding.ServiceInstanceId), InstanceUsable: false, UpdateRepeatable: false})
			return
		} else {
			var srcPolicies []model.NetworkPolicy
			var destPolicies []model.NetworkPolicy

			// get the policies for the source service instance
			if serviceInstance.Metadata.Labels[conf.LabelNameType] != nil && *serviceInstance.Metadata.Labels[conf.LabelNameType] == conf.LabelValueTypeSrc {
				if srcPolicies, err = policies4Source(*serviceInstance.Metadata.Labels[conf.LabelNameName], serviceBinding.AppGuid); err != nil {
					fmt.Printf("failed to get policies for source service instance id %s: %s\n", serviceInstanceGuid, err)
					util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("failed to create policies for source service instance id %s: %s", serviceInstanceGuid, err), InstanceUsable: false, UpdateRepeatable: false})
					return
				} else {
					if conf.Debug {
						fmt.Printf("we found %d source policies for label %s=%s\n", len(srcPolicies), conf.LabelNameSource, *serviceInstance.Metadata.Labels[conf.LabelNameName])
					}
					for ix, policy := range srcPolicies {
						fmt.Printf("policy %d created for source service instance id %s: %s\n", ix, serviceInstanceGuid, policy)
					}
				}
			}
			// get the policies for the destination service instance
			if serviceInstance.Metadata.Labels[conf.LabelNameType] != nil && *serviceInstance.Metadata.Labels[conf.LabelNameType] == conf.LabelValueTypeDest {
				if destPolicies, err = policies4Destination(*serviceInstance.Metadata.Labels[conf.LabelNameSource], serviceBinding.BindResource.AppGuid, serviceBindingParms.Port); err != nil {
					fmt.Printf("failed to get policies for destination service instance id %s: %s\n", serviceInstanceGuid, err)
					util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: fmt.Sprintf("failed to create policies for destination service instance id %s: %s", serviceInstanceGuid, err), InstanceUsable: false, UpdateRepeatable: false})
					return
				} else {
					for ix, policy := range destPolicies {
						fmt.Printf("policy %d created for destination service instance id %s: %s\n", ix, serviceInstanceGuid, policy)
					}
				}
			}
			util.WriteHttpResponse(w, http.StatusOK, model.CreateServiceBindingResponse{Result: fmt.Sprintf("bind completed, created %d policies", len(srcPolicies)+len(destPolicies))})
		}
	}
}

func DeleteServiceBinding(w http.ResponseWriter, r *http.Request) {
	serviceBindingGuid := mux.Vars(r)["service_binding_guid"]
	util.WriteHttpResponse(w, http.StatusOK, model.DeleteServiceBindingResponse{Result: fmt.Sprintf("unbind completed for binding %s\n", serviceBindingGuid)})
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
	if serviceBindingParms.Port <= minvalue || serviceBindingParms.Port >= maxValue {
		return serviceBindingParms, fmt.Errorf("parameter \"port\" is invalid, should be an integer between 1024 and 65535")
	}
	return serviceBindingParms, nil
}

func policies4Source(srcName string, srcAppGuid string) (policies []model.NetworkPolicy, err error) {
	policies = make([]model.NetworkPolicy, 0)
	// find all service instances with label source=srcName
	labelSelector := client.LabelSelector{}
	labelSelector.EqualTo(conf.LabelNameSource, fmt.Sprintf("%s", srcName))
	instanceListoption := client.ServiceInstanceListOptions{ListOptions: &client.ListOptions{LabelSel: labelSelector}}
	if instances, err := conf.CfClient.ServiceInstances.ListAll(conf.CfCtx, &instanceListoption); err != nil {
		fmt.Printf("failed to list service instance with label %s=%s: %s\n", conf.LabelNameSource, srcName, err)
		return nil, err
	} else {
		// can be multiple (many) instances
		if len(instances) < 1 {
			fmt.Printf("could not find any source service instance with label %s=%s\n", conf.LabelNameName, srcName)
		} else {
			serviceGUIDs := make([]string, 0)
			for _, instance := range instances {
				serviceGUIDs = append(serviceGUIDs, instance.GUID)
			}
			fmt.Printf("found %d source service instances with label %s=%s\n", len(serviceGUIDs), conf.LabelNameSource, srcName)
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
						if binding.Metadata.Labels[conf.LabelNamePort] != nil && *binding.Metadata.Labels[conf.LabelNamePort] != "" {
							destPort, _ = strconv.Atoi(*binding.Metadata.Labels[conf.LabelNamePort])
						}
						policy := model.NetworkPolicy{Source: binding.Relationships.App.Data.GUID, SourceName: util.Guid2AppName(srcAppGuid), Destination: binding.Relationships.App.Data.GUID, DestinationName: util.Guid2AppName(binding.Relationships.App.Data.GUID), Protocol: "TCP", Port: destPort}
						policies = append(policies, policy)
					}
				}
			}
		}
	}
	return policies, nil
}

func policies4Destination(srcName string, destAppGuid string, port int) (policies []model.NetworkPolicy, err error) {
	policies = make([]model.NetworkPolicy, 0)
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
						policy := model.NetworkPolicy{Source: binding.Relationships.App.Data.GUID, SourceName: util.Guid2AppName(binding.Relationships.App.Data.GUID), Destination: destAppGuid, DestinationName: util.Guid2AppName(destAppGuid), Protocol: "TCP", Port: destPort}
						policies = append(policies, policy)
					}
				}
			}
		}
	}
	return policies, nil
}
