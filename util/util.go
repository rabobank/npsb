package util

import (
	"bytes"
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"github.com/golang-jwt/jwt"
	"github.com/rabobank/npsb/model"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/config"

	"github.com/rabobank/npsb/conf"
)

var guid2appNameCache = make(map[string]CacheEntry)
var cacheCleanerStarted = false
var spaceCache = make(map[string]*resource.Space)
var orgCache = make(map[string]*resource.Organization)

type CacheEntry struct {
	created time.Time
	name    string
}

func InitCFClient() {
	var err error
	if conf.CfConfig, err = config.New(conf.CfApiURL, config.ClientCredentials(conf.ClientId, conf.ClientSecret), config.SkipTLSValidation(), config.UserAgent(fmt.Sprintf("npsb/%s", conf.GetVersion()))); err != nil {
		log.Fatalf("failed to create new config: %s", err)
	}
	if conf.CfClient, err = client.New(conf.CfConfig); err != nil {
		log.Fatalf("failed to create new client: %s", err)
	} else {
		// refresh the client every hour to get a new refresh token
		go func() {
			channel := time.Tick(time.Duration(15) * time.Minute)
			for range channel {
				conf.CfClient, err = client.New(conf.CfConfig)
				if err != nil {
					log.Printf("failed to refresh cfclient, error is %s", err)
				}
			}
		}()
	}
	return
}

func WriteHttpResponse(w http.ResponseWriter, code int, object interface{}) {
	data, err := json.Marshal(object)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, err.Error())
		return
	}

	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, string(data))
	PrintfIfDebug("response: code:%d, body: %s\n", code, string(data))
}

// BasicAuth validate if user/pass in the http request match the configured service broker user/pass
func BasicAuth(w http.ResponseWriter, r *http.Request, username, password string) bool {
	user, pass, ok := r.BasicAuth()
	if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
		w.Header().Set("WWW-Authenticate", `Basic realm="`+conf.BasicAuthRealm+`"`)
		w.WriteHeader(401)
		_, _ = w.Write([]byte("Unauthorised.\n"))
		return false
	}
	return true
}

func DumpRequest(r *http.Request) {
	if conf.Debug {
		fmt.Printf("dumping %s request for URL: %s\n", r.Method, r.URL)
		fmt.Println("dumping request headers...")
		// Loop over header names
		for name, values := range r.Header {
			if name == "Authorization" {
				fmt.Printf(" %s: %s\n", name, "<redacted>")
			} else {
				// Loop over all values for the name.
				for _, value := range values {
					fmt.Printf(" %s: %s\n", name, value)
				}
			}
		}

		// dump the request body
		fmt.Println("dumping request body...")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("Error reading body: %v\n", err)
		} else {
			fmt.Println(string(body))
		}
		// Restore the io.ReadCloser to it s original state
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}
}

func ProvisionObjectFromRequest(r *http.Request, object interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("failed to read json object from request, error: %s\n", err)
		return err
	}
	PrintfIfDebug("received body:%v\n", string(body))
	err = json.Unmarshal(body, object)
	if err != nil {
		fmt.Printf("failed to parse json object from request, error: %s\n", err)
		return err
	}
	return nil
}

func Guid2AppName(guid string) string {
	if !cacheCleanerStarted {
		cacheCleanerStarted = true
		go func() {
			for {
				time.Sleep(5 * time.Second)
				for key, value := range guid2appNameCache {
					if time.Since(value.created) > 1*time.Minute {
						delete(guid2appNameCache, key)
						PrintfIfDebug("cleaned cache entry for key %s\n", key)
					}
				}
			}
		}()
	}
	if cacheEntry, found := guid2appNameCache[guid]; found {
		PrintfIfDebug("cache hit for guid %s\n", guid)
		return cacheEntry.name
	}
	if app, err := conf.CfClient.Applications.Get(conf.CfCtx, guid); err != nil {
		fmt.Printf("failed to get app by name %s, error: %s\n", guid, err)
		return ""
	} else {
		guid2appNameCache[guid] = CacheEntry{created: time.Now(), name: app.Name}
		return app.Name
	}
}

