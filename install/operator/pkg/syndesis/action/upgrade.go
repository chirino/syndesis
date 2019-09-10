package action

import (
	"context"
	"errors"
	"fmt"
	v1 "github.com/openshift/api/image/v1"
	"github.com/syndesisio/syndesis/install/operator/pkg/generator"
	"github.com/syndesisio/syndesis/install/operator/pkg/syndesis/configuration"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"

	"github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	"github.com/syndesisio/syndesis/install/operator/pkg/syndesis/operation"
	"github.com/syndesisio/syndesis/install/operator/pkg/syndesis/template"
	syndesistemplate "github.com/syndesisio/syndesis/install/operator/pkg/syndesis/template"
	"github.com/syndesisio/syndesis/install/operator/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	UpgradePodPrefix = "syndesis-upgrade-"
)

// Upgrades Syndesis to the version supported by this operator using the upgrade template.
type upgradeAction struct {
	baseAction
	operatorVersion string
}

func newUpgradeAction(mgr manager.Manager, api kubernetes.Interface) SyndesisOperatorAction {
	return &upgradeAction{
		newBaseAction(mgr, api, "upgrade"),
		"",
	}
}

func (a *upgradeAction) CanExecute(syndesis *v1alpha1.Syndesis) bool {
	return syndesisPhaseIs(syndesis, v1alpha1.SyndesisPhaseUpgrading)
}

func (a *upgradeAction) Execute(ctx context.Context, syndesis *v1alpha1.Syndesis) error {
	if a.operatorVersion == "" {
		operatorVersion, err := template.GetSyndesisVersionFromOperatorTemplate(a.scheme)
		if err != nil {
			return err
		}
		a.operatorVersion = operatorVersion
	}

	targetVersion := a.operatorVersion

	if syndesis.Status.Version == targetVersion {
		if syndesis.Status.ForceUpgrade {
			a.log.Info("Syndesis resource already upgraded, but ForceUpgrade is enabled.", "name", syndesis.Name, "targetVersion", targetVersion)
		} else {
			a.log.Info("Syndesis resource already upgraded to version ", "name", syndesis.Name, "targetVersion", targetVersion)
			return a.completeUpgrade(ctx, syndesis, targetVersion)
		}
	}

	// Get the upgrade pod..
	upgradePod, err := util.GetPodWithLabelSelector(a.api, syndesis.Namespace, "syndesis.io/component=syndesis-upgrade,version="+targetVersion)
	if err != nil {

		// Lets try to create it...
		all, err := a.RenderUpgradeResources(ctx, syndesis)
		if err != nil {
			return err
		}

		// Install the upgrade resources.
		for _, res := range all {
			if res.GetKind() == "PersistentVolumeClaim" {
				res.SetNamespace(syndesis.Namespace)
			} else {
				operation.SetLabel(res, "version", targetVersion)
				operation.SetNamespaceAndOwnerReference(res, syndesis)
			}
			_, _, err := util.CreateOrUpdate(ctx, a.client, &res)
			if err != nil {
				a.log.Error(err, "Failed to create or replace resource", "kind", res.GetKind(), "name", res.GetName(), "namespace", res.GetNamespace())
				return err
			}
		}

		return nil // lets get called again... so we lookup the upgrade pod...
	}

	if upgradePod.Status.Phase == corev1.PodSucceeded {

		// Upgrade finished (correctly)
		a.upgradeResources(ctx, syndesis)

		a.log.Info("Syndesis resource upgraded", "name", syndesis.Name, "targetVersion", targetVersion)
		return a.completeUpgrade(ctx, syndesis, targetVersion)

		//if syndesis.Status.Version == targetVersion {
		//	a.log.Info("Upgrade pod terminated successfully but Syndesis version does not reflect target version. Forcing upgrade", "newVersion", syndesis.Status.Version, "targetVersion", targetVersion, "name", syndesis.Name)
		//
		//	var currentAttemptDescr string
		//	if syndesis.Status.UpgradeAttempts > 0 {
		//		currentAttemptDescr = " (attempt " + strconv.Itoa(int(syndesis.Status.UpgradeAttempts+1)) + ")"
		//	}
		//
		//	target := syndesis.DeepCopy()
		//	target.Status.ForceUpgrade = true
		//	target.Status.TargetVersion = targetVersion
		//	target.Status.Description = "Upgrading from " + syndesis.Status.Version + " to " + targetVersion + currentAttemptDescr
		//
		//	return a.client.Update(ctx, target)
		//}

	} else if upgradePod.Status.Phase == corev1.PodFailed {
		//Upgrade failed
		a.log.Error(nil, "Failure while upgrading Syndesis resource: upgrade pod failure", "name", syndesis.Name, "targetVersion", targetVersion)

		target := syndesis.DeepCopy()
		target.Status.Phase = v1alpha1.SyndesisPhaseUpgradeFailureBackoff
		target.Status.Reason = v1alpha1.SyndesisStatusReasonUpgradePodFailed
		target.Status.Description = "Syndesis upgrade from " + syndesis.Status.Version + " to " + targetVersion + " failed (it will be retried again)"
		target.Status.LastUpgradeFailure = &metav1.Time{
			Time: time.Now(),
		}
		target.Status.UpgradeAttempts = target.Status.UpgradeAttempts + 1

		return a.client.Update(ctx, target)
	} else {
		// Still running
		a.log.Info("Syndesis resource is currently being upgraded", "name", syndesis.Name, "targetVersion", targetVersion)
		return nil
	}
}

