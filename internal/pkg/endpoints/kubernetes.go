// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package endpoints

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/newrelic/nri-prometheus/internal/retry"
)

const trueStr = "true"

var klog = logrus.WithField("component", "KubernetesAPI")

// COPIED FROM Prometheus code
const (
	NodeLegacyHostIP          = "LegacyHostIP"
	defaultScrapeEnabledLabel = "prometheus.io/scrape"
	defaultScrapePortLabel    = "prometheus.io/port"
	defaultScrapeSchemeLabel  = "prometheus.io/scheme"
	defaultScrapeScheme       = "http"
	defaultScrapePathLabel    = "prometheus.io/path"
	defaultScrapePath         = "/metrics"
)

// watchableResource identifies a k8s resource that implement the k8s watchable
// interface.
//
// The `listFunction` retrieves the scrapable objects and updates the
// retriever list of targets.
//
// The `watchFunction` returns a channel to use for waiting on events.
type watchableResource struct {
	name                      string
	listFunction              func() error
	watchFunction             func() (watch.Interface, error)
	requireScrapeEnabledLabel bool
}

// nodeAddresses returns the provided node's address, based on the priority:
// 1. NodeInternalIP
// 2. NodeExternalIP
// 3. NodeLegacyHostIP
// 3. NodeHostName
//
// Derived from k8s.io/kubernetes/pkg/util/node/node.go
// COPIED FROM Prometheus code
func nodeAddress(node *corev1.Node) (string, map[corev1.NodeAddressType][]string, error) {
	m := map[corev1.NodeAddressType][]string{}
	for _, a := range node.Status.Addresses {
		m[a.Type] = append(m[a.Type], a.Address)
	}

	if addresses, ok := m[corev1.NodeInternalIP]; ok {
		return addresses[0], m, nil
	}
	if addresses, ok := m[corev1.NodeExternalIP]; ok {
		return addresses[0], m, nil
	}
	if addresses, ok := m[corev1.NodeAddressType(NodeLegacyHostIP)]; ok {
		return addresses[0], m, nil
	}
	if addresses, ok := m[corev1.NodeHostName]; ok {
		return addresses[0], m, nil
	}
	return "", m, fmt.Errorf("host address unknown")
}

// listNodes gets all the scrapable nodes that are currently available
func (k *kubernetesTargetRetriever) listNodes() error {
	nodes, err := k.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, n := range nodes.Items {
		if !isObjectScrapable(&n, k.scrapeEnabledLabel) {
			klog.Debugf("node %s was skipped because label or annotation %s is not true", n.Name, k.scrapeEnabledLabel)
			continue
		}

		targets, err := nodeTargets(&n)
		if err != nil {
			klog.WithError(err).WithField("node", n.Name).Warnf("can't get targets for node. Ignoring")
			continue
		}
		k.targets.Store(string(n.UID), targets)
	}
	return nil
}

func nodeTargets(n *corev1.Node) ([]Target, error) {
	nodeURL := url.URL{
		Scheme: "https",
		Host:   "kubernetes.default.svc",
		Path:   fmt.Sprintf("/api/v1/nodes/%s/proxy/metrics", n.Name),
	}
	cadvisorURL := url.URL{
		Scheme: "https",
		Host:   "kubernetes.default.svc",
		Path:   fmt.Sprintf("/api/v1/nodes/%s/proxy/metrics/cadvisor", n.Name),
	}

	_, addrMap, err := nodeAddress(n)
	if err != nil {
		return nil, err
	}

	lbls := getK8sLabels(n)
	for ty, a := range addrMap {
		ln := "node_address_" + string(ty)
		lbls[ln] = a[0]
	}
	lbls["nodeName"] = n.Name

	object := Object{Name: n.Name, Kind: "node", Labels: lbls}

	return []Target{
		{
			Name:      n.Name,
			URL:       nodeURL,
			Object:    object,
			UseBearer: true,
		},
		{
			Name:      "cadvisor_" + n.Name,
			URL:       cadvisorURL,
			Object:    object,
			UseBearer: true,
		},
	}, nil
}

