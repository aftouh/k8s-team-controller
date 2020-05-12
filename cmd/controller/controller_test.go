package main

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	kinformers "k8s.io/client-go/informers"
	kfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"

	aftouhv1 "github.com/aftouh/k8s-sample-controller/pkg/apis/team/v1"
	tfake "github.com/aftouh/k8s-sample-controller/pkg/client/clientset/versioned/fake"
	tinformers "github.com/aftouh/k8s-sample-controller/pkg/client/informers/externalversions"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/diff"

	core "k8s.io/client-go/testing"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	tClientSet *tfake.Clientset
	kClientSet *kfake.Clientset

	// Objects to put in the store.
	tLister  []*aftouhv1.Team
	nLister  []*corev1.Namespace
	rqLister []*corev1.ResourceQuota

	// Actions expected to happen on the kubernetes client.
	kActions []core.Action
	// Actions expected to happen on the team client.
	tActions []core.Action

	// Objects from here preloaded into NewSimpleFake.
	kObjects []runtime.Object
	tObjects []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.tObjects = []runtime.Object{}
	f.kObjects = []runtime.Object{}
	return f
}

func (f *fixture) newTeamController() (*TeamController, tinformers.SharedInformerFactory, kinformers.SharedInformerFactory) {
	f.tClientSet = tfake.NewSimpleClientset(f.tObjects...)
	f.kClientSet = kfake.NewSimpleClientset(f.kObjects...)

	tInformer := tinformers.NewSharedInformerFactory(f.tClientSet, noResyncPeriodFunc())
	kInfomer := kinformers.NewSharedInformerFactory(f.kClientSet, noResyncPeriodFunc())

	tc := NewTeamController(f.tClientSet, f.kClientSet,
		tInformer.Aftouh().V1().Teams(),
		kInfomer.Core().V1().Namespaces(),
		kInfomer.Core().V1().ResourceQuotas())

	tc.tListerSynced = alwaysReady
	tc.nListerSynced = alwaysReady
	tc.rqListerSynced = alwaysReady

	tc.recorder = &record.FakeRecorder{}

	for _, t := range f.tLister {
		tInformer.Aftouh().V1().Teams().Informer().GetIndexer().Add(t)
	}

	for _, n := range f.nLister {
		kInfomer.Core().V1().Namespaces().Informer().GetIndexer().Add(n)
	}

	for _, rq := range f.rqLister {
		kInfomer.Core().V1().ResourceQuotas().Informer().GetIndexer().Add(rq)
	}

	return tc, tInformer, kInfomer
}

func (f *fixture) addObj(obj metav1.Object) {
	switch obj := obj.(type) {
	case *aftouhv1.Team:
		f.tLister = append(f.tLister, obj)
		f.tObjects = append(f.tObjects, obj)
	case *corev1.Namespace:
		f.nLister = append(f.nLister, obj)
		f.kObjects = append(f.kObjects, obj)
	case *corev1.ResourceQuota:
		f.rqLister = append(f.rqLister, obj)
		f.kObjects = append(f.kObjects, obj)
	}
}

func (f *fixture) run(teamName string) {
	f.runController(teamName, true, false)
}

func (f *fixture) runExpectError(teamName string) {
	f.runController(teamName, true, true)
}

func (f *fixture) runController(teamName string, startInformers bool, expectError bool) {
	tc, _, _ := f.newTeamController()

	err := tc.syncHandler(teamName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing team: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing team, got nil")
	}

	tActions := f.tClientSet.Actions()
	for i, action := range tActions {
		if len(f.tActions) < i+1 {
			f.t.Errorf("%d unexpected team actions: %+v", len(tActions)-len(f.tActions), tActions[i:])
			break
		}

		expectedAction := f.tActions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.tActions) > len(tActions) {
		f.t.Errorf("%d additional expected team actions:%+v", len(f.tActions)-len(tActions), f.tActions[len(tActions):])
	}

	kActions := f.kClientSet.Actions()
	for i, action := range kActions {
		if len(f.kActions) < i+1 {
			f.t.Errorf("%d unexpected kubernetes actions: %+v", len(kActions)-len(f.kActions), kActions[i:])
			break
		}

		expectedAction := f.kActions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.kActions) > len(kActions) {
		f.t.Errorf("%d additional expected kubernetes actions:%+v", len(f.kActions)-len(kActions), f.kActions[len(kActions):])
	}
}

// checkAction verifies that expected and actual actions are equal and both have
// same attached resources
func checkAction(expected, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateActionImpl:
		e, _ := expected.(core.CreateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.UpdateActionImpl:
		e, _ := expected.(core.UpdateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.PatchActionImpl:
		e, _ := expected.(core.PatchActionImpl)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, patch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expPatch, patch))
		}
	default:
		t.Errorf("Uncaptured Action %s %s, you should explicitly add a case to capture it",
			actual.GetVerb(), actual.GetResource().Resource)
	}
}

func (f *fixture) expectCreateNamespaceAction(n *corev1.Namespace) {
	f.kActions = append(f.kActions, core.NewRootCreateAction(schema.GroupVersionResource{Resource: "namespaces"}, n))
}

