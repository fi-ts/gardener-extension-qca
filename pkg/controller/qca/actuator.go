package qualys

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/fi-ts/gardener-extension-qca/charts"
	"github.com/fi-ts/gardener-extension-qca/pkg/apis/qca/v1alpha1"
	"github.com/fi-ts/gardener-extension-qca/pkg/imagevector"
	extensionsconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fi-ts/gardener-extension-qca/pkg/apis/config"
	gardener "github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	managedResourceName = "qca-resource"
)

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(mgr manager.Manager, config config.ControllerConfiguration) (extension.Actuator, error) {
	ca, err := gardener.NewChartApplierForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}
	return &actuator{
		client:       mgr.GetClient(),
		decoder:      serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		config:       config,
		chartApplier: ca,
	}, nil
}

type actuator struct {
	client       client.Client
	decoder      runtime.Decoder
	config       config.ControllerConfiguration
	chartApplier gardener.ChartApplier
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	cluster, err := controller.GetCluster(ctx, a.client, ex.GetNamespace())
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	qcaImage, err := imagevector.ImageVector().FindImage("qualys-cloud-agent")
	if err != nil {
		return fmt.Errorf("failed to find qualys-cloud-agent image: %w", err)
	}

	ci, err := util.NewChartRendererForShoot(cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return fmt.Errorf("failed to create chart renderer: %w", err)
	}

	var qualysConfig v1alpha1.QCAConfig
	if ex.Spec.ProviderConfig != nil {
		_, _, err := a.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, &qualysConfig)
		if err != nil {
			return fmt.Errorf("failed to decode provider config: %w", err)
		}
	}

	log.Info("tenant configs", "configs", a.config.TenantConfigs, "tenant", qualysConfig.TenantId)
	tenantConfig := a.config.TenantConfigs.GetTenantConfig(qualysConfig.TenantId)
	if tenantConfig == nil {
		return fmt.Errorf("tenant config not found for tenant %s", qualysConfig.TenantId)
	}

	// check if the Metal Stack firewall CRD is installed, so no CWNPs are generated
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusterwidenetworkpolicies.metal-stack.io",
		},
	}
	_, shootClient, err := util.NewClientForShoot(ctx, a.client, ex.Namespace, client.Options{}, extensionsconfigv1alpha1.RESTOptions{})

	if err != nil {
		return fmt.Errorf("failed to create shoot client: %w", err)
	}

	u, err := url.Parse(a.config.Proxy)
	if err != nil {
		return fmt.Errorf("wrong proxy configuration: %w", err)
	}

	// for the firewall rules, we need IP:PORT
	// ??? should we detect(fail)/support NAME:PORT ? the name must be resolveable, but in most cases it is
	// an internal name which cannot be resolved by the k8s dns, so an IP must be set.
	firewallProxyList := []string{u.Host}
	err = shootClient.Get(ctx, client.ObjectKeyFromObject(crd), crd)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("unable to retrieve firewall CRD: %w", err)
		}

		// if the CRD is not found, we don't create a ClusterwideNetworkPolicy by setting the list of proxies to empty
		// the helm-chart will then not create a ClusterwideNetworkPolicy
		log.Info("metal-stack firewall CRD not found, not creating ClusterwideNetworkPolicy")
		firewallProxyList = []string{}
	}

	rc, err := ci.RenderEmbeddedFS(charts.InternalChart, filepath.Join("internal", charts.QCAChartsPath), charts.QCAName, charts.QCANamespace, map[string]any{
		"proxyAddresses": firewallProxyList,
		"namespace": map[string]any{
			"create": false,
			"name":   charts.QCANamespace,
		},
		"agent": map[string]any{
			"customerId":   a.config.CustomerId,
			"activationId": tenantConfig.ActivationId,
			"serverUri":    a.config.Server,
			"proxy":        a.config.Proxy,
		},
		"image": map[string]any{
			"repository": qcaImage.Repository,
			"tag":        qcaImage.Tag,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to render chart: %w", err)
	}

	data := map[string][]byte{
		charts.QCAName: rc.Manifest(),
	}

	// log the generated manifest as base64
	// log.Info("reconciling extension", "configuration", data)

	err = managedresources.CreateForShoot(ctx, a.client, ex.GetNamespace(), managedResourceName, "", false, data)

	if err != nil {
		return fmt.Errorf("failed to apply chart: %w", err)
	}
	log.Info("reconciled extension", "configuration", qualysConfig)
	return nil

}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	log.Info("deleting managed resource")
	err := managedresources.DeleteForShoot(ctx, a.client, ex.GetNamespace(), managedResourceName)
	if err != nil {
		log.Error(err, "cannot delete managed resource")
	}
	return err
}

// ForceDelete the Extension resource
func (a *actuator) ForceDelete(_ context.Context, _ logr.Logger, _ *extensionsv1alpha1.Extension) error {
	return nil
}

// Restore the Extension resource.
func (a *actuator) Restore(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.Reconcile(ctx, log, ex)
}

// Migrate the Extension resource.
func (a *actuator) Migrate(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	log.Info("migrating managed resource")
	err := managedresources.DeleteForShoot(ctx, a.client, ex.GetNamespace(), managedResourceName)
	if err != nil {
		log.Error(err, "cannot delete managed resource")
	}
	return err
}
