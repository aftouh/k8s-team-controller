package main

import (
	aftouh "github.com/aftouh/k8s-sample-controller/pkg/apis/team/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
)

func (tc *TeamController) sync(t *aftouh.Team) error {

	_, err := tc.nLister.Get(t.Spec.Name)

	//Namespace does not exist. Create it
	if errors.IsNotFound(err) {
		_, err := tc.kClientSet.CoreV1().Namespaces().Create(tc.newNamespace(t))
		return err
	}

	return err
}

func (tc *TeamController) newNamespace(t *aftouh.Team) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: t.Spec.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(t, aftouh.SchemeGroupVersion.WithKind("Team")),
			},
		},
		Spec: corev1.NamespaceSpec{},
	}
}
