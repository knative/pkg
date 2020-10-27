module knative.dev/pkg

go 1.14

require (
	cloud.google.com/go v0.65.0
	cloud.google.com/go/storage v1.10.0
	contrib.go.opencensus.io/exporter/ocagent v0.7.1-0.20200907061046-05415f1de66d
	contrib.go.opencensus.io/exporter/prometheus v0.2.1-0.20200609204449-6bcf6f8577f0
	contrib.go.opencensus.io/exporter/stackdriver v0.13.2
	contrib.go.opencensus.io/exporter/zipkin v0.1.1
	github.com/blang/semver v3.5.1+incompatible
	github.com/census-instrumentation/opencensus-proto v0.3.0
	github.com/davecgh/go-spew v1.1.1
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.2
	github.com/google/go-github/v27 v27.0.6
	github.com/google/gofuzz v1.1.0
	github.com/google/mako v0.0.0-20190821191249-122f8dcef9e3
	github.com/google/uuid v1.1.1
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/grpc-gateway v1.14.8 // indirect
	github.com/hashicorp/golang-lru v0.5.4
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/markbates/inflect v1.0.4
	github.com/openzipkin/zipkin-go v0.2.2
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/common v0.9.1
	github.com/spf13/pflag v1.0.5
	github.com/tsenart/vegeta/v12 v12.8.4
	go.opencensus.io v0.22.4
	go.uber.org/atomic v1.6.0
	go.uber.org/automaxprocs v1.3.0
	go.uber.org/multierr v1.5.0
	go.uber.org/zap v1.15.0
	golang.org/x/net v0.0.0-20200904194848-62affa334b73
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/sys v0.0.0-20200905004654-be1d3432aa8f // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0
	google.golang.org/api v0.31.0
	google.golang.org/genproto v0.0.0-20200904004341-0bd0a958aa1d
	google.golang.org/grpc v1.31.1
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20190709130402-674ba3eaed22
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
	k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver v0.18.4
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.18.8
	k8s.io/gengo v0.0.0-20200205140755-e0e292d8aa12
	k8s.io/klog v1.0.0
	knative.dev/test-infra v0.0.0-20201026182042-46291de4ab66
	sigs.k8s.io/boskos v0.0.0-20200729174948-794df80db9c9
)

replace (
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/code-generator => k8s.io/code-generator v0.18.8
)
