package main

import (
	"fmt"

	aftouhv1 "github.com/aftouh/k8s-sample-controller/pkg/apis/team/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	rqName = "team-default-rq"
)

func newResourceQuota(t *aftouhv1.Team) *corev1.ResourceQuota {
	return &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rqName,
			Namespace: getTeamNamespace(t),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(t, aftouhv1.SchemeGroupVersion.WithKind("Team")),
			},
		},
		Spec: t.Spec.ResourceQuotaSpec,
	}
}

func newNamespace(t *aftouhv1.Team) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   getTeamNamespace(t),
			Labels: getTeamLabels(t),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(t, aftouhv1.SchemeGroupVersion.WithKind("Team")),
			},
		},
		Spec: corev1.NamespaceSpec{},
	}
}

func newTeam(name, description, environment string, rqSpec corev1.ResourceQuotaSpec) *aftouhv1.Team {
	return &aftouhv1.Team{
		TypeMeta: metav1.TypeMeta{APIVersion: aftouhv1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: aftouhv1.TeamSpec{
			Name:              name,
			Description:       description,
			Environment:       environment,
			ResourceQuotaSpec: rqSpec,
		},
	}
}

func getTeamNamespace(t *aftouhv1.Team) string {
	namespaceFormat := "team-%s-%s"
	return fmt.Sprintf(namespaceFormat, t.Spec.Name, t.Spec.Environment)
}

func getTeamLabels(t *aftouhv1.Team) map[string]string {
	return map[string]string{
		"team": t.Spec.Name,
		"env":  t.Spec.Environment,
	}
}
