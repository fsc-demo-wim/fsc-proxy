module github.com/fsc-demo-wim/fsc-proxy

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/mwitkow/grpc-proxy v0.0.0-20181017164139-0f1106ef9c76
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	google.golang.org/grpc v1.27.0
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/cluster-api v0.3.14
	sigs.k8s.io/controller-runtime v0.7.0
)
