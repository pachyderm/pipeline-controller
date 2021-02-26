module github.com/pachyderm/pipeline-controller

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pachyderm/pachyderm/v2 v2.0.0-alpha.5
	github.com/sirupsen/logrus v1.6.0
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.0
)

replace github.com/pachyderm/pachyderm/v2 => ../../pachyderm

replace k8s.io/api => k8s.io/api v0.0.0-20190918155943-95b840bb6a1f

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible
