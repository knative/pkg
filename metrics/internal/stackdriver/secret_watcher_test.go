/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package stackdriver

import (
	"fmt"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fclient "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	defaultSecretDataKey   = "key.json"
	defaultSecretDataValue = "token"
	defaultSecretName      = "test"
	defaultSecretNamespace = "test-secret"
)

var (
	// kubeclientForTest is a fake kubeclient that can be use for testing.
	// This should be passed to any functionality being tested that require a kubeclient.
	kubeclientForTest kubernetes.Interface

	// dsft is a default secret for tests, for convenience.
	dsft = newSecretForTest(defaultSecretName, defaultSecretNamespace)
	// defaultNumTestObservers is the default number of observer callbacks for tests.
	defaultNumTestObservers = 2

	defaultGlobalSecretWatcherConstructor = func(t *testing.T, obs ...Observer) SecretWatcher {
		return mustNewSecretWatcher(t, obs...)
	}

	defaultSingleNamespaceSecretWatcherConstructor = func(t *testing.T, obs ...Observer) SecretWatcher {
		return mustNewSecretWatcherSingleNamespace(t, dsft.namespace, obs...)
	}

	defaultSingleSecretSecretWatcherConstructor = func(t *testing.T, obs ...Observer) SecretWatcher {
		return mustNewSecretWatcherSingleSecret(t, dsft.namespace, dsft.name, obs...)
	}

	defaultWatchersToTest = []struct {
		name        string
		constructor func(t *testing.T, obs ...Observer) SecretWatcher
	}{
		{
			name:        "GlobalSecretWatcher",
			constructor: defaultGlobalSecretWatcherConstructor,
		},
		{
			name:        "SingleNamespaceSecretWatcher",
			constructor: defaultSingleSecretSecretWatcherConstructor,
		},
		{
			name:        "SingleSecretSecretWatcher",
			constructor: defaultSingleSecretSecretWatcherConstructor,
		},
	}

	defaultObserverFuncs = &ObserverFuncs{
		AddFunc: func(s *corev1.Secret) {
		},
		UpdateFunc: func(sOld *corev1.Secret, sNew *corev1.Secret) {
		},
		DeleteFunc: func(s *corev1.Secret) {
		},
	}
)

// secretForTest encapsulates Secret metadata and a Secret for re-using and verifying values.
type secretForTest struct {
	namespace string
	name      string
	dataKey   string
	dataValue string
	secret    corev1.Secret
}

func (s *secretForTest) UpdateDataKey(dataKey string) *secretForTest {
	s.dataKey = dataKey
	s.secret.Data = map[string][]byte{
		s.dataKey: []byte(s.dataValue),
	}

	return s
}

func (s *secretForTest) UpdateDataValue(dataValue string) *secretForTest {
	s.dataValue = dataValue
	s.secret.Data = map[string][]byte{
		s.dataKey: []byte(s.dataValue),
	}

	return s
}

// newSecretForTest constructs a secretForTest.
func newSecretForTest(namespace string, name string) *secretForTest {
	return &secretForTest{
		namespace: namespace,
		name:      name,
		dataKey:   defaultSecretDataKey,
		dataValue: defaultSecretDataValue,
		secret: corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				defaultSecretDataKey: []byte(defaultSecretDataValue),
			},
			Type: "Opaque",
		},
	}
}

// testObserver is an implementation of the the secretWatcher observer interface
// that also holds state about whether the observer's callbacks have been triggered.
type testObserver struct {
	oFuncs         *ObserverFuncs
	onAddCalled    bool
	onUpdateCalled bool
	onDeleteCalled bool
	mux            sync.Mutex
}

// newTestObserver constructs a testObserver from observerFuncs.
func newTestObserver(inputObsFuncs *ObserverFuncs) *testObserver {
	// If a callback is nil, consider it to be called by default.
	return &testObserver{
		oFuncs:         inputObsFuncs,
		onAddCalled:    inputObsFuncs.AddFunc == nil,
		onUpdateCalled: inputObsFuncs.UpdateFunc == nil,
		onDeleteCalled: inputObsFuncs.DeleteFunc == nil,
	}
}

