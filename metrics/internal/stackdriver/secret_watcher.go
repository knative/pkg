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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// Observer is the interface for callbacks that notify an Observer of the latest
// state of a secret being watched by a secretWatcher.  An Observer should not modify the provided
// Secrets, and should `.DeepCopy()` it for persistence (or otherwise process its
// contents).
type Observer interface {
	OnAdd(*corev1.Secret)
	OnUpdate(*corev1.Secret, *corev1.Secret)
	OnDelete(*corev1.Secret)
}

// ObserverFuncs implements the observer interface.
type ObserverFuncs struct {
	AddFunc    func(*corev1.Secret)
	UpdateFunc func(*corev1.Secret, *corev1.Secret)
	DeleteFunc func(*corev1.Secret)
}

// OnAdd is the OnAdd of the observer interface.
func (of *ObserverFuncs) OnAdd(s *corev1.Secret) {
	// if of.AddFunc == nil {
	// 	return
	// }
	of.AddFunc(s)
}

// OnUpdate is the OnUpdate of the observer interface.
func (of *ObserverFuncs) OnUpdate(sOld *corev1.Secret, sNew *corev1.Secret) {
	// if of.UpdateFunc == nil {
	// 	return
	// }
	of.UpdateFunc(sOld, sNew)
}

// OnDelete is the OnDelete of the observer interface.
func (of *ObserverFuncs) OnDelete(s *corev1.Secret) {
	// if of.DeleteFunc == nil {
	// 	return
	// }
	of.DeleteFunc(s)
}

// SecretWatcher defines the interface that a secret watcher must implement.
type SecretWatcher interface {
	// StartWatch starts the secret watch allowing observer callbacks to be triggered.
	StartWatch() error

	// StopWatch stops the secret watch.
	StopWatch()

	// GetSecret gets the value of the specified secret.
	GetSecret(namespace string, name string) (*corev1.Secret, error)

	// GetNamespaceWatched returns the namespace of the Secret being watched;
	// returns empty string if more than one namespace is being watched.
	GetNamespaceWatched() string

	// GetNameWatched returns the name of the Secret being watched;
	// returns empty string if more than one Secret is being watched.
	GetNameWatched() string
}

var (
	// createStackdriverKubeclientFunc provides a hook for setting up the kubeclient for testing.
	// It is set to createKubeclient otherwise.
	createStackdriverKubeclientFunc func() (kubernetes.Interface, error)
)

func init() {
	createStackdriverKubeclientFunc = createKubeclient
}

// NewSecretWatcher constructs a secretWatcher that watches all Secrets.
func NewSecretWatcher(observers ...Observer) (SecretWatcher, error) {
	return newSecretWatcherInternal("", "", observers...)
}

// NewSecretWatcherSingleNamespace constructs a secretWatcher that watches all Secrets in a namespace.
func NewSecretWatcherSingleNamespace(namespace string, observers ...Observer) (SecretWatcher, error) {
	return newSecretWatcherInternal(namespace, "", observers...)
}

// NewSecretWatcherSingleSecret constructs a secretWatcher that watches all a single Secret.
func NewSecretWatcherSingleSecret(namespace string, name string, observers ...Observer) (SecretWatcher, error) {
	return newSecretWatcherInternal(namespace, name, observers...)
}

// newSecretWatcherInternal constructs a secretWatcher.
func newSecretWatcherInternal(namespace string, name string, observers ...Observer) (SecretWatcher, error) {
	impl := &secretWatcherImpl{
		SecretNamespace: namespace,
		SecretName:      name,
		observers:       observers,
	}

	if clientErr := impl.setupKubeclient(); clientErr != nil {
		return nil, clientErr
	}

	return impl, nil
}

// secretWatherImpl implmements the secretWatcher interface.
type secretWatcherImpl struct {
	// kubeclient is the in-cluster Kubernetes kubeclient, which is lazy-initialized on first use.
	kubeclient kubernetes.Interface

	// watchLock ensures that only one watch is being run at a time.
	watchLock sync.Mutex
	// informer watches the Stackdriver secret if useStackdriverSecretEnabled is true.
	informer cache.SharedIndexInformer
	// stopCh stops the secretWatcherInformer when required.
	stopCh chan struct{}
	// watchStarted is whether or not the informer is running for the secret being watched.
	watchStarted bool

	// SecretName is the name of the secret being watched.
	SecretName string
	// SecretNamespace is the namespace of the secret being watched.
	SecretNamespace string
	// observers is the list of observers to trigger when the Secret is changed.
	observers []Observer
}

