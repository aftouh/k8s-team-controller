package main

import (
	"fmt"

	aftouh "github.com/aftouh/k8s-sample-controller/pkg/apis/team/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	rqName = "team-default-rq"

	messageResourceExists = "Resource %q already exists and is not managed by Team"
	errResourceExists     = "ErrResourceExists"
)

func (tc *TeamController) sync(t *aftouh.Team) error {

	if err := tc.syncNamespace(t); err != nil {
		return err
	}

	if err := tc.syncResourceQuota(t); err != nil {
		return err
	}

	return nil
}

func (tc *TeamController) syncResourceQuota(t *aftouh.Team) error {
	namespaceName := getTeamNamespace(t)
	rq, err := tc.rqLister.ResourceQuotas(namespaceName).Get(rqName)

	//ResourceQuota does not exist. Need to be created
	if errors.IsNotFound(err) {
		klog.V(4).Infof("Creating resourceQuota %q", rqName)
		_, err = tc.kClientSet.CoreV1().ResourceQuotas(namespaceName).Create(newResourceQuota(t))
		return err
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(rq, t) {
		msg := fmt.Sprintf(messageResourceExists, rq.Name)
		tc.recorder.Event(t, corev1.EventTypeWarning, errResourceExists, msg)
		return fmt.Errorf(msg)
	}
	return nil
}

func (tc *TeamController) syncNamespace(t *aftouh.Team) error {

	namespaceName := getTeamNamespace(t)
	namespace, err := tc.nLister.Get(namespaceName)

	//Namespace does not exist. Need to be created
	if errors.IsNotFound(err) {
		klog.V(4).Infof("Creating namespace %q", namespaceName)
		_, err = tc.kClientSet.CoreV1().Namespaces().Create(newNamespace(t))
		return err
	}

	if err != nil {
		return err
	}

	// Namespace should be created by this controller
	if !metav1.IsControlledBy(namespace, t) {
		msg := fmt.Sprintf(messageResourceExists, namespace.Name)
		tc.recorder.Event(t, corev1.EventTypeWarning, errResourceExists, msg)
		return fmt.Errorf(msg)
	}

	return nil
}

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