func GetSpaceByGuidCached(guid string) (space *resource.Space) {
	var err error
	var found bool
	if space, found = spaceCache[guid]; found {
		return space
	}
	if space, err = conf.CfClient.Spaces.Get(conf.CfCtx, guid); err != nil {
		fmt.Printf("failed to get space by guid %s, error: %s\n", guid, err)
	}
	spaceCache[guid] = space
	return space
}

func GetOrgByGuidCached(guid string) (org *resource.Organization) {
	var err error
	var found bool
	if org, found = orgCache[guid]; found {
		return org
	}
	if org, err = conf.CfClient.Organizations.Get(conf.CfCtx, guid); err != nil {
		fmt.Printf("failed to get org by guid %s, error: %s\n", guid, err)
	}
	orgCache[guid] = org
	return org
}

// IsUserAuthorisedForSpace - It takes the jwt, extracts the userId from it, then queries cf (/v3/roles) to check if that user has at least developer or manager role for the give space
func IsUserAuthorisedForSpace(token jwt.Token, spaceGuid string) bool {
	userId := token.Claims.(jwt.MapClaims)["user_id"].(string)
	scopes := token.Claims.(jwt.MapClaims)["scope"].([]interface{})
	if Contains(scopes, "cloud_controller.admin") {
		return true
	}
	roleListOption := client.RoleListOptions{
		ListOptions: &client.ListOptions{}, Types: client.Filter{Values: []string{"space_developer", "space_manager"}}, SpaceGUIDs: client.Filter{Values: []string{spaceGuid}}, UserGUIDs: client.Filter{Values: []string{userId}}}
	if roles, err := conf.CfClient.Roles.ListAll(conf.CfCtx, &roleListOption); err != nil {
		fmt.Printf("failed to query Cloud Controller for roles: %s\n", err)
		return false
	} else {
		if len(roles) == 0 {
			fmt.Printf("no roles found for userId %s and spaceguid %s\n", userId, spaceGuid)
			return false
		}
		return true
	}
}

// Send2PolicyServer - Send the give network policies to the cf policy server (actual update/add of the network policy). If a policy already exists, it will be ignored.
func Send2PolicyServer(action string, policies model.NetworkPolicies) error {
	tokenSource, _ := conf.CfConfig.CreateOAuth2TokenSource(conf.CfCtx)
	token, _ := tokenSource.Token()
	var httpClient http.Client
	if conf.SkipSslValidation {
		// Create new Transport that ignores untrusted CA's
		clientAllowUntrusted := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		httpClient = http.Client{Transport: clientAllowUntrusted, Timeout: 30 * time.Second}
	} else {
		httpClient = http.Client{Timeout: 30 * time.Second}
	}
	policyServerEndpoint := conf.CfApiURL + "/networking/v0/external/policies"
	if action == conf.ActionUnbind {
		policyServerEndpoint = conf.CfApiURL + "/networking/v0/external/policies/delete"
	}
	chunks := chunkSlice(policies.Policies, 500)
	for ix, chunk := range chunks {
		var policiesJsonBA []byte
		policiesJsonBA, err := json.Marshal(model.NetworkPolicies{Policies: chunk})
		if err != nil {
			return fmt.Errorf("failed to marshal policies to json: %s", err)
		} else {
			if conf.Debug {
				fmt.Printf("chunk %d - sending %d policy %s action(s) to policy server:\n%v\n", ix, len(chunk), action, chunk)
			} else {
				fmt.Printf("chunk %d - sending %d policy %s action(s) to policy server\n", ix, len(chunk), action)
			}
			request, err := http.NewRequest("POST", policyServerEndpoint, bytes.NewBuffer(policiesJsonBA))
			if err != nil {
				return fmt.Errorf("The HTTP NewRequest failed with error %s\n", err)
			} else {
				request.Header.Set("Authorization", token.AccessToken)
				request.Header.Set("Content-type", "application/json")
				startTime := time.Now().UnixNano() / int64(time.Millisecond)
				response, err := httpClient.Do(request)
				endTime := time.Now().UnixNano() / int64(time.Millisecond)
				if err != nil || response.StatusCode != http.StatusOK {
					if response != nil {
						if response.StatusCode != http.StatusOK {
							bodyBytes, _ := io.ReadAll(response.Body)
							return fmt.Errorf("The HTTP request failed with response code %d: %s\n", response.StatusCode, bodyBytes)
						}
					} else {
						return fmt.Errorf("The HTTP request failed with error: %s\n", err)
					}
				} else {
					bodyBytes, _ := io.ReadAll(response.Body)
					bodyString := string(bodyBytes)
					_ = response.Body.Close()
					fmt.Printf("response in %d ms from %v: Status code: %v: %v\n", endTime-startTime, policyServerEndpoint, response.Status, bodyString)
				}
			}
		}
	}
	return nil
}

