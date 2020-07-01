module knative.dev/pkg

go 1.14

require (
	cloud.google.com/go v0.61.0
	cloud.google.com/go/storage v1.10.0
	contrib.go.opencensus.io/exporter/ocagent v0.7.0
	contrib.go.opencensus.io/exporter/prometheus v0.2.1-0.20200609204449-6bcf6f8577f0
	contrib.go.opencensus.io/exporter/stackdriver v0.13.2
	contrib.go.opencensus.io/exporter/zipkin v0.1.1
	github.com/aws/aws-sdk-go v1.33.6 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.1
	github.com/davecgh/go-spew v1.1.1
	github.com/dgryski/go-gk v0.0.0-20200319235926-a69029f61654 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.2.0
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.0
	github.com/google/go-github/v27 v27.0.6
	github.com/google/gofuzz v1.1.0
	github.com/google/mako v0.2.0
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.5.0 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/golang-lru v0.5.4
	github.com/influxdata/tdigest v0.0.1 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/markbates/inflect v1.0.4
	github.com/openzipkin/zipkin-go v0.2.2
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.10.0
	github.com/prometheus/procfs v0.1.3 // indirect
	github.com/prometheus/statsd_exporter v0.17.0 // indirect
	github.com/rogpeppe/go-internal v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/tsenart/vegeta v12.7.1-0.20190725001342-b5f4fca92137+incompatible
	go.opencensus.io v0.22.4
	go.uber.org/multierr v1.5.0
	go.uber.org/zap v1.15.0
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0
	google.golang.org/api v0.29.0
	google.golang.org/genproto v0.0.0-20200715011427-11fb19a81f2c
	google.golang.org/grpc v1.30.0
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.18.5
	k8s.io/apiextensions-apiserver v0.18.5
	k8s.io/apimachinery v0.18.5
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.18.5
	k8s.io/gengo v0.0.0-20200710205751-c0d492a0f3ca
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.3.0 // indirect
	k8s.io/kube-openapi v0.0.0-20200615155156-dffdd1682719 // indirect
	k8s.io/test-infra v0.0.0-20200715180038-7d08a0a18660 // indirect
	k8s.io/utils v0.0.0-20200619165400-6e3d28b6ed19 // indirect
	knative.dev/test-infra v0.0.0-20200715150133-fb8ce6a42ed4
	sigs.k8s.io/boskos v0.0.0-20200710214748-f5935686c7fc
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0
	github.com/google/mako => github.com/google/mako v0.0.0-20190821191249-122f8dcef9e3
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.0.0-20190815212128-ab0dd09aa10e
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/code-generator => k8s.io/code-generator v0.17.6
	k8s.io/gengo => k8s.io/gengo v0.0.0-20200630090205-15d76db0a9e6
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.0.0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200331211046-561ec9aa347a
)
