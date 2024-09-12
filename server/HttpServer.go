package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rabobank/npsb/conf"
	"github.com/rabobank/npsb/controllers"
	"github.com/rabobank/npsb/security"
)

func StartServer() {
	brokerRouter := mux.NewRouter()

	brokerRouter.Use(controllers.DebugMiddleware)
	// oauth2 interceptor. It will only handle /api endpoints
	brokerRouter.Use(security.MatchPrefix("/health").AuthenticateWith(security.Anonymous).
		MatchPrefix("/api").AuthenticateWith(security.UAA).
		Default(security.BasicAuth).Build())
	brokerRouter.Use(controllers.AuditLogMiddleware)

	// service broker endpoints
	brokerRouter.HandleFunc("/v2/catalog", controllers.Catalog).Methods("GET")
	brokerRouter.HandleFunc("/v2/service_instances/{service_instance_guid}", controllers.CreateOrUpdateServiceInstance).Methods("PUT", "PATCH")
	brokerRouter.HandleFunc("/v2/service_instances/{service_instance_guid}", controllers.DeleteServiceInstance).Methods("DELETE")
	brokerRouter.HandleFunc("/v2/service_instances/{service_instance_guid}/service_bindings/{service_binding_guid}", controllers.CreateServiceBinding).Methods("PUT")
	brokerRouter.HandleFunc("/v2/service_instances/{service_instance_guid}/service_bindings/{service_binding_guid}", controllers.DeleteServiceBinding).Methods("DELETE")

	// key management api endpoints
	brokerRouter.HandleFunc("/api/{service_instance_guid}/keys", controllers.ListServiceKeys).Methods("GET")
	brokerRouter.HandleFunc("/api/{service_instance_guid}/keys", controllers.UpdateServiceKeys).Methods("PUT")
	brokerRouter.HandleFunc("/api/{service_instance_guid}/keys", controllers.DeleteServiceKeys).Methods("DELETE")
	brokerRouter.HandleFunc("/api/{service_instance_guid}/versions", controllers.ListServiceVersions).Methods("GET")
	brokerRouter.HandleFunc("/api/{service_instance_guid}/version/{version_id}", controllers.ReinstateServiceVersion).Methods("PUT")

	http.Handle("/", brokerRouter)

	brokerRouter.Use(controllers.AddHeadersMiddleware)

	fmt.Printf("server started, listening on port %d...\n", conf.ListenPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", conf.ListenPort), nil)
	if err != nil {
		fmt.Printf("failed to start http server on port %d, err: %s\n", conf.ListenPort, err)
		os.Exit(8)
	}
}