func (o *upgradeAction) upgradeResources(ctx context.Context, syndesis *v1alpha1.Syndesis) error {
	backupTypes := []metav1.TypeMeta{
		metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"},
		metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "RoleBinding"},
		metav1.TypeMeta{APIVersion: "template.openshift.io/v1", Kind: "Template"},
		metav1.TypeMeta{APIVersion: "build.openshift.io/v1", Kind: "BuildConfig"},
		metav1.TypeMeta{APIVersion: "apps.openshift.io/v1", Kind: "DeploymentConfig"},
		metav1.TypeMeta{APIVersion: "route.openshift.io/v1", Kind: "Route"},
	}

	selector, err := labels.Parse("syndesis.io/app=syndesis,syndesis.io/type=infrastructure")
	if err != nil {
		return err
	}

	for _, typeMeta := range backupTypes {
		options := client.ListOptions{
			Namespace:     syndesis.Namespace,
			LabelSelector: selector,
			Raw: &metav1.ListOptions{
				TypeMeta: typeMeta,
				Limit:    200,
			},
		}
		list := unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"apiVersion": typeMeta.APIVersion,
				"kind":       typeMeta.Kind,
			},
		}
		err = util.ListInChunks(ctx, o.client, &options, &list, func(resources []unstructured.Unstructured) error {
			for _, res := range resources {
				// Make sure we are the owners..
				operation.SetNamespaceAndOwnerReference(res, syndesis)
				_, _, err := util.CreateOrUpdate(ctx, o.client, &res)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	return nil
}

func (a *upgradeAction) RenderUpgradeResources(ctx context.Context, syndesis *v1alpha1.Syndesis) ([]unstructured.Unstructured, error) {

	secret := &corev1.Secret{}
	err := a.client.Get(ctx, types.NamespacedName{Namespace: syndesis.Namespace, Name: SyndesisPullSecret}, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			secret = nil
		} else {
			return nil, err
		}
	}
	renderContext, err := syndesistemplate.GetTemplateContext()
	if err != nil {
		return nil, err
	}
	config, err := configuration.GetSyndesisEnvVarsFromOpenshiftNamespace(ctx, a.client, syndesis.Namespace)
	if err != nil {
		config = map[string]string{}
	}
	err = syndesistemplate.SetupRenderContext(renderContext, syndesis, syndesistemplate.ResourceParams{}, config)
	if err != nil {
		return nil, err
	}
	if secret != nil {
		renderContext.ImagePullSecrets = append(renderContext.ImagePullSecrets, secret.Name)
	}

	configuration.SetConfigurationFromEnvVars(renderContext.Env, syndesis)

	err = checkTags(renderContext)
	if err != nil {
		return nil, err
	}

	if syndesis.Spec.DevImages {
		is := v1.ImageStream{}
		err := a.client.Get(ctx, client.ObjectKey{syndesis.Namespace, "syndesis-operator"}, &is)
		if err == nil {
			// if the image stream repo looks like a dev build: 172.30.1.1:5000/test2/syndesis-operator
			splits := strings.Split(is.Status.DockerImageRepository, "/")
			if len(splits) == 3 && splits[2] == "syndesis-operator" {
				syndesis.Spec.Registry = splits[0]
				renderContext.Images.SyndesisImagesPrefix = splits[1]
			}
		}
	}

	all, err := generator.RenderDir("./upgrade/", renderContext)
	if err != nil {
		return nil, err
	}
	return all, nil
}

func (a *upgradeAction) completeUpgrade(ctx context.Context, syndesis *v1alpha1.Syndesis, newVersion string) error {
	// After upgrade, pods may be detached
	if err := operation.AttachSyndesisToResource(ctx, a.scheme, a.client, syndesis); err != nil {
		return err
	}

	target := syndesis.DeepCopy()
	target.Status.Phase = v1alpha1.SyndesisPhaseInstalled
	target.Status.TargetVersion = ""
	target.Status.Reason = v1alpha1.SyndesisStatusReasonMissing
	target.Status.Description = ""
	target.Status.Version = newVersion
	target.Status.LastUpgradeFailure = nil
	target.Status.UpgradeAttempts = 0
	target.Status.ForceUpgrade = false

	return a.client.Update(ctx, target)
}

func (a *upgradeAction) getUpgradeResources(scheme *runtime.Scheme, syndesis *v1alpha1.Syndesis) ([]runtime.Object, error) {

	c, err := template.GetTemplateContext()

	unstructured, err := template.GetUpgradeResources(scheme, syndesis, template.ResourceParams{
		OAuthClientSecret: "-",
		UpgradeRegistry:   c.Registry,
	})
	if err != nil {
		return nil, err
	}

	structured := []runtime.Object{}
	structured, unstructured = util.SeperateStructuredAndUnstructured(a.scheme, unstructured)

	if len(unstructured) > 0 {
		return nil, fmt.Errorf("Could not convert some objects to runtime.Object")
	}
	return structured, nil
}

func (a *upgradeAction) findUpgradePod(resources []runtime.Object) (*corev1.Pod, error) {
	for _, res := range resources {
		if pod, ok := res.(*corev1.Pod); ok {
			if strings.HasPrefix(pod.Name, UpgradePodPrefix) {
				return pod, nil
			}
		}
	}
	return nil, errors.New("upgrade pod not found")
}

func (a *upgradeAction) getUpgradePodFromNamespace(ctx context.Context, podTemplate *corev1.Pod, syndesis *v1alpha1.Syndesis) (*corev1.Pod, error) {
	pod := corev1.Pod{}
	key := client.ObjectKey{
		Namespace: syndesis.Namespace,
		Name:      podTemplate.Name,
	}
	err := a.client.Get(ctx, key, &pod)
	return &pod, err
}

func getTypes(api kubernetes.Interface) ([]metav1.TypeMeta, error) {
	resources, err := api.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		return nil, err
	}

	types := make([]metav1.TypeMeta, 0)
	for _, resource := range resources {
		for _, r := range resource.APIResources {
			types = append(types, metav1.TypeMeta{
				Kind:       r.Kind,
				APIVersion: resource.GroupVersion,
			})
		}
	}

	return types, nil
}
