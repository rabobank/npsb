## Network Policy Service Broker

A Cloud Foundry Service Broker that can create/delete CF network policies based on binds by applications.

## Intro

The configuration for the broker consists of the following environment variables:
* **DEBUG** - Debugging on or off, default is false.
* **HTTP_TIMEOUT** - Timeout in seconds for connecting to either UAA or Credhub endpoint, default is 10.
* **CLIENT_ID** - The uaa client to use for logging in to credhub, should have credhub_admin scope.
* **BROKER_USER** - The userid for the broker (should be specified issuing the _cf create-service-broker_ cmd).
* **CATALOG_DIR** - The directory where to find the cf catalog for the broker, the directory should contain a file called catalog.json.
* **LISTEN_PORT** - The port that the broker should listen on, default is 8080.
* **CFAPI_URL** - The URL of the cf api (i.e. https://api.sys.mydomain.com).
* **SKIP_SSL_VALIDATION** - Skip ssl validation or not, default is false.

Describe the instance create parameters and the bind parameters here.

Instance create parameters:
* **type** - This can be either "source" or "destination", indicating the "direction" of the policy. This is a required parameter.
* **name** - The logical name to assign to the instance, only applicable for source instance. This name can be queried later to get a list of all source policies that can be used by type=destination instances. This is a required parameter for type=source instances.
* **description** - The description of the instance, only applicable for source instances. This description can be queried later to get a list of all source policies that can be used by type=destination instances. This is a required parameter for type=source instances.
* **scope** - Can be either local or global. Local scope means only visible to the org/space where the policy is created, global means visible to all orgs/spaces. This is a required parameter for type=source instances.
* **source** - The name of the source service instance that should be linked to this instance, only applicable for destination instances. This is a required parameter. This is a required parameter for type=destination instances.

Instance bind parameters:
* **port** - The port to use for the network policy (i.e. the port the application listens on). This is a optional parameter for type=destination, default is 8080.

## Deploying/installing the broker

First make sure the broker itself runs (as a cf app, since it needs access to credhub.service.cf.internal), and the URL is available to the Cloud Controller.
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
The broker has the envvar CREDS_PATH which points to an entry in the runtime credhub where the following 2 credentials should be stored:
* BROKER_PASSWORD - The password for the broker (should be specified when issuing the _cf create-service-broker_ cmd).
* CLIENT_SECRET - The password for CLIENT_ID

To create the proper credhub entry in the runtime credhub, use the following sample command: 
```
credhub set -n /brokers/npsb/credentials --type json --value='{ "BROKER_PASSWORD": "pswd1", "CLIENT_SECRET": "pswd2" }'
```
