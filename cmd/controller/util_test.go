package main

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestGetTeamNamespaceQ(t *testing.T) {
	team := newTeam("team1", "", "dev", corev1.ResourceQuotaSpec{})
	expected := "team-team1-dev"
	got := getTeamNamespace(team)
	if got != expected {
		t.Errorf("expected namespace %q, got %q", expected, got)
	}
}
