package gitserver

import (
	"context"
	"fmt"

	routeApi "github.com/openshift/api/route/v1"
	"github.com/tektoncd/triggers/pkg/reconciler/eventlistener"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/tektoncd"
)

type CreateEventListener struct {
	k8sClient client.Client
}

func NewCreateEventListener(k8sClient client.Client) *CreateEventListener {
	return &CreateEventListener{k8sClient: k8sClient}
}

func (h *CreateEventListener) ServeRequest(ctx context.Context, gitServer *codebaseApi.GitServer) error {
	if err := h.createEventListener(ctx, gitServer); err != nil {
		return err
	}

	if platform.IsOpenshift() {
		return h.createRoute(ctx, gitServer)
	}

	return h.createIngress(ctx, gitServer)
}

func (h *CreateEventListener) createEventListener(ctx context.Context, gitServer *codebaseApi.GitServer) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Creating EventListener")

	// Use Unstructured to avoid direct dependency on "knative.dev/pkg/apis/duck/v1" because EventListener relies on it.
	// This dependency can conflict with the operator's dependencies.
	// https://github.com/tektoncd/triggers/blob/v0.27.0/pkg/apis/triggers/v1beta1/event_listener_types.go#L86
	el := tektoncd.NewEventListenerUnstructured()

	el.SetName(gitServer.Name)
	el.SetNamespace(gitServer.Namespace)
	el.Object["spec"] = map[string]interface{}{
		"serviceAccountName": "default",
		"triggers": []interface{}{
			map[string]interface{}{
				"triggerRef": fmt.Sprintf("%s-build", gitServer.Spec.GitProvider),
			},
			map[string]interface{}{
				"triggerRef": fmt.Sprintf("%s-review", gitServer.Spec.GitProvider),
			},
		},
		"resources": map[string]interface{}{
			"kubernetesResource": map[string]interface{}{
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"serviceAccountName": fmt.Sprintf("tekton-triggers-sa-%s", gitServer.Namespace),
							"containers": []interface{}{
								map[string]interface{}{
									"name": "",
									"resources": map[string]interface{}{
										"limits": map[string]interface{}{
											"cpu":    "500m",
											"memory": "128Mi",
										},
										"requests": map[string]interface{}{
											"cpu":    "50m",
											"memory": "64Mi",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(gitServer, el, h.k8sClient.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference for EventListener: %w", err)
	}

	elCheck := tektoncd.NewEventListenerUnstructured()
	err := h.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: gitServer.Namespace,
		Name:      gitServer.Name,
	}, elCheck)

	if err == nil {
		log.Info("EventListener already exists", "EventListener", el.GetName())
		return nil
	}

	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get EventListener: %w", err)
	}

	if err = h.k8sClient.Create(ctx, el); err != nil {
		return fmt.Errorf("failed to create EventListener: %w", err)
	}

	log.Info("EventListener has been created", "EventListener", el.GetName())

	return nil
}

func (h *CreateEventListener) createIngress(ctx context.Context, gitServer *codebaseApi.GitServer) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Creating Ingress for EventListener")

	pathType := networkingv1.PathTypePrefix
	name := GenerateIngressName(gitServer.Name)

	err := h.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: gitServer.Namespace,
		Name:      name,
	}, &networkingv1.Ingress{})

	if err == nil {
		log.Info("Ingress already exists", "Ingress", name)

		return nil
	}

	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get Ingress: %w", err)
	}

	edpConf, err := platform.GetEdpConfig(ctx, h.k8sClient, gitServer.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get dnsWildcard: %w", err)
	}

	// This port is hardcoded in Tekton Triggers.
	//https://github.com/tektoncd/triggers/blob/v0.27.0/pkg/reconciler/eventlistener/resources/service.go#L37
	const elServicePort = 8080

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: gitServer.Namespace,
			Labels: map[string]string{
				"app.edp.epam.com/gitServer": gitServer.Name,
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: fmt.Sprintf(
						"el-%s-%s.%s",
						gitServer.Name,
						gitServer.Namespace,
						edpConf.DnsWildcard,
					),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: fmt.Sprintf("%s-%s", eventlistener.GeneratedResourcePrefix, gitServer.Name),
											Port: networkingv1.ServiceBackendPort{
												Number: elServicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err = controllerutil.SetControllerReference(gitServer, ingress, h.k8sClient.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference for Ingress: %w", err)
	}

	if err = h.k8sClient.Create(ctx, ingress); err != nil {
		return fmt.Errorf("failed to create Ingress: %w", err)
	}

	log.Info("Ingress has been created", "Ingress", ingress.Name)

	return nil
}

func (h *CreateEventListener) createRoute(ctx context.Context, gitServer *codebaseApi.GitServer) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Creating Route for EventListener")

	name := GenerateIngressName(gitServer.Name)

	err := h.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: gitServer.Namespace,
		Name:      name,
	}, &routeApi.Route{})

	if err == nil {
		log.Info("Route already exists", "Route", name)

		return nil
	}

	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get Route: %w", err)
	}

	edpConf, err := platform.GetEdpConfig(ctx, h.k8sClient, gitServer.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get dnsWildcard: %w", err)
	}

	const routeWeight = int32(100)

	route := &routeApi.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: gitServer.Namespace,
		},
		Spec: routeApi.RouteSpec{
			Host: fmt.Sprintf(
				"el-%s-%s.%s",
				gitServer.Name,
				gitServer.Namespace,
				edpConf.DnsWildcard,
			),
			TLS: &routeApi.TLSConfig{
				InsecureEdgeTerminationPolicy: routeApi.InsecureEdgeTerminationPolicyRedirect,
				Termination:                   routeApi.TLSTerminationEdge,
			},
			To: routeApi.RouteTargetReference{
				Kind:   "Service",
				Name:   fmt.Sprintf("%s-%s", eventlistener.GeneratedResourcePrefix, gitServer.Name),
				Weight: ptr.To(routeWeight),
			},
			Port: &routeApi.RoutePort{
				TargetPort: intstr.FromString("http-listener"),
			},
			WildcardPolicy: routeApi.WildcardPolicyNone,
		},
	}

	if err = controllerutil.SetControllerReference(gitServer, route, h.k8sClient.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference for Route: %w", err)
	}

	if err = h.k8sClient.Create(ctx, route); err != nil {
		return fmt.Errorf("failed to create Route: %w", err)
	}

	log.Info("Route has been created", "Route", route.Name)

	return nil
}

func GenerateIngressName(gitServerName string) string {
	return fmt.Sprintf("event-listener-%s", gitServerName)
}
