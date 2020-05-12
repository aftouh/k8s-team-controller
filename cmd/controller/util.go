package main

import (
	"bytes"
	"fmt"

	aftouhv1 "github.com/aftouh/k8s-sample-controller/pkg/apis/team/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	rqName = "team-default-rq"
)

func newResourceQuota(t *aftouhv1.Team) *corev1.ResourceQuota {
	return &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rqName,
			Namespace: getTeamNamespace(t),
			Labels:    getTeamLabels(t),
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

func missingLabels(t *aftouhv1.Team, obj metav1.Object) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return true
	}
	for k, v := range getTeamLabels(t) {
		v2, ok := labels[k]
		if !ok || (v != v2) {
			return true

		}
	}
	return false
}

func mergeLabels(t *aftouhv1.Team, obj metav1.Object) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range getTeamLabels(t) {
		labels[k] = v
	}
	obj.SetLabels(labels)
}

func (tc *TeamController) calculateTeamStatus(t *aftouhv1.Team) (aftouhv1.TeamStatus, error) {
	var ts aftouhv1.TeamStatus
	//Get namespace
	allNS, err := tc.nLister.List(labels.Everything())
	if err != nil {
		return ts, err
	}
	var ownedNS []*corev1.Namespace
	buf := new(bytes.Buffer)
	for _, ns := range allNS {
		if !metav1.IsControlledBy(ns, t) {
			continue
		}
		if len(ownedNS) > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(ns.Name)
		ownedNS = append(ownedNS, ns)
	}

	switch len(ownedNS) {
	case 0:
		return ts, nil
	case 1:
		ts.Namespace = ownedNS[0].Name
		rq, err := tc.rqLister.ResourceQuotas(ts.Namespace).Get(rqName)

		switch {
		case errors.IsNotFound(err):
			ts.ResourceQuota = ""
		case err != nil:
			return ts, fmt.Errorf("Unable to get ResourceQuota %s/%s from cache: %v", ts.Namespace, rq.Name, err)
		case !metav1.IsControlledBy(rq, t):
			ts.ResourceQuota = ""
		default:
			ts.ResourceQuota = rqName
		}
	default:
		return ts, fmt.Errorf("Team %q owns more than one namespace: %v", t.Name, string(buf.Bytes()))
	}

	return ts, nil
}
