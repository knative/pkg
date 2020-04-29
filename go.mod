module knative.dev/pkg

go 1.13

require (
	cloud.google.com/go v0.55.0
	cloud.google.com/go/storage v1.6.0
	contrib.go.opencensus.io/exporter/ocagent v0.6.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	contrib.go.opencensus.io/exporter/zipkin v0.1.1
	github.com/aws/aws-sdk-go v1.29.34 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dgryski/go-gk v0.0.0-20200319235926-a69029f61654 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/gobuffalo/envy v1.7.0 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.3.5
	github.com/google/go-cmp v0.4.0
	github.com/google/go-github/v27 v27.0.6
	github.com/google/gofuzz v1.1.0
	github.com/google/mako v0.0.0-20190821191249-122f8dcef9e3
	github.com/google/uuid v1.1.1
	github.com/gorilla/websocket v1.4.0
	github.com/grpc-ecosystem/grpc-gateway v1.12.1 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/influxdata/tdigest v0.0.0-20181121200506-bf2b5ad3c0a9 // indirect
	github.com/jmespath/go-jmespath v0.3.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/markbates/inflect v1.0.4
	github.com/openzipkin/zipkin-go v0.2.2
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/common v0.9.1
	github.com/prometheus/procfs v0.0.11 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/tsenart/vegeta v12.7.1-0.20190725001342-b5f4fca92137+incompatible
	go.opencensus.io v0.22.3
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0
	go.uber.org/zap v1.9.2-0.20180814183419-67bc79d13d15
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/sys v0.0.0-20200327173247-9dae0f8f5775 // indirect
	golang.org/x/tools v0.0.0-20200329025819-fd4102a86c65 // indirect
	gomodules.xyz/jsonpatch/v2 v2.0.1
	google.golang.org/api v0.20.0
	google.golang.org/genproto v0.0.0-20200326112834-f447254575fd // indirect
	google.golang.org/grpc v1.28.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.7
	k8s.io/api v0.16.4
	k8s.io/apiextensions-apiserver v0.16.4
	k8s.io/apimachinery v0.16.5-beta.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.18.0
	k8s.io/gengo v0.0.0-20200205140755-e0e292d8aa12
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
	k8s.io/test-infra v0.0.0-20191212060232-70b0b49fe247
	k8s.io/utils v0.0.0-20190907131718-3d4f5b7dea0b // indirect
	knative.dev/test-infra v0.0.0-20200429132042-cb2fc4ae428f
)

replace (
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2

	k8s.io/api => k8s.io/api v0.16.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.4
	k8s.io/client-go => k8s.io/client-go v0.16.4
	k8s.io/code-generator => k8s.io/code-generator v0.16.4
)