func (tObs *testObserver) markCallbackCalled(callbackSentinel *bool) {
	tObs.mux.Lock()
	defer tObs.mux.Unlock()
	*callbackSentinel = true
}

func (tObs *testObserver) OnAdd(s *corev1.Secret) {
	tObs.oFuncs.OnAdd(s)
	tObs.markCallbackCalled(&tObs.onAddCalled)
}

func (tObs *testObserver) OnUpdate(sOld *corev1.Secret, sNew *corev1.Secret) {
	tObs.oFuncs.OnUpdate(sOld, sNew)
	tObs.markCallbackCalled(&tObs.onUpdateCalled)
}

func (tObs *testObserver) OnDelete(s *corev1.Secret) {
	tObs.oFuncs.OnDelete(s)
	tObs.markCallbackCalled(&tObs.onDeleteCalled)
}

func (tObs *testObserver) CheckOnAddCalled() bool {
	return tObs.wasCallbackCalled(&tObs.onAddCalled)
}

func (tObs *testObserver) CheckOnUpdateCalled() bool {
	return tObs.wasCallbackCalled(&tObs.onUpdateCalled)
}

func (tObs *testObserver) CheckOnDeleteCalled() bool {
	return tObs.wasCallbackCalled(&tObs.onDeleteCalled)
}

func (tObs *testObserver) wasCallbackCalled(callbackSentinel *bool) bool {
	tObs.mux.Lock()
	defer tObs.mux.Unlock()
	return *callbackSentinel
}

func (tObs *testObserver) AllCallbacksCalled() bool {
	tObs.mux.Lock()
	defer tObs.mux.Unlock()
	return tObs.onAddCalled && tObs.onUpdateCalled && tObs.onDeleteCalled
}

func testSetupWithCustomKubeclient(customKubeclient kubernetes.Interface) {
	// Reset kubeclient on every test to clear out state
	kubeclientForTest = customKubeclient
	// Ensure test and secret watcher share the same kubeclient
	createStackdriverKubeclientFunc = func() (kubernetes.Interface, error) {
		return kubeclientForTest, nil
	}

	// Always reset secret
	dsft = newSecretForTest(defaultSecretName, defaultSecretNamespace)
}

// testSetup sets up tests.
func testSetup() {
	testSetupWithCustomKubeclient(fclient.NewSimpleClientset())
}

// testCleanup cleans up tests.
func testCleanup() {
	createStackdriverKubeclientFunc = createKubeclient
}

func TestNewSecretWatcher(t *testing.T) {
	for _, test := range defaultWatchersToTest {
		t.Run(test.name, func(t *testing.T) {
			testSetup()
			test.constructor(t, defaultObserverFuncs)
			testCleanup()
		})
	}
}

func TestStartStopWatch(t *testing.T) {
	o := &ObserverFuncs{
		AddFunc: func(s *corev1.Secret) {},
	}

	for _, test := range defaultWatchersToTest {
		t.Run(test.name, func(t *testing.T) {
			testSetup()

			tObs := newTestObserver(o)
			watcher := test.constructor(t, tObs)
			// Test weird patterns of stop/start
			watcher.StopWatch()
			watcher.StartWatch()
			watcher.StartWatch()
			watcher.StopWatch()
			watcher.StopWatch()

			watcher.StartWatch()
			defer watcher.StopWatch()

			kubeclientForTest.CoreV1().Secrets(dsft.namespace).Create(&dsft.secret)
			waitForCondition(t, func() bool {
				return tObs.CheckOnAddCalled()
			}, 3)

			testCleanup()
		})
	}
}

