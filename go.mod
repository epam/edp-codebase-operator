module github.com/epmd-edp/codebase-operator/v2

go 1.14

replace git.apache.org/thrift.git => github.com/apache/thrift v0.12.0

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20180801171038-322a19404e37

require (
	github.com/DATA-DOG/go-sqlmock v1.4.1
	github.com/andygrunwald/go-jira v1.12.0
	github.com/bndr/gojenkins v0.2.1-0.20181125150310-de43c03cf849
	github.com/dchest/uniuri v0.0.0-20160212164326-8902c56451e9
	github.com/epmd-edp/edp-component-operator v0.1.1-0.20200827122548-e87429a916e0
	github.com/epmd-edp/jenkins-operator/v2 v2.3.0-130.0.20200525102742-f56cd8641faa
	github.com/epmd-edp/perf-operator/v2 v2.0.0-20201028094858-f3e9f1e831b1
	github.com/go-openapi/spec v0.19.3
	github.com/lib/pq v1.0.0
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/operator-framework/operator-sdk v0.0.0-20190530173525-d6f9cdf2f52e
	github.com/pkg/errors v0.8.1
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.4.0
	golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/src-d/go-git.v4 v4.10.0
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/kube-openapi v0.0.0-20181109181836-c59034cc13d5
	sigs.k8s.io/controller-runtime v0.1.12
)
