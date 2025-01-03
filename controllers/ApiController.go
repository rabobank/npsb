package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/context"
	"github.com/rabobank/npsb/conf"
	"github.com/rabobank/npsb/model"
	"github.com/rabobank/npsb/util"
	"io"
	"net/http"
)

func GetSources(w http.ResponseWriter, r *http.Request) {
	if isValid, _, _ := ValidateRequest(w, r); isValid {
		sourcesList := model.SourcesResponseList{}
		// find all service instances with a "name" label
		labelSelector := client.LabelSelector{}
		labelSelector.EqualTo(conf.LabelNameType, conf.LabelValueTypeSrc)
		instanceListOption := client.ServiceInstanceListOptions{ListOptions: &client.ListOptions{LabelSel: labelSelector, PerPage: 5000}}
		if instances, err := conf.CfClient.ServiceInstances.ListAll(conf.CfCtx, &instanceListOption); err != nil {
			fmt.Printf("failed to list service instances with label %s : %s\n", conf.LabelNameName, err)
			util.WriteHttpResponse(w, http.StatusInternalServerError, "failed to list sources, internal error")
			return
		} else {
			if len(instances) == 0 {
				util.PrintfIfDebug("could not find any service instances with label %s\n", conf.LabelNameName)
			} else {
				for _, instance := range instances {
					if name, ok := instance.Metadata.Labels[conf.LabelNameName]; ok {
						space := util.GetSpaceByGuidCached(instance.Relationships.Space.Data.GUID)
						org := util.GetOrgByGuidCached(space.Relationships.Organization.Data.GUID)
						desc := instance.Metadata.Annotations[conf.AnnotationNameDesc]
						sourcesList.SourcesResponses = append(sourcesList.SourcesResponses, model.SourceResponse{Source: *name, Org: org.Name, Space: space.Name, Description: *desc})
					}
				}
				util.PrintfIfDebug("found %d sources\n", len(sourcesList.SourcesResponses))
			}
		}

		if len(sourcesList.SourcesResponses) > 0 {
			util.WriteHttpResponse(w, http.StatusOK, sourcesList)
		} else {
			util.WriteHttpResponse(w, http.StatusNoContent, "no sources found")
		}
	}
}

// ValidateRequest - We validate the incoming http request, it should have a valid JWT, there should be a user_id claim in the JWT, the request body should be json-parse-able and the user should be authorized for the requested space.
func ValidateRequest(w http.ResponseWriter, r *http.Request) (bool, string, model.GenericRequest) {
	var userId string
	var requestObject model.GenericRequest
	if token, ok := context.Get(r, "jwt").(jwt.Token); !ok {
		util.WriteHttpResponse(w, http.StatusBadRequest, "failed to parse access token")
	} else {
		userId = token.Claims.(jwt.MapClaims)["user_id"].(string)
		if body, err := io.ReadAll(r.Body); err != nil {
			util.WriteHttpResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to read request body: %s", err))
		} else {
			if err = json.Unmarshal(body, &requestObject); err != nil {
				util.WriteHttpResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to parse request body: %s", err))
			} else {
				if util.IsUserAuthorisedForSpace(token, requestObject.SpaceGUID) {
					return true, userId, requestObject
				} else {
					util.WriteHttpResponse(w, http.StatusUnauthorized, fmt.Sprintf("you are not authorized for space %s", requestObject.SpaceGUID))
				}
			}
		}
	}
	return false, userId, requestObject
}
