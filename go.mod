module knative.dev/pkg

go 1.14

require (
	cloud.google.com/go v0.72.0
	cloud.google.com/go/storage v1.10.0
	contrib.go.opencensus.io/exporter/ocagent v0.7.1-0.20200907061046-05415f1de66d
	contrib.go.opencensus.io/exporter/prometheus v0.3.0
	contrib.go.opencensus.io/exporter/stackdriver v0.13.5
	contrib.go.opencensus.io/exporter/zipkin v0.1.2
	github.com/blang/semver/v4 v4.0.0
	github.com/blendle/zapdriver v1.3.1
	github.com/census-instrumentation/opencensus-proto v0.3.0
	github.com/davecgh/go-spew v1.1.1
	github.com/dgryski/go-gk v0.0.0-20200319235926-a69029f61654 // indirect
	github.com/evanphx/json-patch/v5 v5.5.0
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/gobuffalo/flect v0.2.2
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v27 v27.0.6
	github.com/google/gofuzz v1.2.0
	github.com/google/mako v0.0.0-20190821191249-122f8dcef9e3
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/grpc-gateway v1.14.8 // indirect
	github.com/hashicorp/golang-lru v0.5.4
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/influxdata/tdigest v0.0.0-20181121200506-bf2b5ad3c0a9 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/onsi/gomega v1.10.1 // indirect
	github.com/openzipkin/zipkin-go v0.2.5
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.26.0
	github.com/spf13/pflag v1.0.5
	github.com/tsenart/vegeta/v12 v12.8.4
	go.opencensus.io v0.23.0
	go.uber.org/atomic v1.8.0
	go.uber.org/automaxprocs v1.4.0
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	golang.org/x/tools v0.1.2
	gomodules.xyz/jsonpatch/v2 v2.2.0
	google.golang.org/api v0.36.0
	google.golang.org/genproto v0.0.0-20210416161957-9910b6c460de
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.20.7
	k8s.io/apiextensions-apiserver v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	k8s.io/code-generator v0.20.7
	k8s.io/gengo v0.0.0-20210203185629-de9496dff47b
	k8s.io/klog v1.0.0
	knative.dev/hack v0.0.0-20210609124042-e35bcb8f21ec
	sigs.k8s.io/yaml v1.2.0
)