// listEndpoints gets the scrapable endpoints that are currently available
func (k *kubernetesTargetRetriever) listEndpoints() error {
	endpoints, err := k.client.CoreV1().Endpoints("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	services, err := k.client.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	// we create this tmp data structure only to speed up finding the service related to an endpoint
	tmp := map[string]corev1.Service{}
	for _, s := range services.Items {
		tmp[s.Namespace+"/"+s.Name] = s
	}

	for _, e := range endpoints.Items {
		s, ok := tmp[e.Namespace+"/"+e.Name]
		if !ok {
			continue
		}
		// In order to understand if an endpoint is scrapable we need to rely on the service annotations/labels
		if isObjectScrapable(&s, k.scrapeEnabledLabel) {
			k.targets.Store(string(e.UID), endpointsTargets(&e, &s))
		}
	}

	return nil
}

// listServices gets the scrapable services that are currently available
func (k *kubernetesTargetRetriever) listServices() error {
	services, err := k.client.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, s := range services.Items {
		if isObjectScrapable(&s, k.scrapeEnabledLabel) {
			k.targets.Store(string(s.UID), serviceTargets(&s))
		}
	}

	return nil
}

func isObjectScrapable(o metav1.Object, label string) bool {
	return o.GetLabels()[label] == trueStr || o.GetAnnotations()[label] == trueStr
}

func getK8sLabels(o metav1.Object) labels.Set {
	lbls := labels.Set{}
	for lk, lv := range o.GetLabels() {
		lbls["label."+lk] = lv
	}
	return lbls
}

func isInt(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func getScheme(o metav1.Object) string {
	// Annotations take precedence over labels.
	if annotation, ok := o.GetAnnotations()[defaultScrapeSchemeLabel]; ok {
		return annotation
	}
	if label, ok := o.GetLabels()[defaultScrapeSchemeLabel]; ok {
		return label
	}

	return defaultScrapeScheme
}

func getPath(o metav1.Object) string {
	// Annotations take precedence over labels.
	if annotation, ok := o.GetAnnotations()[defaultScrapePathLabel]; ok {
		return annotation
	}
	if label, ok := o.GetLabels()[defaultScrapePathLabel]; ok {
		return label
	}
	return defaultScrapePath
}

func getPort(o metav1.Object) string {
	// Annotations take precedence over labels.
	if annotation, ok := o.GetAnnotations()[defaultScrapePortLabel]; ok {
		if isInt(annotation) {
			return annotation
		}
	}
	if label, ok := o.GetLabels()[defaultScrapePortLabel]; ok {
		if isInt(label) {
			return label
		}
	}
	return ""
}

// parsePath parses a partial URL query from the prometheus.io/path annotation, such as `/metrics?format=foo` and
// returns separately the path and the query. This is needed because the `prometheus.io/path` annotation is often
// abused, and query arguments are included in it.
// If the contents of prometheus.io/path cannot be parsed as a URL, it returns an error, which will skip the target.
// Note that url.Parse will not return an error for an empty rawUrl.
func parsePath(pathQuery string) (string, string, error) {
	u, err := url.Parse(pathQuery)
	if err != nil {
		return "", "", fmt.Errorf("cannot parse url from %q: %w", pathQuery, err)
	}

	return u.Path, u.RawQuery, nil
}

func endpointsTarget(e *corev1.Endpoints, u url.URL) Target {
	lbls := getK8sLabels(e)
	// Name and Namespace of services and endpoints collides
	lbls["serviceName"] = e.Name
	lbls["namespaceName"] = e.Namespace

	return Target{
		Name: e.Name,
		URL:  u,
		Object: Object{
			Name:   e.Name,
			Kind:   "endpoints",
			Labels: lbls,
		},
	}
}

// returns all the possible targets for a endpoint (multiple targets per port)
func endpointsTargets(e *corev1.Endpoints, s *corev1.Service) []Target {
	// we need to pass the service since the annotations are not inherited
	port := getPort(s)
	scheme := getScheme(s)
	path, query, err := parsePath(getPath(s))
	if err != nil {
		klog.WithError(err).Warnf("Skipping endpoints from  %s/%s", s.Namespace, s.Name)
		return nil
	}

	var targets []Target
	for _, subset := range e.Subsets {
		// we are skipping eSub.NotReadyAddresses
		for _, eSubAddr := range subset.Addresses {
			for _, eSubPort := range subset.Ports {
				if eSubPort.Protocol != corev1.ProtocolTCP {
					continue
				}

				subPortStr := fmt.Sprintf("%d", eSubPort.Port)
				if port != "" && port != subPortStr {
					// If we parsed a port from the config, then only grab subsets which contain said port.
					continue
				}
				u := url.URL{
					Scheme:   scheme,
					Host:     net.JoinHostPort(eSubAddr.IP, subPortStr),
					Path:     path,
					RawQuery: query,
				}
				targets = append(targets, endpointsTarget(e, u))
			}
		}
	}
	return targets
}

func serviceTarget(s *corev1.Service, u url.URL) Target {
	lbls := getK8sLabels(s)
	lbls["serviceName"] = s.Name
	lbls["namespaceName"] = s.Namespace

	return Target{
		Name: s.Name,
		URL:  u,
		Object: Object{
			Name:   s.Name,
			Kind:   "service",
			Labels: lbls,
		},
	}
}

// returns all the possible targets for a service (1 target per port)
func serviceTargets(s *corev1.Service) []Target {
	port := getPort(s)
	scheme := getScheme(s)
	path, query, err := parsePath(getPath(s))
	if err != nil {
		klog.WithError(err).Warnf("Skipping service  %s/%s", s.Namespace, s.Name)
		return nil
	}

	if port != "" {
		u := url.URL{
			Scheme: scheme,
			Host:   net.JoinHostPort(fmt.Sprintf("%s.%s.svc", s.Name, s.Namespace), port),
			Path:   path,
		}
		return []Target{serviceTarget(s, u)}
	}

	// No port specified so return a target for each Port defined for the Service.
	var targets []Target
	for _, port := range s.Spec.Ports {
		u := url.URL{
			Scheme:   scheme,
			Host:     net.JoinHostPort(fmt.Sprintf("%s.%s.svc", s.Name, s.Namespace), fmt.Sprintf("%d", port.Port)),
			Path:     path,
			RawQuery: query,
		}
		targets = append(targets, serviceTarget(s, u))
	}
	return targets
}

func (k *kubernetesTargetRetriever) listPods() error {
	pods, err := k.client.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, p := range pods.Items {
		if isObjectScrapable(&p, k.scrapeEnabledLabel) {
			k.targets.Store(string(p.UID), podTargets(&p))
		}
	}
	return nil
}

func getPodDeployment(p *corev1.Pod) string {
	var deploymentName string
	if len(p.OwnerReferences) > 0 {
		podOwner := p.OwnerReferences[0]
		if podOwner.Kind == "ReplicaSet" {
			s := strings.Split(podOwner.Name, "-")
			deploymentName = strings.Join(s[:len(s)-1], "-")
		}
	}
	return deploymentName
}

func podTarget(p *corev1.Pod, u url.URL) Target {
	lbls := getK8sLabels(p)
	lbls["podName"] = p.Name
	lbls["namespaceName"] = p.Namespace
	lbls["nodeName"] = p.Spec.NodeName
	lbls["deploymentName"] = getPodDeployment(p)

	return Target{
		Name: p.Name,
		URL:  u,
		Object: Object{
			Name:   p.Name,
			Kind:   "pod",
			Labels: lbls,
		},
	}
}

func podTargets(p *corev1.Pod) []Target {
	// if the Pod has not yet been allocated to a Node, or Kubelet/CNI has not yet assigned an ipAddress,
	// the pod is not yet scrapable.
	if p.Status.PodIP == "" {
		return nil
	}

	port := getPort(p)
	scheme := getScheme(p)
	path, query, err := parsePath(getPath(p))
	if err != nil {
		klog.WithError(err).Warnf("Skipping endpoints pod  %s/%s", p.Namespace, p.Name)
		return nil
	}

	if port != "" {
		u := url.URL{
			Scheme:   scheme,
			Host:     net.JoinHostPort(p.Status.PodIP, port),
			Path:     path,
			RawQuery: query,
		}
		return []Target{podTarget(p, u)}
	}

	// No port specified so return a target for each ContainerPort defined for the pod.
	var targets []Target
	for _, c := range p.Spec.Containers {
		for _, port := range c.Ports {
			u := url.URL{
				Scheme:   scheme,
				Host:     net.JoinHostPort(p.Status.PodIP, fmt.Sprintf("%d", port.ContainerPort)),
				Path:     path,
				RawQuery: query,
			}
			targets = append(targets, podTarget(p, u))
		}
	}
	return targets
}

// Option is implemented by functions that configure the kubernetesTargetRetriever
type Option func(*kubernetesTargetRetriever) error

// WithKubeConfig configures the kubernetesTargetRetriever to load the Kubernetes configuration
// from a kubeconfig file. This file is usually found in ~/.kube/config
func WithKubeConfig(kubeConfigFile string) Option {
	return func(ktr *kubernetesTargetRetriever) error {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			return fmt.Errorf("could not read kubeconfig file: %w", err)
		}

		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("could create kubernetes client: %w", err)
		}

		ktr.client = client
		return nil
	}
}

