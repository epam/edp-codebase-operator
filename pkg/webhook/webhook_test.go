package webhook

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var _ = Describe("webhooks registration", func() {
	It("should register validation webhooks", func() {
		By("creating manager")
		k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: k8sClient.Scheme(),
			Metrics: metricsserver.Options{
				BindAddress: "0",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		By("registering validation webhooks")
		err = RegisterValidationWebHook(k8sManager)
		Expect(err).ToNot(HaveOccurred())
	})

	When("scheme doesn't contain cdpipeline types", func() {
		It("should return error", func() {
			By("creating manager")
			k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme: scheme.Scheme,
				Metrics: metricsserver.Options{
					BindAddress: "0",
				},
			})
			Expect(err).ToNot(HaveOccurred())

			By("registering validation webhooks")
			err = RegisterValidationWebHook(k8sManager)
			Expect(err).To(HaveOccurred())
		})
	})
})
