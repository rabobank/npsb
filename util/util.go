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
	guid    string
}

func InitCFClient() *client.Client {
	var err error
	if conf.CfConfig, err = config.New(conf.CfApiURL, config.ClientCredentials(conf.ClientId, conf.ClientSecret), config.SkipTLSValidation()); err != nil {
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
	return conf.CfClient
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
		// Restore the io.ReadCloser to it's original state
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
					if time.Since(value.created) > 5*time.Second {
						delete(guid2appNameCache, key)
						PrintfIfDebug("cleaned cache entry for key %s\n", key)
					}
				}
			}
		}()
	}
	if cacheEntry, found := guid2appNameCache[guid]; found {
		PrintfIfDebug("cache hit for guid %s\n", guid)
		return cacheEntry.guid
	}
	if app, err := conf.CfClient.Applications.Get(conf.CfCtx, guid); err != nil {
		fmt.Printf("failed to get app by guid %s, error: %s\n", guid, err)
		return ""
	} else {
		guid2appNameCache[guid] = CacheEntry{created: time.Now(), guid: app.Name}
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
					if response.StatusCode != http.StatusOK {
						bodyBytes, _ := io.ReadAll(response.Body)
						return fmt.Errorf("The HTTP request failed with response code %d: %s\n", response.StatusCode, bodyBytes)
					} else {
						return fmt.Errorf("The HTTP request failed with error: %s\n", err)
					}
				} else {
					defer func() { _ = response.Body.Close() }()
					bodyBytes, _ := io.ReadAll(response.Body)
					bodyString := string(bodyBytes)
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