// WithInClusterConfig configures the kubernetesTargetRetriever to load the Kubernetes configuration
// from within a running pod in the cluster (/var/run/secrets/kubernetes.io/serviceaccount/*)
func WithInClusterConfig() Option {
	return func(ktr *kubernetesTargetRetriever) error {
		config, err := rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("could not read inclusterconfig: %w", err)
		}

		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("could create kubernetes client: %w", err)
		}

		ktr.client = client
		return nil
	}
}

// kubernetesTargetRetriever sets the watchers for the different Targets
// and listens for the arrival of new data from them.
type kubernetesTargetRetriever struct {
	watching                          bool
	client                            kubernetes.Interface
	targets                           *sync.Map
	scrapeEnabledLabel                string
	scrapeServices                    bool
	scrapeEndpoints                   bool
	requireScrapeEnabledLabelForNodes bool
}

// NewKubernetesTargetRetriever creates a new kubernetesTargetRetriever
// setting the required label to identified targets that can be scrapped.
func NewKubernetesTargetRetriever(scrapeEnabledLabel string, requireScrapeEnabledLabelForNodes bool, scrapeServices bool, scrapeEndpoints bool, options ...Option) (TargetRetriever, error) {
	if scrapeEnabledLabel == "" {
		scrapeEnabledLabel = defaultScrapeEnabledLabel
	}

	ktr := &kubernetesTargetRetriever{
		targets:                           new(sync.Map),
		scrapeEnabledLabel:                scrapeEnabledLabel,
		scrapeEndpoints:                   scrapeEndpoints,
		scrapeServices:                    scrapeServices,
		requireScrapeEnabledLabelForNodes: requireScrapeEnabledLabelForNodes,
	}

	for _, opt := range options {
		if err := opt(ktr); err != nil {
			return nil, err
		}
	}

	if ktr.client == nil {
		return nil, errors.New("newKubernetesTargetRetriever requires a valid Kubernetes configuration option, none are given")
	}

	return ktr, nil
}

