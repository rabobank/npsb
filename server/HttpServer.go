package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rabobank/npsb/conf"
	"github.com/rabobank/npsb/controllers"
)

func StartServer() {
	brokerRouter := mux.NewRouter()

	brokerRouter.Use(controllers.DebugMiddleware)
	brokerRouter.Use(controllers.AddHeadersMiddleware)
	brokerRouter.Use(controllers.BasicAuthMiddleware)
	brokerRouter.HandleFunc("/v2/catalog", controllers.Catalog).Methods("GET")
	brokerRouter.HandleFunc("/v2/service_instances/{service_instance_guid}", controllers.CreateOrUpdateServiceInstance).Methods("PUT")
	brokerRouter.HandleFunc("/v2/service_instances/{service_instance_guid}", controllers.DeleteServiceInstance).Methods("DELETE")
	brokerRouter.HandleFunc("/v2/service_instances/{service_instance_guid}/service_bindings/{service_binding_guid}", controllers.CreateServiceBinding).Methods("PUT")
	brokerRouter.HandleFunc("/v2/service_instances/{service_instance_guid}/service_bindings/{service_binding_guid}", controllers.DeleteServiceBinding).Methods("DELETE")
	http.Handle("/v2/", brokerRouter)

	apiRouter := mux.NewRouter()
	apiRouter.Use(controllers.DebugMiddleware)
	apiRouter.Use(controllers.AddHeadersMiddleware)
	apiRouter.Use(controllers.CheckJWTMiddleware)
	apiRouter.HandleFunc("/api/sources", controllers.GetSources).Methods(http.MethodGet)
	http.Handle("/api/", apiRouter)

	fmt.Printf("server started, listening on port %d...\n", conf.ListenPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", conf.ListenPort), nil)
	if err != nil {
		fmt.Printf("failed to start http server on port %d, err: %s\n", conf.ListenPort, err)
		os.Exit(8)
	}
}
