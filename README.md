# Network Policy Service Broker

A Cloud Foundry Service Broker that can create/delete CF network policies based on binds by applications.

## Introduction

The configuration for the broker consists of the following environment variables:
* **DEBUG** - Debugging on or off, default is false.
* **CLIENT_ID** - The uaa client to use for logging in to credhub, should have credhub_admin scope.
* **CATALOG_DIR** - The directory where to find the cf catalog for the broker, the directory should contain a file called catalog.json.
* **LISTEN_PORT** - The port that the broker should listen on, default is 8080.
* **SYNC_INTERVAL_SECS** - The interval the broker will sync the required network policies (according to the service bindings) with the actual network policies, and will create the missing policies, default is 300.
* **CFAPI_URL** - The URL of the cf api (i.e. https://api.sys.mydomain.com).
* **SKIP_SSL_VALIDATION** - Skip ssl validation or not, default is false.

Instance create parameters:
* **type** - This can be either "source" or "destination", indicating the "direction" of the policy. This is a required parameter.
* **name** - The logical name to assign to the instance, only applicable for source instance. This name can be queried later to get a list of all source policies that can be used by type=destination instances. This is a required parameter for type=source instances.
* **description** - The description of the instance, only applicable for source instances. This description is added as an annotation to the service instance. This is an optional parameter for type=source instances.
* **scope** - Can be either local or global. Local scope means only visible to the org/space where the policy is created, global means visible to all orgs/spaces. This is a required parameter for type=source instances.
* **source** - The name of the source service instance that should be linked to this instance, only applicable for destination instances. This is a required parameter. This is a required parameter for type=destination instances.

Instance bind parameters:
* **port** - The port to use for the network policy (i.e. the port the application listens on). This is an optional parameter for type=destination, default is 8080.
* **protocol** - The protocol to use for the network policy (i.e. tcp or udp). This is an optional parameter for type=destination, default is tcp.

## Deploying/installing the broker

First make sure the broker itself runs (as a cf app, since it needs access to credhub.service.cf.internal), and the broker is available to the Cloud Controller.
Then install the broker:
```
#  the user and password should match with the user/pass you use when starting the broker app
cf create-service-broker npsb <broker-user> <broker-password> <https://url.where.the.broker.runs>
```
Give access to the service (all plans to all orgs):
```
cf enable-service-access npsb
```

## Creating the credentials in the runtime credhub
The broker is reading credentials from credhub, so you need to create a credhub service instance named npsb-credentials: 
```
cf create-service credhub default npsb-credentials -c '{ "brokerUser": "<broker-user>", "brokerPassword": "<broker-password>", "clientId": "<client-id>" , "clientSecret": "<clientsecret>" }'
```

As a fallback if you don't use the above credhub service instance (not recommended for security reasons since having credentials in envvars are easily leaked) you can create these envvars:
* BROKER_USER - The user you created with the cf create-service-broker command
* BROKER_PASSWORD - The password for the broker (should be specified when issuing the _cf create-service-broker_ cmd).
* CLIENT_ID - The uaa client_id used to query the cloud controller and to create network policies
* CLIENT_SECRET - The password for CLIENT_ID

## CC Queries to determine the needed network policies

### Service bind on type=source instances
When doing a service bind on a type=source service instance, the broker should first get a list of all type=target service instances that have the label source=<srcname> where <srcname> is the name of the source service instance:
````
GET /v3/service_instances?label_selector=npsb.type=destination,npsb.dest.source=srcapp1
````
Then we should get a list of service bindings for each of these service instances:
````
GET /v3/service_credential_bindings?per_page=1000&service_instance_guids=guid1,guid2,guid999&label_selector=npsb.dest.port
````
Then we can create network policies for each of these service bindings, where:
* source is the guid of the app that is being bound to the type=source service instance
* targets is a list of guids of the apps that are being bound to the type=destination service instance (.relationships.app.data.guid)
* port is optional, can be derived from the service binding of the type=destination service bindings (metadata.labels.npsb.dest.port), default is 8080

### Service bind on type=destination instances
When doing a service bind on a type=destination service instance, the broker should first get the service instance that has the label name=<srcname> where <srcname> comes from the source label of the current destination binding.
````
GET /v3/service_instances?label_selector=npsb.type=source,npsb.source.name=srcapp1
````
Then we should get a list of service bindings for this service instance (guid1 is the guid of the only type=source service instance that was found with the previous query):
````
GET /v3/service_credential_bindings?per_page=1000&service_instance_guids=guid1
````
Then we can create network policies for each of these service bindings, where:
* source is the guid of the app that bound to the type=source service instance (.relationships.app.data.guid)
* destination is the guid of the app that is currently being bound to the type=destination service instance
* port is optional, can be derived from the service binding of the type=destination service bindings (metadata.labels.npsb.dest.port), default is 8080
* protocol is optional, can be derived from the service binding of the type=destination service bindings (metadata.labels.npsb.dest.protocol), default is tcp
