package main

import (
	"fmt"

	aftouh "github.com/aftouh/k8s-sample-controller/pkg/apis/team/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	rqName = "team-default-rq"
)

func newResourceQuota(t *aftouh.Team) *corev1.ResourceQuota {
	return &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: rqName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(t, aftouh.SchemeGroupVersion.WithKind("Team")),
			},
		},
		Spec: t.Spec.ResourceQuotaSpec,
	}
}

func newNamespace(t *aftouh.Team) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   getTeamNamespace(t),
			Labels: getTeamLabels(t),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(t, aftouh.SchemeGroupVersion.WithKind("Team")),
			},
		},
		Spec: corev1.NamespaceSpec{},
	}
}

func getTeamNamespace(t *aftouh.Team) string {
	namespaceFormat := "team-%s-%s"
	return fmt.Sprintf(namespaceFormat, t.Spec.Name, t.Spec.Environment)
}

func getTeamLabels(t *aftouh.Team) map[string]string {
	return map[string]string{
		"team": t.Spec.Name,
		"env":  t.Spec.Environment,
	}
}
