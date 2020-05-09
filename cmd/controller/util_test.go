package main

import (
	"testing"

	aftouhv1 "github.com/aftouh/k8s-sample-controller/pkg/apis/team/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func TestGetTeamNamespaceQ(t *testing.T) {
	team := newTeam("team1", "", "dev", corev1.ResourceQuotaSpec{})
	expected := "team-team1-dev"
	got := getTeamNamespace(team)
	if got != expected {
		t.Errorf("expected namespace %q, got %q", expected, got)
	}
}
