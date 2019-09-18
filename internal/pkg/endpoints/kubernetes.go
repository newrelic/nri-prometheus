// Package endpoints ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package endpoints

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

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
func nodeAddress(node *apiv1.Node) (string, map[apiv1.NodeAddressType][]string, error) {
	m := map[apiv1.NodeAddressType][]string{}
	for _, a := range node.Status.Addresses {
		m[a.Type] = append(m[a.Type], a.Address)
	}

	if addresses, ok := m[apiv1.NodeInternalIP]; ok {
		return addresses[0], m, nil
	}
	if addresses, ok := m[apiv1.NodeExternalIP]; ok {
		return addresses[0], m, nil
	}
	if addresses, ok := m[apiv1.NodeAddressType(NodeLegacyHostIP)]; ok {
		return addresses[0], m, nil
	}
	if addresses, ok := m[apiv1.NodeHostName]; ok {
		return addresses[0], m, nil
	}
	return "", m, fmt.Errorf("host address unknown")
}

// listNodes gets all the scrapable nodes that are currently available
func (k *KubernetesTargetRetriever) listNodes() error {
	nodes, err := k.client.CoreV1().Nodes().List(metav1.ListOptions{})
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

func nodeTargets(n *apiv1.Node) ([]Target, error) {
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

	lbls := labels.Set{}
	for lk, lv := range n.Labels {
		lbls["label."+lk] = lv
	}

	for ty, a := range addrMap {
		ln := "node_address_" + string(ty)
		lbls[ln] = a[0]
	}
	lbls["nodeName"] = n.Name

	object := Object{Name: n.Name, Kind: "node", Labels: lbls}

	return []Target{
		New(n.Name, nodeURL, object),
		New("cadvisor_"+n.Name, cadvisorURL, object),
	}, nil
}

// listServices gets the scrapable services that are currently available
func (k *KubernetesTargetRetriever) listServices() error {
	services, err := k.client.CoreV1().Services("").List(metav1.ListOptions{})
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

func objectTargets(object metav1.Object) []Target {
	switch obj := object.(type) {
	case *apiv1.Service:
		return serviceTargets(obj)
	case *apiv1.Pod:
		return podTargets(obj)
	case *apiv1.Node:
		targets, err := nodeTargets(obj)
		if err != nil {
			klog.WithError(err).WithField("node", obj.Name).Warn("can't get targets for node. Ignoring")
			return nil
		}
		return targets
	}
	return nil
}

func serviceTarget(s *apiv1.Service, port, path string) Target {
	lbls := labels.Set{}
	hostname := fmt.Sprintf("%s.%s.svc", s.Name, s.Namespace)
	addr := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(hostname, port),
		Path:   path,
	}
	for lk, lv := range s.Labels {
		lbls["label."+lk] = lv
	}
	lbls["serviceName"] = s.Name
	lbls["namespaceName"] = s.Namespace
	return New(s.Name, addr, Object{Name: s.Name, Kind: "service", Labels: lbls})
}

// returns all the possible targets for a service (1 target per port)
func serviceTargets(s *apiv1.Service) []Target {
	// Annotations take precedence over labels.
	path, ok := s.Annotations[defaultScrapePathLabel]
	if !ok {
		path, ok = s.Labels[defaultScrapePathLabel]
		if !ok {
			path = defaultScrapePath
		}
	}
	port, ok := s.Annotations[defaultScrapePortLabel]
	if !ok {
		port, ok = s.Labels[defaultScrapePortLabel]
	}

	// Only return a target for the specified port.
	if ok {
		return []Target{serviceTarget(s, port, path)}
	}

	// No port specified so return a target for each Port defined for the Service.
	targets := make([]Target, 0, len(s.Spec.Ports))
	for _, port := range s.Spec.Ports {
		targets = append(targets, serviceTarget(s, strconv.FormatInt(int64(port.Port), 10), path))
	}
	return targets
}

func (k *KubernetesTargetRetriever) listPods() error {
	pods, err := k.client.CoreV1().Pods("").List(metav1.ListOptions{})
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

func getPodDeployment(p *apiv1.Pod) string {
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

func podTarget(p *apiv1.Pod, port, path string) Target {
	lbls := labels.Set{}
	addr := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(p.Status.PodIP, port),
		Path:   path,
	}
	for lk, lv := range p.Labels {
		lbls["label."+lk] = lv
	}
	lbls["podName"] = p.Name
	lbls["namespaceName"] = p.Namespace
	lbls["nodeName"] = p.Spec.NodeName
	lbls["deploymentName"] = getPodDeployment(p)
	return New(p.Name, addr, Object{Name: p.Name, Kind: "pod", Labels: lbls})
}

func podTargets(p *apiv1.Pod) []Target {
	// Annotations take precedence over labels.
	path, ok := p.Annotations[defaultScrapePathLabel]
	if !ok {
		path, ok = p.Labels[defaultScrapePathLabel]
		if !ok {
			path = defaultScrapePath
		}
	}

	// Annotations take precedence over labels.
	port, ok := p.Annotations[defaultScrapePortLabel]
	if !ok {
		port, ok = p.Labels[defaultScrapePortLabel]
	}

	// Only return a target for the specified port.
	if ok {
		return []Target{podTarget(p, port, path)}
	}

	// No port specified so return a target for each ContainerPort defined for the pod.
	targets := make([]Target, 0, len(p.Spec.Containers))
	for _, c := range p.Spec.Containers {
		for _, port := range c.Ports {
			targets = append(targets, podTarget(p, strconv.FormatInt(int64(port.ContainerPort), 10), path))
		}
	}
	return targets
}

// KubernetesTargetRetriever sets the watchers for the different Targets
// and listens for the arrival of new data from them.
type KubernetesTargetRetriever struct {
	watching                          bool
	client                            kubernetes.Interface
	targets                           *sync.Map
	scrapeEnabledLabel                string
	requireScrapeEnabledLabelForNodes bool
}

// NewKubernetesTargetRetriever creates a new KubernetesTargetRetriever
// setting the required label to identified targets that can be scrapped.
func NewKubernetesTargetRetriever(scrapeEnabledLabel string, requireScrapeEnabledLabelForNodes bool) (*KubernetesTargetRetriever, error) {
	if scrapeEnabledLabel == "" {
		scrapeEnabledLabel = defaultScrapeEnabledLabel
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubernetesTargetRetriever{
		client:                            client,
		targets:                           new(sync.Map),
		scrapeEnabledLabel:                scrapeEnabledLabel,
		requireScrapeEnabledLabelForNodes: requireScrapeEnabledLabelForNodes,
	}, nil
}

// Watch retrieves and caches an initial list of URLs and triggers a process in background
func (k *KubernetesTargetRetriever) Watch() error {
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

// Name returns the identifying name of the KubernetesTargetRetriever.
func (k *KubernetesTargetRetriever) Name() string {
	return "kubernetes"
}

// GetTargets returns a slice with all the targets currently registered.
func (k *KubernetesTargetRetriever) GetTargets() ([]Target, error) {
	length := 0
	k.targets.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	targets := make([]Target, 0, length)
	k.targets.Range(func(_, y interface{}) bool {
		targets = append(targets, y.([]Target)...)
		return true
	})
	return targets, nil
}

func (k *KubernetesTargetRetriever) listTargets() {
	_ = k.listPods()
	_ = k.listServices()
	_ = k.listNodes()
}

func (k *KubernetesTargetRetriever) watchTargets() {
	for _, r := range k.getWatchableResources() {
		go k.watchResource(r)
	}
}

func (k *KubernetesTargetRetriever) getWatchableResources() []watchableResource {
	return []watchableResource{{
		name:                      "pod",
		listFunction:              k.listPods,
		requireScrapeEnabledLabel: true,
		watchFunction: func() (watch.Interface, error) {
			return k.client.CoreV1().Pods("").Watch(metav1.ListOptions{})
		},
	}, {
		name:                      "node",
		listFunction:              k.listNodes,
		requireScrapeEnabledLabel: k.requireScrapeEnabledLabelForNodes,
		watchFunction: func() (watch.Interface, error) {
			return k.client.CoreV1().Nodes().Watch(metav1.ListOptions{})
		},
	}, {
		name:                      "service",
		requireScrapeEnabledLabel: true,
		listFunction:              k.listServices,
		watchFunction: func() (watch.Interface, error) {
			return k.client.CoreV1().Services("").Watch(metav1.ListOptions{})
		},
	}}
}

func (k *KubernetesTargetRetriever) processEvent(event watch.Event, requireLabel bool) {
	object := event.Object.(metav1.Object)
	var seen, scrapable bool
	_, seen = k.targets.Load(string(object.GetUID()))
	scrapable = isObjectScrapable(object, k.scrapeEnabledLabel)
	if klog.Level <= logrus.DebugLevel {
		klog.WithFields(logrus.Fields{
			"action": event.Type,
			"name":   object.GetName(),
			"uid":    object.GetUID(),
			"ns":     object.GetNamespace(),
		}).Debug("kubernetes event received")
	}

	// Please, do not try to reduce the amount of code below or simplify the conditionals.
	// This logic is very complex and full of different cases, it's better to be more verbose
	// and have a logic that is easier to reason about.
	switch event.Type {
	case watch.Added:
		// If the object requires labeling, has the right label and was not seen before,
		// we add it.
		if requireLabel && scrapable && !seen {
			k.targets.Store(string(object.GetUID()), objectTargets(object))
			debugLogEvent(klog, event.Type, "added", object)
			return
		}
		// If the object doesn't require labels to be added, we always add.
		// In some configurations this is the case for nodes.
		if !requireLabel {
			k.targets.Store(string(object.GetUID()), objectTargets(object))
			debugLogEvent(klog, event.Type, "added", object)
		}
	case watch.Modified:
		if requireLabel {
			// If the object requires labels, is scrapable and was not seen before,
			// we add it.
			if scrapable && !seen {
				k.targets.Store(string(object.GetUID()), objectTargets(object))
				debugLogEvent(klog, event.Type, "added", object)
				return
			}
			// If the object is not scrapable and we've seen it before, we remove it.
			if !scrapable && seen {
				k.targets.Delete(string(object.GetUID()))
				debugLogEvent(klog, event.Type, "deleted", object)
			}
		}
		if !requireLabel {
			// If the object doesn't require label and was not seen before, we add it.
			if !seen {
				k.targets.Store(string(object.GetUID()), objectTargets(object))
				debugLogEvent(klog, event.Type, "added", object)
				return
			}
			// If the doesn't doesn't require label and we already have it, update its data.
			// Things like the IP could be changing.
			if seen {
				k.targets.Store(string(object.GetUID()), objectTargets(object))
				debugLogEvent(klog, event.Type, "modified", object)
				return
			}
		}
	case watch.Deleted:
		k.targets.Delete(string(object.GetUID()))
		debugLogEvent(klog, event.Type, "deleted", object)
	case watch.Error:
		k.targets.Delete(string(object.GetUID()))
		debugLogEvent(klog, event.Type, "deleted", object)
	}
}

func debugLogEvent(log *logrus.Entry, event watch.EventType, action string, object metav1.Object) {
	log.WithFields(logrus.Fields{
		"action": action,
		"event":  event,
		"name":   object.GetName(),
		"uid":    object.GetUID(),
	}).Debug("kubernetes event handled")
}

// watchResource retrieves the scrapable resources and watches for changes
// on such resources. If the watch connection is terminated, the process is
// started again to ensure no updates are lost between watch restarts.
func (k *KubernetesTargetRetriever) watchResource(resource watchableResource) {
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