// StartWatch implements StartWatch of secretWatcher interface.
func (s *secretWatcherImpl) StartWatch() error {
	s.watchLock.Lock()
	defer s.watchLock.Unlock()
	if s.watchStarted {
		return nil
	}
	s.watchStarted = true

	s.setupInformer()
	s.stopCh = make(chan struct{})
	go s.informer.Run(s.stopCh)

	return nil
}

// StopWatch implements StopWatch secretWatcher interface.
func (s *secretWatcherImpl) StopWatch() {
	s.watchLock.Lock()
	defer s.watchLock.Unlock()
	if !s.watchStarted {
		return
	}
	s.watchStarted = false

	close(s.stopCh)
}

// GetSecret implements GetSecret of secretWatcher interface.
func (s *secretWatcherImpl) GetSecret(namespace string, name string) (*corev1.Secret, error) {
	if namespace == "" || name == "" {
		return nil, fmt.Errorf("Must specify a non-empty Secret namespace and name to get, namespace specified: %v, name specified: %v", namespace, name)
	}
	sec, secErr := s.kubeclient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})

	if secErr != nil {
		return nil, fmt.Errorf("Error getting Secret [%v] in namespace [%v]: %v", name, namespace, secErr)
	}

	return sec, nil
}

// GetNamespaceWatched implements GetNamespaceWatched of secretWatcher interface.
func (s *secretWatcherImpl) GetNamespaceWatched() string {
	return s.SecretNamespace
}

// GetNameWatched implements GetNameWatched of secretWatcher interface.
func (s *secretWatcherImpl) GetNameWatched() string {
	return s.SecretName
}

// setupInformer sets up a kubernetes informer to watch Secrets.
func (s *secretWatcherImpl) setupInformer() {
	var sharedInformerOpts []informers.SharedInformerOption
	if s.SecretNamespace != "" {
		namespaceOpt := informers.WithNamespace(s.SecretNamespace)
		sharedInformerOpts = append(sharedInformerOpts, namespaceOpt)

		if s.SecretName != "" {
			nameFieldSelectorFunc := func(listOpts *metav1.ListOptions) {
				listOpts.FieldSelector = fields.OneTermEqualSelector("metadata.name", s.SecretName).String()
			}
			nameOpt := informers.WithTweakListOptions(nameFieldSelectorFunc)
			sharedInformerOpts = append(sharedInformerOpts, nameOpt)
		}
	}

	informers := informers.NewSharedInformerFactoryWithOptions(s.kubeclient, 0, sharedInformerOpts...)
	s.informer = informers.Core().V1().Secrets().Informer()

	s.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sec := obj.(*corev1.Secret)
			for _, obs := range s.observers {
				obs.OnAdd(sec)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			sOld := oldObj.(*corev1.Secret)
			sNew := newObj.(*corev1.Secret)

			for _, obs := range s.observers {
				obs.OnUpdate(sOld, sNew)
			}
		},
		DeleteFunc: func(obj interface{}) {
			sec := obj.(*corev1.Secret)
			for _, obs := range s.observers {
				obs.OnDelete(sec)
			}
		},
	})
}

// createKubeclient creates a kubeclient from the in-cluster config.
// This only works when running on a kubernetes cluster.
func createKubeclient() (kubernetes.Interface, error) {
	config, configErr := rest.InClusterConfig()
	if configErr != nil {
		return nil, configErr
	}

	cs, clientErr := kubernetes.NewForConfig(config)
	if clientErr != nil {
		return nil, clientErr
	}

	return cs, nil
}

// setupKubeclient initializes the kubeclient.
func (s *secretWatcherImpl) setupKubeclient() error {
	client, err := createStackdriverKubeclientFunc()
	if err != nil {
		return err
	}

	s.kubeclient = client

	return nil
}