// Watch retrieves and caches an initial list of URLs and triggers a process in background
func (k *kubernetesTargetRetriever) Watch() error {
	if k.watching {
		return errors.New("already watching")
	}

	// List all the targets before starting the watch to ensure the fetcher has
	// targets for its first run. Dismiss any error because the list function
	// will be invoked properly (with back-off and retries) per resource kind
	// in `watchResource`.
	k.listTargets()

	k.watchTargets()

	k.watching = true

	return nil
}

// Name returns the identifying name of the kubernetesTargetRetriever.
func (k *kubernetesTargetRetriever) Name() string {
	return "kubernetes"
}

// GetTargets returns a slice with all the targets currently registered.
func (k *kubernetesTargetRetriever) GetTargets() ([]Target, error) {
	length := 0
	k.targets.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	targets := make([]Target, 0, length)
	k.targets.Range(func(_, y interface{}) bool {
		for _, t := range y.([]Target) {
			if t.Object.Kind == "service" && !k.scrapeServices {
				continue
			}
			if t.Object.Kind == "endpoints" && !k.scrapeEndpoints {
				continue
			}
			targets = append(targets, t)
		}
		return true
	})
	return targets, nil
}

func (k *kubernetesTargetRetriever) listTargets() {
	_ = k.listPods()
	_ = k.listServices()
	_ = k.listEndpoints()
	_ = k.listNodes()
}

