module github.com/rabobank/npsb

go 1.23

replace (
	golang.org/x/net => golang.org/x/net v0.29.0
	golang.org/x/sys => golang.org/x/sys v0.25.0
	golang.org/x/text => golang.org/x/text v0.18.0
)

require (
	github.com/cloudfoundry-community/go-cfenv v1.18.0
	github.com/cloudfoundry-community/go-uaa v0.3.3
	github.com/cloudfoundry/go-cfclient/v3 v3.0.0-alpha.9
	github.com/gorilla/mux v1.8.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/codegangsta/inject v0.0.0-20150114235600-33e0aa1cb7c0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-martini/martini v0.0.0-20170121215854-22fa46961aab // indirect
	github.com/martini-contrib/render v0.0.0-20150707142108-ec18f8345a11 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/oauth2 v0.22.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)
