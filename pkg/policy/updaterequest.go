package policy

import (
	"context"
	"errors"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	kyvernov1beta1 "github.com/kyverno/kyverno/api/kyverno/v1beta1"
	common "github.com/kyverno/kyverno/pkg/background/common"
	"github.com/kyverno/kyverno/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

func newUR(pc *policyController, policy kyvernov1.PolicyInterface, trigger kyvernov1.ResourceSpec, ruleName string, ruleType kyvernov1beta1.RequestType, deleteDownstream bool) (*kyvernov1beta1.UpdateRequest, error) {
	var policyNameNamespaceKey string
	var ctx context.Context

	if policy.IsNamespaced() {
		policyNameNamespaceKey = policy.GetNamespace() + "/" + policy.GetName()
		isTerminating, err := namespaceIsTerminating(ctx, pc, policyNameNamespaceKey)
		if err != nil {
			return nil, errors.New("an error occurred in asserting the status of the namespace of the namespaced resource")
		} else if isTerminating {
			return nil, errors.New("namespace of the namespaced resource is of status 'terminating'")
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
	}, nil
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

func namespaceIsTerminating(ctx context.Context, pc *policyController, name string) (bool, error) {

	defer ctx.Done()

	// Should not be necessary, but for good measure
	getOpts := metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "Namespace",
		},
	}

	// Get the namespace object
	namespace, err := pc.client.GetKubeClient().CoreV1().Namespaces().Get(ctx, name, getOpts)

	// Check if it is in terminating status
	if err != nil {
		return false, err
	} else {
		if namespace.Status.String() == "terminating" {
			return true, nil
		} else {
			return false, nil
		}
	}

}