func (f *fixture) expectCreateResourceQuotaAction(rq *corev1.ResourceQuota) {
	f.kActions = append(f.kActions, core.NewCreateAction(schema.GroupVersionResource{Resource: "resourcequotas"}, rq.Namespace, rq))
}

func (f *fixture) expectUpdateNamespaceAction(n *corev1.Namespace) {
	f.kActions = append(f.kActions, core.NewRootUpdateAction(schema.GroupVersionResource{Resource: "namespaces"}, n))
}

func (f *fixture) expectUpdateResourceQuotaAction(rq *corev1.ResourceQuota) {
	f.kActions = append(f.kActions, core.NewUpdateAction(schema.GroupVersionResource{Resource: "resourcequotas"}, rq.Namespace, rq))
}

func (f *fixture) expectUpdateTeamStatus(t *aftouhv1.Team) {
	f.tActions = append(f.tActions, core.NewRootUpdateAction(schema.GroupVersionResource{
		Resource: "teams",
		Group:    aftouhv1.SchemeGroupVersion.Group,
		Version:  aftouhv1.SchemeGroupVersion.Version,
	}, t))
}

func TestCreateNamespace(t *testing.T) {
	f := newFixture(t)
	team := newTeam("test", "test desciption", "dev", corev1.ResourceQuotaSpec{})
	f.addObj(team)

	f.expectCreateNamespaceAction(newNamespace(team))

	//We expect error because the new namespace is not visible by lister
	//so resourcequota syncing return not found error
	f.runExpectError(team.Name)
}

func TestCreateResourceQuota(t *testing.T) {
	f := newFixture(t)

	team := newTeam("test", "test desciption", "dev", corev1.ResourceQuotaSpec{})

	f.addObj(team)
	ns := newNamespace(team)
	ns.Status.Phase = corev1.NamespaceActive
	f.addObj(ns)

	f.expectCreateResourceQuotaAction(newResourceQuota(team))

	team.Status.Namespace = "team-test-dev"
	f.expectUpdateTeamStatus(team)

	f.run(team.Name)
}

func TestDeletedTeam(t *testing.T) {
	f := newFixture(t)
	team := newTeam("test", "test desciption", "dev", corev1.ResourceQuotaSpec{})
	f.run(team.Name)
}

func TestUpdateNamespaceLabels(t *testing.T) {
	f := newFixture(t)

	//Create team
	team := newTeam("test", "test desciption", "dev", corev1.ResourceQuotaSpec{})
	f.addObj(team)

	//Create namespace with invalid labels
	ns := newNamespace(team)
	ns.Labels["env"] = "prod"
	ns.Labels["other"] = "other"
	ns.Status.Phase = corev1.NamespaceActive
	f.addObj(ns)

	//Create team RS
	rq := newResourceQuota(team)
	f.addObj(rq)

	//expect rq update
	expectedNS := newNamespace(team)
	expectedNS.Labels["other"] = "other"
	expectedNS.Status.Phase = corev1.NamespaceActive
	f.expectUpdateNamespaceAction(expectedNS)

	team.Status.Namespace = "team-test-dev"
	team.Status.ResourceQuota = rqName
	f.expectUpdateTeamStatus(team)

	f.run(team.Name)
}

func TestUpdateRQLabels(t *testing.T) {
	f := newFixture(t)

	//Create team
	team := newTeam("test", "test desciption", "dev", corev1.ResourceQuotaSpec{})
	f.addObj(team)

	//Create team
	f.addObj(newNamespace(team))

	//Create namespace with invalid labels
	rq := newResourceQuota(team)
	rq.Labels["env"] = "prod"
	rq.Labels["other"] = "other"
	f.addObj(rq)

	//expect rq update
	expectedNS := newResourceQuota(team)
	expectedNS.Labels["other"] = "other"
	f.expectUpdateResourceQuotaAction(expectedNS)

	team.Status.Namespace = "team-test-dev"
	team.Status.ResourceQuota = rqName
	f.expectUpdateTeamStatus(team)

	f.run(team.Name)
}

func TestUpdateRQSpec(t *testing.T) {
	f := newFixture(t)

	//Create team
	team := newTeam("test", "test desciption", "dev", corev1.ResourceQuotaSpec{
		Hard: corev1.ResourceList{
			corev1.ResourceCPU: *resource.NewQuantity(4, resource.DecimalSI),
		},
	})
	f.addObj(team)

	//Create team
	f.addObj(newNamespace(team))

	//Create namespace with invalid labels
	rq := newResourceQuota(team)
	rq.Spec = corev1.ResourceQuotaSpec{
		Hard: corev1.ResourceList{
			corev1.ResourceCPU: *resource.NewQuantity(5, resource.DecimalSI),
		},
	}
	f.addObj(rq)

	//expect rq update
	expectedNS := newResourceQuota(team)
	f.expectUpdateResourceQuotaAction(expectedNS)

	team.Status.Namespace = "team-test-dev"
	team.Status.ResourceQuota = rqName
	f.expectUpdateTeamStatus(team)

	f.run(team.Name)
}