func TestNilObserverFuncs(t *testing.T) {
	testSetup()
	defer testSetup()

	// Setup two watchers that watch the same Secret, but only have a subset of callbacks
	o := &ObserverFuncs{
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc: nil,
	}

	// The testObserver wrapper struct will invoke the funcs from "o" and track whether they were called
	tObs := newTestObserver(o)
	watcher := defaultGlobalSecretWatcherConstructor(t, tObs)

	watcher.StartWatch()
	defer watcher.StopWatch()

	kubeclientForTest.CoreV1().Secrets(dsft.namespace).Create(&dsft.secret)
	waitForCondition(t, func() bool {
		return tObs.CheckOnAddCalled()
	}, 3)

	dsft.UpdateDataValue("newToken")
	kubeclientForTest.CoreV1().Secrets(dsft.namespace).Update(&dsft.secret)
	waitForCondition(t, func() bool {
		return tObs.CheckOnUpdateCalled()
	}, 3)

	kubeclientForTest.CoreV1().Secrets(dsft.namespace).Delete(dsft.name, &metav1.DeleteOptions{})
	waitForCondition(t, func() bool {
		return tObs.CheckOnDeleteCalled()
	}, 3)

	testCleanup()
}

func TestSecretWatcherCallbacks(t *testing.T) {
	nsApple := "apple"
	nsOrange := "orange"
	nameOne := "one"
	nameTwo := "two"

	// These secrets will be created in the order they are declared
	sOrangeOne := newSecretForTest(nsOrange, nameOne)
	sAppleOne := newSecretForTest(nsApple, nameOne)
	sAppleTwo := newSecretForTest(nsApple, nameTwo)

	globalSecretWatcherConstructor := func(obs ...Observer) SecretWatcher { return mustNewSecretWatcher(t, obs...) }
	appleNamespaceWatcherConstructor := func(obs ...Observer) SecretWatcher {
		return mustNewSecretWatcherSingleNamespace(t, sAppleOne.namespace, obs...)
	}
	appleTwoSecretWatcherConstructor := func(obs ...Observer) SecretWatcher {
		return mustNewSecretWatcherSingleSecret(t, sAppleTwo.namespace, sAppleTwo.name, obs...)
	}

	// When using the fake kubernetes client to list resources, label and field filters passed through meta ListOptions are not applied.
	// The filtering is normally done server side and is not implemented on the fake kubernetes client (https://github.com/kubernetes/client-go/issues/326).
	// Modify the fake kubernetes client to fake out list filtering for this one test, it specifically filters out Secret named "apple/one".
	appleNameTwoSecretWatcherChangeAppleOneTestSetup := func() {
		f := fclient.NewSimpleClientset()
		// Intercept all list actions by the fake kubeclient and modify the return.
		// List is used by SharedInformers for detecting OnAdd and OnUpdate notifications.
		f.Fake.PrependReactor("list", "secrets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			listAction := action.(k8stesting.ListAction)
			listRestrictions := listAction.GetListRestrictions()
			// There's no way to get the actual results of the list action, but we can check that the list action was for the "apple" namespace.
			// And verify that the SharedInformer passed ListOptions that filtered by "metadata.name".
			if listAction.GetNamespace() == nsApple && listRestrictions.Fields.String() == fmt.Sprintf("metadata.name=%v", nameTwo) {
				secrets := &corev1.SecretList{
					TypeMeta: metav1.TypeMeta{
						Kind:       "SecretList",
						APIVersion: "apps/v1",
					},
					Items: []corev1.Secret{}, // omit "apple/one"
				}
				return true, secrets, nil
			}

			return false, nil, nil
		})

		// Intercept all delete actions by the fake kubeclient and modify the return.
		// Delete is used by SharedInformers for detecting OnDelete notifications.
		f.Fake.PrependReactor("delete", "secrets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			act := action.(k8stesting.DeleteAction)
			ns := act.GetNamespace()
			name := act.GetName()

			// Omit "apple/one"
			if ns == nsApple && name == nameOne {
				return true, nil, nil
			}

			return false, nil, nil
		})

		testSetupWithCustomKubeclient(f)
	}

	var tests = []struct {
		name                 string
		testSetupFunc        func()
		watcherConstructor   func(observers ...Observer) SecretWatcher
		secretToCreate       *secretForTest
		shouldTriggerWatcher bool
	}{
		{
			name:                 "GlobalWatcherChangeOrangeOne",
			watcherConstructor:   globalSecretWatcherConstructor,
			secretToCreate:       sOrangeOne,
			shouldTriggerWatcher: true,
		},
		{
			name:                 "GlobalWatcherChangeAppleOne",
			watcherConstructor:   globalSecretWatcherConstructor,
			secretToCreate:       sAppleOne,
			shouldTriggerWatcher: true,
		},
		{
			name:                 "GlobalWatcherChangeAppleTwo",
			watcherConstructor:   globalSecretWatcherConstructor,
			secretToCreate:       sAppleTwo,
			shouldTriggerWatcher: true,
		},
		{
			name:                 "AppleNamespaceWatcherChangeOrangeOne",
			watcherConstructor:   appleNamespaceWatcherConstructor,
			secretToCreate:       sOrangeOne,
			shouldTriggerWatcher: false,
		},
		{
			name:                 "AppleNamespaceWatcherChangeAppleOne",
			watcherConstructor:   appleNamespaceWatcherConstructor,
			secretToCreate:       sAppleOne,
			shouldTriggerWatcher: true,
		},
		{
			name:                 "AppleNamespaceWatcherChangeAppleTwo",
			watcherConstructor:   appleNamespaceWatcherConstructor,
			secretToCreate:       sAppleTwo,
			shouldTriggerWatcher: true,
		},
		{
			name:                 "AppleNameTwoSecretWatcherChangeOrangeOne",
			watcherConstructor:   appleTwoSecretWatcherConstructor,
			secretToCreate:       sOrangeOne,
			shouldTriggerWatcher: false,
		},
		{
			name:                 "AppleNameTwoSecretWatcherChangeAppleOne",
			testSetupFunc:        appleNameTwoSecretWatcherChangeAppleOneTestSetup,
			watcherConstructor:   appleTwoSecretWatcherConstructor,
			secretToCreate:       sAppleOne,
			shouldTriggerWatcher: false,
		},
		{
			name:                 "AppleNameTwoSecretWatcherChangeAppleTwo",
			watcherConstructor:   appleTwoSecretWatcherConstructor,
			secretToCreate:       sAppleTwo,
			shouldTriggerWatcher: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.testSetupFunc != nil {
				test.testSetupFunc()
			} else {
				testSetup()
			}

			obsFuncs := &ObserverFuncs{
				AddFunc:    func(s *corev1.Secret) {},
				UpdateFunc: func(sOld *corev1.Secret, sNew *corev1.Secret) {},
				DeleteFunc: func(s *corev1.Secret) {},
			}

			kubeclientForTest.CoreV1().Secrets(test.secretToCreate.namespace).Create(&test.secretToCreate.secret)

			testObs := make([]*testObserver, defaultNumTestObservers)
			watchers := make([]SecretWatcher, defaultNumTestObservers)
			for i := 0; i < defaultNumTestObservers; i++ {
				testObs[i] = newTestObserver(obsFuncs)
				watchers[i] = test.watcherConstructor(testObs[i])
				watchers[i].StartWatch()
			}

			// SecretWatchers should get a create/add notification when the watch starts for every existing Secret they are watching
			waitForObserverTriggers(t, testObs, func(tObs *testObserver) bool { return tObs.CheckOnAddCalled() }, test.shouldTriggerWatcher)

			kubeclientForTest.CoreV1().Secrets(test.secretToCreate.namespace).Delete(test.secretToCreate.name, &metav1.DeleteOptions{})
			waitForObserverTriggers(t, testObs, func(tObs *testObserver) bool { return tObs.CheckOnDeleteCalled() }, test.shouldTriggerWatcher)

			kubeclientForTest.CoreV1().Secrets(test.secretToCreate.namespace).Create(&test.secretToCreate.secret)
			waitForObserverTriggers(t, testObs, func(tObs *testObserver) bool { return tObs.CheckOnAddCalled() }, test.shouldTriggerWatcher)

			// Modify secret value and update
			test.secretToCreate.UpdateDataValue("newToken")
			kubeclientForTest.CoreV1().Secrets(test.secretToCreate.namespace).Update(&test.secretToCreate.secret)
			waitForObserverTriggers(t, testObs, func(tObs *testObserver) bool { return tObs.CheckOnUpdateCalled() }, test.shouldTriggerWatcher)

			for i := 0; i < defaultNumTestObservers; i++ {
				watchers[i].StopWatch()
			}

			testCleanup()
		})
	}
}