func (k *kubernetesTargetRetriever) watchTargets() {
	for _, r := range k.getWatchableResources() {
		go k.watchResource(r)
	}
}

func (k *kubernetesTargetRetriever) getWatchableResources() []watchableResource {
	return []watchableResource{{
		name:                      "pod",
		listFunction:              k.listPods,
		requireScrapeEnabledLabel: true,
		watchFunction: func() (watch.Interface, error) {
			return k.client.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{})
		},
	}, {
		name:                      "node",
		listFunction:              k.listNodes,
		requireScrapeEnabledLabel: k.requireScrapeEnabledLabelForNodes,
		watchFunction: func() (watch.Interface, error) {
			return k.client.CoreV1().Nodes().Watch(context.TODO(), metav1.ListOptions{})
		},
	}, {
		name:                      "service",
		requireScrapeEnabledLabel: true,
		listFunction:              k.listServices,
		watchFunction: func() (watch.Interface, error) {
			return k.client.CoreV1().Services("").Watch(context.TODO(), metav1.ListOptions{})
		},
	}, {
		name:                      "endpoints",
		requireScrapeEnabledLabel: true,
		listFunction:              k.listEndpoints,
		watchFunction: func() (watch.Interface, error) {
			return k.client.CoreV1().Endpoints("").Watch(context.TODO(), metav1.ListOptions{})
		},
	}}
}

func (k *kubernetesTargetRetriever) processEvent(event watch.Event, requireLabel bool) {
	object := event.Object.(metav1.Object)

	scrapable := k.isEventScrapable(object)
	_, seen := k.targets.Load(string(object.GetUID()))
	setLogLevelEvent(event, object)

	// Please, do not try to reduce the amount of code below or simplify the conditionals.
	// This logic is very complex and full of different cases, it's better to be more verbose
	// and have a logic that is easier to reason about.
	switch event.Type {
	case watch.Added:
		// If the object requires labeling, has the right label and was not seen before,
		// we add it.
		if requireLabel && scrapable && !seen {
			k.addTarget(object, event.Type)
			return
		}
		// If the object doesn't require labels to be added, we always add.
		// In some configurations this is the case for nodes.
		if !requireLabel {
			k.addTarget(object, event.Type)
		}
	case watch.Modified:
		if requireLabel {
			// If the object requires labels, is scrapable we update it
			if scrapable {
				k.addTarget(object, event.Type)
				return
			}
			// If the object is not scrapable and we've seen it before, we remove it.
			if !scrapable && seen {
				k.targets.Delete(string(object.GetUID()))
				debugLogEvent(klog, event.Type, "deleted", object)
				switch obj := object.(type) {
				case *corev1.Service:
					if e, err := k.client.CoreV1().Endpoints(obj.Namespace).Get(context.TODO(), obj.Name, metav1.GetOptions{}); err == nil {
						k.targets.Delete(string(e.GetUID()))
						debugLogEvent(klog, event.Type, "deleted", e)
					}
				}
			}
		} else {
			// If the object doesn't require label and was not seen before, we add it.
			if !seen {
				k.addTarget(object, event.Type)
				return
			}
			// If the doesn't doesn't require label and we already have it, update its data.
			// Things like the IP could be changing.
			if seen {
				k.addTarget(object, event.Type)
				debugLogEvent(klog, event.Type, "modified", object)
			}
		}
	case watch.Deleted, watch.Error:
		k.targets.Delete(string(object.GetUID()))
		debugLogEvent(klog, event.Type, "deleted", object)
	}
}

func setLogLevelEvent(event watch.Event, object metav1.Object) {
	if klog.Level <= logrus.DebugLevel {
		klog.WithFields(logrus.Fields{
			"action": event.Type,
			"name":   object.GetName(),
			"uid":    object.GetUID(),
			"ns":     object.GetNamespace(),
		}).Trace("kubernetes event received")
	}
}