// chunkSlice - "chop" the give slice in smaller pieces and return them
func chunkSlice(slice []model.NetworkPolicy, chunkSize int) [][]model.NetworkPolicy {
	var chunks [][]model.NetworkPolicy
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// GetAccessTokenFromRequest - get the JWT from the request
func GetAccessTokenFromRequest(r *http.Request) (string, error) {
	var accessToken string
	if authHeaders := r.Header["Authorization"]; authHeaders != nil && len(authHeaders) != 0 {
		accessToken = strings.TrimPrefix(authHeaders[0], "bearer ")
	} else {
		return accessToken, errors.New("no Authorization header found")
	}
	return accessToken, nil
}

// IsValidJKU - We compare the jku with the api hostname, only the first part should be different: like uaa.sys.cfd04.aws.rabo.cloud versus api.sys.cfd04.aws.rabo.cloud
func IsValidJKU(jkuURL string) bool {
	parsedJkuURL, err := url.Parse(jkuURL)
	if err != nil {
		fmt.Printf("jku URL %s is invalid: %s", jkuURL, err)
		return false
	}
	apiURL, _ := url.Parse(conf.CfApiURL)
	apiDomain := strings.TrimPrefix(apiURL.Hostname(), "api.")
	jkuDomain := strings.TrimPrefix(parsedJkuURL.Hostname(), "uaa.")
	if jkuDomain != apiDomain {
		fmt.Printf("jku URL %s is invalid", jkuURL)
		return false
	}
	return true
}

func PrintIfDebug(msg string) {
	if conf.Debug {
		fmt.Print(msg)
	}
}

func PrintfIfDebug(msg string, args ...interface{}) {
	PrintIfDebug(fmt.Sprintf(msg, args...))
}

func Contains(elems []interface{}, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

// SyncLabels2Policies - Find all ServiceInstances and their bound apps, figure out what network policies they represent, check if they exist, and if not, report and create them.
func SyncLabels2Policies() {
	PrintfIfDebug("syncing labels to network policies...\n")
	startTime := time.Now()
	var allInstancesWithBinds []model.InstancesWithBinds
	var totalServiceInstances int
	var totalBinds int

	//
	// find all Instances with their binds, both source and destination
	labelSelector := client.LabelSelector{}
	labelSelector.Existence(conf.LabelNameType)
	instanceListOption := client.ServiceInstanceListOptions{ListOptions: &client.ListOptions{LabelSel: labelSelector, PerPage: 5000}}
	if instances, err := conf.CfClient.ServiceInstances.ListAll(conf.CfCtx, &instanceListOption); err != nil {
		fmt.Printf("failed to list all service instances with label %s: %s\n", conf.LabelNameType, err)
	} else {
		if len(instances) < 1 {
			PrintfIfDebug("could not find any service instances with label %s\n", conf.LabelNameType)
		} else {
			totalServiceInstances = len(instances)

			//
			// get all "npsb" service bindings (by filtering on the presence of the label npsb.dest.port)
			labelSelector = client.LabelSelector{}
			labelSelector.Existence(conf.LabelNamePort)
			bindListOption := client.ServiceCredentialBindingListOptions{ListOptions: &client.ListOptions{LabelSel: labelSelector, PerPage: 5000}}
			if bindings, err := conf.CfClient.ServiceCredentialBindings.ListAll(conf.CfCtx, &bindListOption); err != nil {
				fmt.Printf("failed to list all service bindings with label %s: %s\n", conf.LabelNamePort, err)
			} else {
				if len(bindings) < 1 {
					PrintfIfDebug("could not find any service bindings with label %s\n", conf.LabelNameType)
				} else {
					totalBinds = len(bindings)
					for _, instance := range instances {
						var nameOrSource string
						if instance.Metadata.Labels[conf.LabelNameName] != nil && *instance.Metadata.Labels[conf.LabelNameName] != "" {
							nameOrSource = *instance.Metadata.Labels[conf.LabelNameName]
						}
						if instance.Metadata.Labels[conf.LabelNameSourceName] != nil && *instance.Metadata.Labels[conf.LabelNameSourceName] != "" {
							nameOrSource = *instance.Metadata.Labels[conf.LabelNameSourceName]
						}
						instanceWithBinds := model.InstancesWithBinds{
							BoundApps:    make([]model.Destination, 0),
							SrcOrDst:     *instance.Metadata.Labels[conf.LabelNameType],
							NameOrSource: nameOrSource,
						}
						for _, binding := range bindings {
							if binding.Relationships.ServiceInstance.Data.GUID == instance.GUID {
								if instanceWithBinds.SrcOrDst == conf.LabelValueTypeSrc {
									// if it is a type=source, we only need the app name
									instanceWithBinds.BoundApps = append(instanceWithBinds.BoundApps, model.Destination{Id: binding.Relationships.App.Data.GUID})
								} else {
									port := 8080
									if binding.Metadata.Labels[conf.LabelNamePort] != nil && *binding.Metadata.Labels[conf.LabelNamePort] != "" && *binding.Metadata.Labels[conf.LabelNamePort] != "0" {
										port, _ = strconv.Atoi(*binding.Metadata.Labels[conf.LabelNamePort])
									}
									protocol := conf.LabelValueProtocolTCP
									if binding.Metadata.Labels[conf.LabelNameProtocol] != nil && *binding.Metadata.Labels[conf.LabelNameProtocol] != "" {
										protocol = *binding.Metadata.Labels[conf.LabelNameProtocol]
									}
									instanceWithBinds.BoundApps = append(instanceWithBinds.BoundApps, model.Destination{Id: binding.Relationships.App.Data.GUID, Protocol: protocol, Port: port})
								}
							}
						}
						allInstancesWithBinds = append(allInstancesWithBinds, instanceWithBinds)
					}
				}
			}
		}
		PrintfIfDebug("found %d instances with label %s, %d instances have binds:\n", len(instances), conf.LabelNameType, len(allInstancesWithBinds))

		//
		// for each type=source instances, find the destination instances that point to this source instance, and generate the required network policies objects
		var requiredNetworkPolicies []model.NetworkPolicy
		for _, sourceInstance := range allInstancesWithBinds {
			if sourceInstance.SrcOrDst == conf.LabelValueTypeSrc {
				for _, destinationInstance := range allInstancesWithBinds {
					if destinationInstance.SrcOrDst == conf.LabelValueTypeDest && destinationInstance.NameOrSource == sourceInstance.NameOrSource {
						for _, sourceApp := range sourceInstance.BoundApps {
							for _, destinationApp := range destinationInstance.BoundApps {
								networkPolicy := model.NetworkPolicy{Source: model.Source{Id: sourceApp.Id}, Destination: model.Destination{Id: destinationApp.Id, Port: destinationApp.Port, Protocol: destinationApp.Protocol}}
								// add the network policy to the list of network policies
								requiredNetworkPolicies = append(requiredNetworkPolicies, networkPolicy)
								// check if the network policy already exists, if not, create it
							}
						}
					}
				}
			}
		}
		npString := ""
		//for _, np := range requiredNetworkPolicies {
		//	npString += fmt.Sprintf("%s=>%s:%d(%s)\n", Guid2AppName(np.Source.Id), Guid2AppName(np.Destination.Id), np.Destination.Port, np.Destination.Protocol)
		//}
		PrintfIfDebug("found %d network policies that should exist according to labels:\n%v\n", len(requiredNetworkPolicies), npString)

		//
		// get all existing network policies, then for each network policy object check if a real network policy exists, if not, create it
		existingNetworkPolicies := getAllNetworkPolicies()
		policiesFixed := 0
		for _, requiredNetworkPolicy := range requiredNetworkPolicies {
			found := false
			for _, existingNetworkPolicy := range existingNetworkPolicies {
				if existingNetworkPolicy.Source.Id == requiredNetworkPolicy.Source.Id && existingNetworkPolicy.Destination.Id == requiredNetworkPolicy.Destination.Id && existingNetworkPolicy.Destination.Port == requiredNetworkPolicy.Destination.Port && existingNetworkPolicy.Destination.Protocol == requiredNetworkPolicy.Destination.Protocol {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("network policy %s=>%s:%d(%s) does not exist, creating it\n", Guid2AppName(requiredNetworkPolicy.Source.Id), Guid2AppName(requiredNetworkPolicy.Destination.Id), requiredNetworkPolicy.Destination.Port, requiredNetworkPolicy.Destination.Protocol)
				err := Send2PolicyServer(conf.ActionBind, model.NetworkPolicies{Policies: []model.NetworkPolicy{requiredNetworkPolicy}})
				if err != nil {
					fmt.Printf("failed to create network policy %s=>%s:%d(%s): %s\n", Guid2AppName(requiredNetworkPolicy.Source.Id), Guid2AppName(requiredNetworkPolicy.Destination.Id), requiredNetworkPolicy.Destination.Port, requiredNetworkPolicy.Destination.Protocol, err)
				} else {
					policiesFixed++
				}
			}
		}
		endTime := time.Now()
		fmt.Printf("checked %d service instances, checked %d binds, fixed %d missing network policies in %d ms\n", totalServiceInstances, totalBinds, policiesFixed, endTime.Sub(startTime).Milliseconds())
	}
}

// getAllNetworkPolicies - query the policy server and return all network-policies
func getAllNetworkPolicies() []model.NetworkPolicy {
	polServerResponse := &model.PolicyServerGetResponse{}
	policyServerEndpoint := conf.CfApiURL + "/networking/v0/external/policies"
	tokenSource, _ := conf.CfClient.CreateOAuth2TokenSource(conf.CfCtx)
	token, _ := tokenSource.Token()
	requestHeader := map[string][]string{"Content-Type": {"application/json"}, "Authorization": {token.AccessToken}}
	requestUrl, _ := url.Parse(policyServerEndpoint)
	httpRequest := http.Request{Method: http.MethodGet, URL: requestUrl, Header: requestHeader}
	var httpClient http.Client
	if conf.SkipSslValidation {
		// Create new Transport that ignores untrusted CA's
		clientAllowUntrusted := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		httpClient = http.Client{Transport: clientAllowUntrusted, Timeout: 30 * time.Second}
	} else {
		httpClient = http.Client{Timeout: 30 * time.Second}
	}
	response, err := httpClient.Do(&httpRequest)
	if err != nil || (response != nil && response.StatusCode != http.StatusOK) {
		if err != nil {
			fmt.Printf("request to policy server failed: %s \n", err)
		}
		if response != nil && response.StatusCode != http.StatusOK {
			fmt.Printf("request to policy server failed with response code %d\n", response.StatusCode)
		}
	} else {
		defer func() { _ = response.Body.Close() }()
		bodyBytes, _ := io.ReadAll(response.Body)
		if err = json.Unmarshal(bodyBytes, polServerResponse); err != nil {
			fmt.Printf("Failed to parse GET response from Policy Server: %s\n", err)
		}
	}
	PrintfIfDebug("found %d existing network policies\n", len(polServerResponse.Policies))
	return polServerResponse.Policies
}