// waitForObserverTriggers is a helper function for testing multiple observers and minimizing the time spent waiting.
// If shouldTrigger is true, the function waits up to 1 second for obsTrigger to be true for each *testObserver in tObs, otherwise the test errors.
// If shouldTrigger is false, the function waits 1 second and then ensures that obsTrigger is false, otherwise the test errors.
func waitForObserverTriggers(t *testing.T, tObs []*testObserver, obsTrigger func(*testObserver) bool, shouldTrigger bool) {
	if shouldTrigger {
		// Waits up to 1 second, but can return instantaneously.
		waitForCondition(t, func() bool {
			for _, o := range tObs {
				if !obsTrigger(o) {
					return false
				}
			}
			return true
		}, 1)
	} else {
		// Wait 1 second to give triggers time to trigger, otherwise assume the triggers will never trigger.
		time.Sleep(1 * time.Second)
		for i, o := range tObs {
			if obsTrigger(o) {
				t.Errorf("Secret watcher at idx[%d] did not trigger as expected on Secret change. Expected to trigger? %v. OnAdd triggered? %v. OnUpdate triggered? %v. OnDelete triggered? %v",
					i, shouldTrigger, o.onAddCalled, o.onUpdateCalled, o.onDeleteCalled)
			}
		}
	}
}