func (k *kubernetesTargetRetriever) isEventScrapable(object metav1.Object) bool {
	scrapable := isObjectScrapable(object, k.scrapeEnabledLabel)
	switch obj := object.(type) {
	case *corev1.Endpoints:
		if s, err := k.client.CoreV1().Services(obj.Namespace).Get(context.TODO(), obj.Name, metav1.GetOptions{}); err == nil {
			// For endpoints we need to rely on the service annotations/labels since they are not always propagated
			scrapable = isObjectScrapable(s, k.scrapeEnabledLabel)
		}
	}
	return scrapable
}

// addTarget adds the target to the cache k.targets
func (k *kubernetesTargetRetriever) addTarget(object metav1.Object, event watch.EventType) {
	// targets variable stores a list of n httpEndpoints linked to an object.
	// That will be stored into the k.targets map having object.uuid as key
	var targets []Target
	var err error
	switch obj := object.(type) {
	case *corev1.Endpoints:
		if obj.Subsets == nil {
			k.targets.Delete(string(object.GetUID()))
			return
		}
		// In this case we should fetch the service since the path annotation depends on the service
		if s, err := k.client.CoreV1().Services(obj.Namespace).Get(context.TODO(), obj.Name, metav1.GetOptions{}); err == nil {
			targets = endpointsTargets(obj, s)
		}

	case *corev1.Service:
		targets = serviceTargets(obj)
		// In this case we should update as well the endpoints since
		// the annotation could have been added enabling the scraping not triggering an endpoints events
		// This is not ideal but its the only way to support annotation since those are not inherited by endpoints
		if e, err := k.client.CoreV1().Endpoints(obj.Namespace).Get(context.TODO(), obj.Name, metav1.GetOptions{}); err == nil {
			endpointsTargets := endpointsTargets(e, obj)
			if len(endpointsTargets) != 0 {
				k.targets.Store(string(e.GetUID()), endpointsTargets)
			} else {
				// When modifying a service it could happen that there are no targets and therefore we should delete the old ones
				k.targets.Delete(string(e.GetUID()))
			}
		}

	case *corev1.Pod:
		targets = podTargets(obj)

	case *corev1.Node:
		targets, err = nodeTargets(obj)
		if err != nil {
			klog.WithError(err).WithField("node", obj.Name).Warn("can't get targets for node. Ignoring")
			debugLogEvent(klog, event, "ignored", object)
			return
		}
	}
	if len(targets) == 0 {
		k.targets.Delete(string(object.GetUID()))
		debugLogEvent(klog, event, "deleted", object)
		return
	}
	k.targets.Store(string(object.GetUID()), targets)
	debugLogEvent(klog, event, "added", object)
}

func debugLogEvent(log *logrus.Entry, event watch.EventType, action string, object metav1.Object) {
	log.WithFields(logrus.Fields{
		"action": action,
		"event":  event,
		"name":   object.GetName(),
		"uid":    object.GetUID(),
	}).Trace("kubernetes event handled")
}

// watchResource retrieves the scrapable resources and watches for changes
// on such resources. If the watch connection is terminated, the process is
// started again to ensure no updates are lost between watch restarts.
func (k *kubernetesTargetRetriever) watchResource(resource watchableResource) {
	for {
		timer := prometheus.NewTimer(
			prometheus.ObserverFunc(
				listTargetsDurationByKind.WithLabelValues(k.Name(), resource.name).Set,
			),
		)
		err := retry.Do(resource.listFunction)
		timer.ObserveDuration()
		if err != nil {
			klog.WithError(err).Warnf("couldn't list %s resource, retrying", resource.name)
			continue
		}

		watches, err := resource.watchFunction()
		if err != nil {
			klog.WithError(err).Warnf(
				"couldn't subscribe for %s resource watch, retrying",
				resource.name,
			)
			continue
		}
		for w := range watches.ResultChan() {
			k.processEvent(w, resource.requireScrapeEnabledLabel)
		}
		klog.WithError(err).Warnf(
			"disconnected from %s resource watch, reconnecting",
			resource.name,
		)
	}
}
