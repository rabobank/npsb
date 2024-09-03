package controllers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rabobank/npsb/model"
	"github.com/rabobank/npsb/util"
)

type Credentials string

func (c Credentials) ServiceInstanceId(serviceInstanceId string) string {
	return fmt.Sprintf(string(c), serviceInstanceId)
}

const credentialsPath = Credentials("/pcsb/%s/credentials")

func CreateServiceBinding(w http.ResponseWriter, r *http.Request) {
	var err error
	serviceInstanceId := mux.Vars(r)["service_instance_guid"]
	serviceBindingId := mux.Vars(r)["service_binding_guid"]
	var serviceBinding model.ServiceBinding
	err = util.ProvisionObjectFromRequest(r, &serviceBinding)
	if err != nil {
		util.WriteHttpResponse(w, http.StatusBadRequest, model.BrokerError{Error: "FAILED", Description: err.Error(), InstanceUsable: false, UpdateRepeatable: false})
		return
	}
	credhubPath := credentialsPath.ServiceInstanceId(serviceInstanceId)

	fmt.Printf("create service binding id %s for service instance id %s, creating path %s...\n", serviceBindingId, serviceInstanceId, credhubPath)
	response := model.CreateServiceBindingResponse{Credentials: &model.Credentials{CredhubRef: credhubPath}}
	util.WriteHttpResponse(w, http.StatusCreated, response)
}

func DeleteServiceBinding(w http.ResponseWriter, r *http.Request) {
	serviceInstanceId := mux.Vars(r)["service_instance_guid"]
	serviceBindingId := mux.Vars(r)["service_binding_guid"]
	fmt.Printf("delete service binding id %s for service instance id %s...\n", serviceBindingId, serviceInstanceId)
	util.WriteHttpResponse(w, http.StatusOK, model.DeleteServiceBindingResponse{Result: "unbind completed"})
}