func waitForCondition(t *testing.T, condition func() bool, timeoutSec int) {
	// To speed up tests when condition() becomes true quickly, sleep for a short period of time before starting normal wait loop.
	// 10ms was selected by empirical testing of what made the tests pass the fastest.
	time.Sleep(time.Millisecond * 10)

	for i := 0; i < (timeoutSec*2)+1; i++ {
		if condition() {
			return
		}
		// Relatively long sleep period so in cases when condition() is never true, it's not checked too often.
		time.Sleep(time.Millisecond * 500)
	}

	t.Errorf("Timed out waiting for condition to become true, checked every 0.5s for %d seconds", timeoutSec)
}

func mustNewSecretWatcher(t *testing.T, observers ...Observer) SecretWatcher {
	return mustNewSecretWatcherHelper(t, func() (SecretWatcher, error) { return NewSecretWatcher(observers...) })
}

func mustNewSecretWatcherSingleNamespace(t *testing.T, namespace string, observers ...Observer) SecretWatcher {
	return mustNewSecretWatcherHelper(t, func() (SecretWatcher, error) { return NewSecretWatcherSingleNamespace(namespace, observers...) })
}

func mustNewSecretWatcherSingleSecret(t *testing.T, namespace string, name string, observers ...Observer) SecretWatcher {
	return mustNewSecretWatcherHelper(t, func() (SecretWatcher, error) { return NewSecretWatcherSingleSecret(namespace, name, observers...) })
}

func mustNewSecretWatcherHelper(t *testing.T, constructor func() (SecretWatcher, error)) SecretWatcher {
	watcher, err := constructor()
	if err != nil {
		t.Errorf("Failed to create secret watcher: %v", err)
	}

	return watcher
}

func assertSecretValue(t *testing.T, secret *corev1.Secret, namespace string, name string, expectedDataKey string, expectedDataValue string) {
	if val, ok := secret.Data[expectedDataKey]; !ok {
		t.Errorf("Retrieved incorrect Secret data for Secret %v/%v from secret watcher. Expected data to contain field '%v'. Got secret data: %v", namespace, name, expectedDataKey, secret.Data)
	} else if string(val) != expectedDataValue {
		t.Errorf("Retrieved incorrect Secret data for Secret %v/%v from secret watcher. Expected field '%v' to have value '%v'. Got secret data: %v", namespace, name, expectedDataKey, []byte(expectedDataValue), secret.Data)
	}
}
