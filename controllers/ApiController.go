package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rabobank/npsb/util"
)

func userAndService(w http.ResponseWriter, r *http.Request) (username string, serviceInstanceId string, ok bool) {
	ok = false
	authenticationContext, isType := r.Context().Value("authentication").(map[string]interface{})
	if !isType {
		fmt.Println("API reached without an authentication context. Failing")
		util.WriteHttpResponse(w, http.StatusInternalServerError, "Authorization Failure")
		return
	}
	entry, found := authenticationContext["user"]
	if !found {
		fmt.Println("API reached without an authentication context user. Failing")
		util.WriteHttpResponse(w, http.StatusInternalServerError, "Authorization Failure")
		return
	}
	username = entry.(string)
	serviceInstanceId = mux.Vars(r)["service_instance_guid"]
	ok = true
	return
}

func listMapKeys(m map[string]interface{}, keys []string, prefix string) []string {
	for k, v := range m {
		if subMap, isType := v.(map[string]interface{}); isType {
			keys = listMapKeys(subMap, keys, prefix+k+".")
		} else {
			keys = append(keys, prefix+k)
		}
	}
	return keys
}

func updateMap(originalMap map[string]interface{}, updatedValues map[string]interface{}) {
	for k, v := range updatedValues {
		if originalValue, found := originalMap[k]; !found {
			// the original map doesn't have the key, just set it with whatever value is given
			originalMap[k] = v
		} else if updateSubMap, isMap := v.(map[string]interface{}); !isMap {
			// the updated value for the key is not a map, simply overwrite the value
			originalMap[k] = v
		} else if originalSubMap, isMap := originalValue.(map[string]interface{}); isMap {
			// the updated value is a map and the original value for the key is also a map, let's merge it recursively
			updateMap(originalSubMap, updateSubMap)
		} else {
			// the original value is not a map, simply set it with whatever value is provided
			originalMap[k] = v
		}
	}
}

func deleteKey(credentials map[string]interface{}, parts []string) bool {
	if value, isFound := credentials[parts[0]]; isFound {
		if len(parts) > 1 {
			// the key has more parts, let's check if it's a map
			if subMap, isMap := value.(map[string]interface{}); isMap {
				if deleteKey(subMap, parts[1:]) {
					if len(subMap) == 0 {
						delete(credentials, parts[0])
					}
					return true
				}
			}
		} else {
			delete(credentials, parts[0])
			return true
		}
	}
	return false
}

func deleteKeys(credentials map[string]interface{}, keysToDelete []string) ([]string, bool) {
	var ignoredKeys []string
	var deletedKeys bool
	for _, k := range keysToDelete {
		keyParts := strings.Split(k, ".")
		if len(keyParts) == 0 {
			ignoredKeys = append(ignoredKeys, k)
		} else if !deleteKey(credentials, keyParts) {
			ignoredKeys = append(ignoredKeys, k)
		} else {
			deletedKeys = true
		}
	}
	return ignoredKeys, deletedKeys
}

func ListServiceKeys(w http.ResponseWriter, r *http.Request) {
	if username, serviceInstanceId, ok := userAndService(w, r); ok {
		fmt.Printf("[API] %s Listing keys of service %s\n", username, serviceInstanceId)
		fmt.Printf("credentials for service %s do not have a json object map as a value\n", serviceInstanceId)
		util.WriteHttpResponse(w, http.StatusBadRequest, "credentials are not a json object")
	}
}

func ListServiceVersions(w http.ResponseWriter, r *http.Request) {
	if username, serviceInstanceId, ok := userAndService(w, r); ok {
		fmt.Printf("[API] %s listing credential versions for service %s\n", username, serviceInstanceId)
		util.WriteHttpResponse(w, http.StatusNotFound, "credentials don't have any valid versions")
	}
}

func UpdateServiceKeys(w http.ResponseWriter, r *http.Request) {
	if username, serviceInstanceId, ok := userAndService(w, r); ok {
		fmt.Printf("[API] %s updating keys of service %s\n", username, serviceInstanceId)
		util.WriteHttpResponse(w, http.StatusBadRequest, "credentials are not a json object")
	}
}

func DeleteServiceKeys(w http.ResponseWriter, r *http.Request) {
	if username, serviceInstanceId, ok := userAndService(w, r); ok {
		fmt.Printf("[API] %s deleting keys from service %s\n", username, serviceInstanceId)
		util.WriteHttpResponse(w, http.StatusBadRequest, "credentials are not a json object")
	}
}

func ReinstateServiceVersion(w http.ResponseWriter, r *http.Request) {
	if username, serviceInstanceId, ok := userAndService(w, r); ok {
		versionId := mux.Vars(r)["version_id"]
		fmt.Printf("[API] %s reinstating credential version %s for service %s\n", username, versionId, serviceInstanceId)

		util.WriteHttpResponse(w, http.StatusAccepted, "Credentials Updated")
	}
}
