package policy

import (
	"context"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	kyvernov1beta1 "github.com/kyverno/kyverno/api/kyverno/v1beta1"
	common "github.com/kyverno/kyverno/pkg/background/common"
	"github.com/kyverno/kyverno/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

func newUR(pc *policyController, policy kyvernov1.PolicyInterface, trigger kyvernov1.ResourceSpec, ruleName string, ruleType kyvernov1beta1.RequestType, deleteDownstream bool) *kyvernov1beta1.UpdateRequest {
	var policyNameNamespaceKey string
	var ctx context.Context

	if policy.IsNamespaced() {
		policyNameNamespaceKey = policy.GetNamespace() + "/" + policy.GetName()
		isTerminating, _ := isTerminating(ctx, pc, policyNameNamespaceKey)
		if isTerminating {
			//TODO: How do we handle this error
		}

	} else {
		policyNameNamespaceKey = policy.GetName()
	}

	var label labels.Set
	if ruleType == kyvernov1beta1.Mutate {
		label = common.MutateLabelsSet(policyNameNamespaceKey, trigger)
	} else {
		label = common.GenerateLabelsSet(policyNameNamespaceKey, trigger)
	}

	return &kyvernov1beta1.UpdateRequest{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kyvernov1beta1.SchemeGroupVersion.String(),
			Kind:       "UpdateRequest",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ur-",
			Namespace:    config.KyvernoNamespace(),
			Labels:       label,
		},
		Spec: kyvernov1beta1.UpdateRequestSpec{
			Type:   ruleType,
			Policy: policyNameNamespaceKey,
			Rule:   ruleName,
			Resource: kyvernov1.ResourceSpec{
				Kind:       trigger.GetKind(),
				Namespace:  trigger.GetNamespace(),
				Name:       trigger.GetName(),
				APIVersion: trigger.GetAPIVersion(),
				UID:        trigger.GetUID(),
			},
			DeleteDownstream: deleteDownstream,
		},
	}
}

func newURStatus(downstream unstructured.Unstructured) kyvernov1beta1.UpdateRequestStatus {

	return kyvernov1beta1.UpdateRequestStatus{
		State: kyvernov1beta1.Pending,
		GeneratedResources: []kyvernov1.ResourceSpec{
			{
				APIVersion: downstream.GetAPIVersion(),
				Kind:       downstream.GetKind(),
				Namespace:  downstream.GetNamespace(),
				Name:       downstream.GetName(),
				UID:        downstream.GetUID(),
			},
		},
	}
}

func isTerminating(ctx context.Context, pc *policyController, name string) (bool, error) {

	getOpts := metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "Namespace",
		},
	}

	namespace, err := pc.client.GetKubeClient().CoreV1().Namespaces().Get(ctx, name, getOpts)

	if err != nil {
		return true, err
	} else {
		if namespace.Status.String() == "terminating" {
			return true, nil
		} else {
			return false, nil
		}
	}

}
