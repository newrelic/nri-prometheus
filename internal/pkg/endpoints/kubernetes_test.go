// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package endpoints

import (
	"context"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/newrelic/nri-prometheus/internal/retry"
)

func TestWatch_Endpoints(t *testing.T) {
	t.Parallel()

	// This test doublecheck as well that endpoints labels are ignored
	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	err := retriever.Watch()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	err = populateFakeEndpointsData(client)
	if err != nil {
		t.Fatalf("error populating fake api: %s", err)
	}

	// As the data processing involved in the watchers is asynchronous, we might not have seen the data yet. So this
	// retries up to 10 times with an exponential backoff delay.
	err = retry.Do(func() error {
		targets, err := retriever.GetTargets()
		if err != nil {
			return err
		}
		if len(targets) != 6 {
			return errors.New("targets len didn't match")
		}

		target := targets[0]

		if target.Name != endpointsName {
			return errors.New("target name didn't match")
		}
		var listURLs []string
		for _, t := range targets {
			listURLs = append(listURLs, t.URL.String())
		}
		require.Contains(t, listURLs, "http://1.2.3.4:1/metrics", "this target was expected")
		require.Contains(t, listURLs, "http://1.2.3.4:2/metrics", "this target was expected")
		require.Contains(t, listURLs, "http://1.2.3.4:3/metrics", "this target was expected")
		require.Contains(t, listURLs, "http://1.2.3.4:4/metrics", "this target was expected")
		require.Contains(t, listURLs, "http://5.6.7.8:1/metrics", "this target was expected")
		require.Contains(t, listURLs, "http://5.6.7.8:2/metrics", "this target was expected")
		require.NotContains(t, listURLs, "http://10.20.30.40:1/metrics", "this target was not expected")
		require.NotContains(t, listURLs, "http://10.20.30.40:2/metrics", "this target was not expected")

		return nil
	}, retry.Timeout(2*time.Second), retry.Delay(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatch_EndpointsSinglePort(t *testing.T) {
	t.Parallel()

	// This test doublecheck as well that endpoints labels are ignored
	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	err := retriever.Watch()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	err = populateFakeEndpointsDataSinglePort(client)
	if err != nil {
		t.Fatalf("error populating fake api: %s", err)
	}

	// As the data processing involved in the watchers is asynchronous, we might not have seen the data yet. So this
	// retries up to 10 times with an exponential backoff delay.
	err = retry.Do(func() error {
		targets, err := retriever.GetTargets()
		if err != nil {
			return err
		}
		if len(targets) != 3 {
			return errors.New("targets len didn't match")
		}

		target := targets[0]
		if target.Name != endpointsName {
			return errors.New("target name didn't match")
		}
		var listURLs []string
		for _, t := range targets {
			listURLs = append(listURLs, t.URL.String())
		}
		require.Contains(t, listURLs, "http://1.2.3.4:1/metrics", "this target was expected")
		require.NotContains(t, listURLs, "http://1.2.3.4:2/metrics", "this target was not expected")
		require.NotContains(t, listURLs, "http://1.2.3.4:3/metrics", "this target was not expected")
		require.NotContains(t, listURLs, "http://1.2.3.4:4/metrics", "this target was not expected")
		require.Contains(t, listURLs, "http://5.6.7.8:1/metrics", "this target was expected")
		require.Contains(t, listURLs, "http://my-endpoints.test-ns.svc:1/metrics", "this target was expected")
		require.NotContains(t, listURLs, "http://5.6.7.8:2/metrics", "this target was not expected")
		require.NotContains(t, listURLs, "http://10.20.30.40:1/metrics", "this target was not expected")
		require.NotContains(t, listURLs, "http://10.20.30.40:2/metrics", "this target was not expected")

		return nil
	}, retry.Timeout(2*time.Second), retry.Delay(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatch_EndpointsModify(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	err := retriever.Watch()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	err = populateFakeEndpointsDataWithModify(client)
	if err != nil {
		t.Fatalf("error populating fake api: %s", err)
	}

	err = retry.Do(func() error {
		targets, err := retriever.GetTargets()
		if err != nil {
			return err
		}

		if len(targets) != 8 {
			return errors.New("targets len didn't match")
		}
		target := targets[0]
		if target.Name != endpointsName {
			return errors.New("target name didn't match")
		}
		var listURLs []string
		for _, t := range targets {
			listURLs = append(listURLs, t.URL.String())
		}

		// Notice that we are testing both update and annotation Override
		require.Contains(t, listURLs, "http://1.2.3.4:1/metricsOverride", "this target was expected")
		require.Contains(t, listURLs, "http://1.2.3.4:2/metricsOverride", "this target was expected")
		require.Contains(t, listURLs, "http://1.2.3.4:3/metricsOverride", "this target was expected")
		require.Contains(t, listURLs, "http://1.2.3.4:4/metricsOverride", "this target was expected")
		require.Contains(t, listURLs, "http://5.6.7.8:1/metricsOverride", "this target was expected")
		require.Contains(t, listURLs, "http://5.6.7.8:2/metricsOverride", "this target was expected")
		require.Contains(t, listURLs, "http://10.20.30.40:1/metricsOverride", "this target was not expected")
		require.Contains(t, listURLs, "http://10.20.30.40:2/metricsOverride", "this target was not expected")

		return nil
	}, retry.Timeout(2*time.Second), retry.Delay(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
}

const endpointsName = "my-endpoints"

func populateFakeEndpointsData(clientset *fake.Clientset) error {
	e := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("niceUid"),
			Name: endpointsName,
			Labels: map[string]string{
				// this labels should be ignored
				"prometheus.io/scrape": "false",
				"prometheus.io/path":   "/metricsDifferent",
				"prometheus.io/port":   "portNotExisting",
				"app":                  "my-app",
			},
			Annotations: map[string]string{
				// this annotations should be ignored
				"prometheus.io/scrape": "false",
				"prometheus.io/path":   "/metricsDifferent",
				"prometheus.io/port":   "portNotExisting",
				"app":                  "my-app",
			},
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.2.3.4",
					},
					{
						IP: "5.6.7.8",
					},
				},
				NotReadyAddresses: []corev1.EndpointAddress{
					{
						IP: "10.20.30.40",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port:     1,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Port:     2,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.2.3.4",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port:     3,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Port:     4,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		},
	}
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: endpointsName,
			Labels: map[string]string{
				// This labels should be overwritten
				"prometheus.io/scrape": "false",
				"prometheus.io/path":   "/metricsDifferent",
				"app":                  "my-app",
			},
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/path":   "/metrics",
			},
		},
	}

	_, err := clientset.CoreV1().Services("test-ns").Create(context.TODO(), s, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().Endpoints("test-ns").Create(context.TODO(), e, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func populateFakeEndpointsDataSinglePort(clientset *fake.Clientset) error {
	e := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("niceUid"),
			Name: endpointsName,
			Labels: map[string]string{
				// this labels should be ignored
				"prometheus.io/scrape": "false",
				"prometheus.io/path":   "/metricsDifferent",
				"prometheus.io/port":   "portNotExisting",
				"app":                  "my-app",
			},
			Annotations: map[string]string{
				// this annotations should be ignored
				"prometheus.io/scrape": "false",
				"prometheus.io/path":   "/metricsDifferent",
				"prometheus.io/port":   "portNotExisting",
				"app":                  "my-app",
			},
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.2.3.4",
					},
					{
						IP: "5.6.7.8",
					},
				},
				NotReadyAddresses: []corev1.EndpointAddress{
					{
						IP: "10.20.30.40",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port:     1,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Port:     2,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.2.3.4",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port:     3,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Port:     4,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		},
	}
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: endpointsName,
			Labels: map[string]string{
				// This labels should be overwritten
				"prometheus.io/scrape": "false",
				"prometheus.io/path":   "/metricsDifferent",
				"prometheus.io/port":   "notexisting",
				"app":                  "my-app",
			},
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/path":   "/metrics",
				"prometheus.io/port":   "1",
			},
		},
	}

	_, err := clientset.CoreV1().Services("test-ns").Create(context.TODO(), s, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().Endpoints("test-ns").Create(context.TODO(), e, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func populateFakeEndpointsDataWithModify(clientset *fake.Clientset) error {
	e := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("niceUid"),
			Name: endpointsName,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.2.3.4",
					},
					{
						IP: "5.6.7.8",
					},
				},
				NotReadyAddresses: []corev1.EndpointAddress{
					{
						IP: "10.20.30.40",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port:     1,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Port:     2,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.2.3.4",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port:     3,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Port:     4,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		},
	}

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: endpointsName,
			Labels: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/path":   "/metrics",
				"app":                  "my-app",
			},
			Annotations: map[string]string{
				"prometheus.io/path": "/metricsOverride",
			},
		},
	}

	_, err := clientset.CoreV1().Services("test-ns").Create(context.TODO(), s, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().Endpoints("test-ns").Create(context.TODO(), e, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	e.Subsets[0].NotReadyAddresses = nil
	addr := e.Subsets[0].Addresses
	e.Subsets[0].Addresses = append(addr, corev1.EndpointAddress{IP: "10.20.30.40"})
	_, err = clientset.CoreV1().Endpoints("test-ns").Update(context.TODO(), e, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func TestWatch_Services(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	err := retriever.Watch()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	err = populateFakeServiceData(client)
	if err != nil {
		t.Fatalf("error populating fake api: %s", err)
	}

	// As the data processing involved in the watchers is asynchronous, we might not have seen the data yet. So this
	// retries up to 10 times with an exponential backoff delay.
	err = retry.Do(func() error {
		targets, err := retriever.GetTargets()
		if err != nil {
			return err
		}

		if len(targets) != 1 {
			return errors.New("targets len didn't match")
		}

		target := targets[0]
		if target.Name != "my-service" {
			return errors.New("target name didn't match")
		}
		if target.URL.String() != "http://my-service.test-ns.svc:8080/metrics/federate?format=prometheus" {
			return errors.New("target URL didn't match: " + target.URL.String())
		}
		return nil
	}, retry.Timeout(2*time.Second), retry.Delay(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatch_Pods(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	err := retriever.Watch()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	err = populateFakePodData(client)
	if err != nil {
		t.Fatalf("error populating fake api: %s", err)
	}

	err = retry.Do(func() error {
		targets, err := retriever.GetTargets()
		if err != nil {
			return err
		}

		if len(targets) != 1 {
			return errors.New("targets len didn't match")
		}

		target := targets[0]
		if target.Name != "my-pod" {
			return errors.New("target name didn't match")
		}
		if target.URL.String() != "http://10.10.10.1:8080/metrics/federate?format=prometheus" {
			return errors.New("target URL didn't match: " + target.URL.String())
		}
		return nil
	}, retry.Timeout(2*time.Second), retry.Delay(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatch_PodsModify(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	err := retriever.Watch()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	err = populateFakePodDataModify(client)
	if err != nil {
		t.Fatalf("error populating fake api: %s", err)
	}

	err = retry.Do(func() error {
		targets, err := retriever.GetTargets()
		if err != nil {
			return err
		}

		if len(targets) != 1 {
			return errors.New("targets len didn't match")
		}

		target := targets[0]
		if target.Name != "my-pod-2" {
			return errors.New("target name didn't match")
		}
		if target.URL.String() != "http://10.10.10.2:8080/metrics" {
			return errors.New("target URL didn't match: " + target.URL.String())
		}
		return nil
	}, retry.Timeout(2*time.Second), retry.Delay(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatch_NodeReconnect(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	retriever.watching = true

	watcher := watch.NewRaceFreeFake()
	resource := watchableResource{
		name:         "node",
		listFunction: retriever.listNodes,
		watchFunction: func() (watch.Interface, error) {
			return watcher, nil
		},
	}

	// Start watching for node resources
	go retriever.watchResource(resource)

	targets, err := retriever.GetTargets()
	require.NoError(t, err)
	assert.Equal(t, 0, len(targets))
	ns := fakeNodeData()
	watcher.Add(ns[0])

	err = retry.Do(func() error {
		targets, err = retriever.GetTargets()
		if err != nil {
			return err
		}

		// Node add event detected by watcher. It's 2 because we add the node
		// and cadvisor as targets
		if len(targets) != 2 {
			return errors.New("targets len didn't match: " + strconv.Itoa(len(targets)))
		}
		return nil
	}, retry.Timeout(2*time.Second), retry.Delay(100*time.Millisecond))
	require.NoError(t, err)

	// Close the channel, trigger a reconnect and add a new node
	watcher.Stop()
	watcher.Reset()
	watcher.Add(ns[1])

	time.Sleep(100 * time.Millisecond)

	err = retry.Do(
		func() error {
			targets, err = retriever.GetTargets()
			if err != nil {
				return err
			}

			// New node detected after reconnect
			if len(targets) != 4 {
				return errors.New("targets len after reconnect didn't match: " + strconv.Itoa(len(targets)))
			}
			return nil
		},
		retry.Timeout(2*time.Second), retry.Delay(100*time.Millisecond),
	)
	require.NoError(t, err)
}

func TestWatch_Nodes(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	err := retriever.Watch()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	err = populateFakeNodeData(client)
	if err != nil {
		t.Fatalf("error populating fake api: %s", err)
	}

	err = retry.Do(func() error {
		targets, err := retriever.GetTargets()
		if err != nil {
			return err
		}

		if len(targets) != 4 {
			return errors.New("targets len didn't match: " + strconv.Itoa(len(targets)))
		}

		return nil
	}, retry.Timeout(2*time.Second), retry.Delay(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatch_Nodes_NodesWithNoScrapeLabelAreNotBeingScraped(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	retriever.requireScrapeEnabledLabelForNodes = true

	err := retriever.Watch()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	err = populateFakeNodeData(client)
	if err != nil {
		t.Fatalf("error populating fake api: %s", err)
	}

	err = retry.Do(func() error {
		targets, err := retriever.GetTargets()
		if err != nil {
			t.Fatal(err)
		}

		if len(targets) > 0 {
			t.Fatal("no targets were expected but received: " + strconv.Itoa(len(targets)))
		}

		return nil
	}, retry.Timeout(2*time.Second), retry.Delay(100*time.Millisecond))
}

func newFakeKubernetesTargetRetriever(client *fake.Clientset) *KubernetesTargetRetriever {
	return &KubernetesTargetRetriever{
		client:             client,
		targets:            new(sync.Map),
		scrapeEnabledLabel: "prometheus.io/scrape",
		scrapeServices:     true,
		scrapeEndpoints:    true,
	}
}

func fakeNodeData() []*corev1.Node {
	return []*corev1.Node{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:    types.UID("zetano"),
				Name:   "my-node",
				Labels: map[string]string{},
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{
						Type:    corev1.NodeInternalIP,
						Address: "127.0.0.1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:    types.UID("perengano"),
				Name:   "my-node2",
				Labels: map[string]string{},
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{
						Type:    corev1.NodeInternalIP,
						Address: "127.0.0.2",
					},
				},
			},
		},
	}
}

func populateFakeNodeData(clientset *fake.Clientset) error {
	ns := fakeNodeData()
	_, err := clientset.CoreV1().Nodes().Create(context.TODO(), ns[0], metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Nodes().Create(context.TODO(), ns[1], metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func populateFakePodDataModify(clientset *fake.Clientset) error {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("mengano"),
			Name: "my-pod",
			Labels: map[string]string{
				"prometheus.io/scrape": "true",
				"app":                  "pod-my-app",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Ports: []corev1.ContainerPort{
					{
						Name:          "http-metrics",
						Protocol:      corev1.ProtocolTCP,
						ContainerPort: 8080,
					},
				},
			},
		}},
		Status: corev1.PodStatus{
			PodIP: "10.10.10.1",
		},
	}
	p2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("zutano"),
			Name: "my-pod-2",
			Labels: map[string]string{
				"prometheus.io/scrape": "falsey",
				"app":                  "pod-my-app-2",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Ports: []corev1.ContainerPort{
					{
						Name:          "http-metrics",
						Protocol:      corev1.ProtocolTCP,
						ContainerPort: 8080,
					},
				},
			},
		}},
		Status: corev1.PodStatus{
			PodIP: "10.10.10.2",
		},
	}

	p3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("pepito"),
			Name: "my-pod-3",
			Labels: map[string]string{
				"prometheus.io/scrape": "true",
				"app":                  "pod-my-app-3",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Ports: []corev1.ContainerPort{
					{
						Name:          "http-metrics",
						Protocol:      corev1.ProtocolTCP,
						ContainerPort: 8080,
					},
				},
			},
		}},
		Status: corev1.PodStatus{
			PodIP: "10.10.10.3",
		},
	}

	_, err := clientset.CoreV1().Pods("test-ns").Create(context.TODO(), p, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods("test-ns").Create(context.TODO(), p2, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods("test-ns").Create(context.TODO(), p3, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	p.ObjectMeta.Labels["prometheus.io/scrape"] = "falsy"
	p2.ObjectMeta.Labels["prometheus.io/scrape"] = "true"
	delete(p3.ObjectMeta.Labels, "prometheus.io/scrape")

	_, err = clientset.CoreV1().Pods("test-ns").Update(context.TODO(), p, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods("test-ns").Update(context.TODO(), p2, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods("test-ns").Update(context.TODO(), p3, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func populateFakePodData(clientset *fake.Clientset) error {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("mengano"),
			Name: "my-pod",
			Labels: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/path":   "metrics/federate?format=prometheus",
				"app":                  "pod-my-app",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Ports: []corev1.ContainerPort{
					{
						Name:          "http-metrics",
						Protocol:      corev1.ProtocolTCP,
						ContainerPort: 8080,
					},
				},
			},
		}},
		Status: corev1.PodStatus{
			PodIP: "10.10.10.1",
		},
	}

	p2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("zutano"),
			Name: "my-pod-2",
			Labels: map[string]string{
				"prometheus.io/scrape": "falsey",
				"app":                  "pod-my-app-2",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Ports: []corev1.ContainerPort{
					{
						Name:          "http-metrics",
						Protocol:      corev1.ProtocolTCP,
						ContainerPort: 8080,
					},
				},
			},
		}},
		Status: corev1.PodStatus{
			PodIP: "10.10.10.2",
		},
	}

	_, err := clientset.CoreV1().Pods("test-ns").Create(context.TODO(), p, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods("test-ns").Create(context.TODO(), p2, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func populateFakeServiceData(clientset *fake.Clientset) error {
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("niceUid"),
			Name: "my-service",
			Labels: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/path":   "/metrics/federate?format=prometheus",
				"app":                  "my-app",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "http-metrics",
					Protocol: corev1.ProtocolTCP,
					Port:     8080,
				},
			},
		},
	}
	s2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID:    types.UID("notNiceUid"),
			Name:   "my-service-no-scrapeable",
			Labels: map[string]string{},
		},
	}

	_, err := clientset.CoreV1().Services("test-ns").Create(context.TODO(), s, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().Services("test-ns").Create(context.TODO(), s2, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func TestPodTargetsPortAnnotationsOverrideLabels(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		podTargets(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pod",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "8080",
				},
				Labels: map[string]string{
					// annotation should override this.
					"prometheus.io/port": "80",
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node-a",
				Containers: []corev1.Container{
					{
						Name: "app",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http-app",
								ContainerPort: 80,
							},
							{
								Name:          "http-metrics",
								ContainerPort: 8080,
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		}),
		[]Target{
			{
				Name: "my-pod",
				Object: Object{
					Name: "my-pod",
					Kind: "pod",
					Labels: labels.Set{
						"podName":                  "my-pod",
						"namespaceName":            "test-ns",
						"deploymentName":           "",
						"nodeName":                 "node-a",
						"label.prometheus.io/port": "80",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "10.0.0.1:8080",
					Path:   "/metrics",
				},
			},
		},
	)
}

func TestPodTargetsNoPort(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		podTargets(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pod",
				Namespace: "test-ns",
			},
			Spec: corev1.PodSpec{
				NodeName: "node-a",
				Containers: []corev1.Container{
					{
						Name: "app",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http-app",
								ContainerPort: 80,
							},
							{
								Name:          "http-metrics",
								ContainerPort: 8080,
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		}),
		[]Target{
			{
				Name: "my-pod",
				Object: Object{
					Name: "my-pod",
					Kind: "pod",
					Labels: labels.Set{
						"podName":        "my-pod",
						"namespaceName":  "test-ns",
						"deploymentName": "",
						"nodeName":       "node-a",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "10.0.0.1:80",
					Path:   "/metrics",
				},
			},
			{
				Name: "my-pod",
				Object: Object{
					Name: "my-pod",
					Kind: "pod",
					Labels: labels.Set{
						"podName":        "my-pod",
						"namespaceName":  "test-ns",
						"deploymentName": "",
						"nodeName":       "node-a",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "10.0.0.1:8080",
					Path:   "/metrics",
				},
			},
		},
	)
}

func TestPodTargetsPortAnnotation(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		podTargets(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pod",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "8080",
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node-a",
				Containers: []corev1.Container{
					{
						Name: "app",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http-app",
								ContainerPort: 80,
							},
							{
								Name:          "http-metrics",
								ContainerPort: 8080,
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		}),
		[]Target{
			{
				Name: "my-pod",
				Object: Object{
					Name: "my-pod",
					Kind: "pod",
					Labels: labels.Set{
						"podName":        "my-pod",
						"namespaceName":  "test-ns",
						"deploymentName": "",
						"nodeName":       "node-a",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "10.0.0.1:8080",
					Path:   "/metrics",
				},
			},
		},
	)
}

func TestPodTargetsInvalidURL(t *testing.T) {
	t.Parallel()

	assert.Empty(
		t,
		podTargets(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pod",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "foobar",
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node-a",
				Containers: []corev1.Container{
					{
						Name: "app",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http-app",
								ContainerPort: 80,
							},
							{
								Name:          "http-metrics",
								ContainerPort: 8080,
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		}),
	)
}

func TestPodTargetsPortLabels(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		podTargets(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pod",
				Namespace: "test-ns",
				Labels: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "8080",
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node-a",
				Containers: []corev1.Container{
					{
						Name: "app",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http-app",
								ContainerPort: 80,
							},
							{
								Name:          "http-metrics",
								ContainerPort: 8080,
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		}),
		[]Target{
			{
				Name: "my-pod",
				Object: Object{
					Name: "my-pod",
					Kind: "pod",
					Labels: labels.Set{
						"podName":                    "my-pod",
						"namespaceName":              "test-ns",
						"deploymentName":             "",
						"nodeName":                   "node-a",
						"label.prometheus.io/scrape": "true",
						"label.prometheus.io/port":   "8080",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "10.0.0.1:8080",
					Path:   "/metrics",
				},
			},
		},
	)
}

func TestServiceTargetsPortAnnotationsOverrideLabels(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		serviceTargets(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-service",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "8080",
				},
				Labels: map[string]string{
					// annotation should override this.
					"prometheus.io/port": "80",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: "http-app",
						Port: 80,
					},
					{
						Name: "http-metrics",
						Port: 8080,
					},
				},
			},
		}),
		[]Target{
			{
				Name: "my-service",
				Object: Object{
					Name: "my-service",
					Kind: "service",
					Labels: labels.Set{
						"serviceName":              "my-service",
						"namespaceName":            "test-ns",
						"label.prometheus.io/port": "80",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "my-service.test-ns.svc:8080",
					Path:   "/metrics",
				},
			},
		},
	)
}

func TestServiceTargetsPortAnnotation(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		serviceTargets(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-service",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "8080",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: "http-app",
						Port: 80,
					},
					{
						Name: "http-metrics",
						Port: 8080,
					},
				},
			},
		}),
		[]Target{
			{
				Name: "my-service",
				Object: Object{
					Name: "my-service",
					Kind: "service",
					Labels: labels.Set{
						"serviceName":   "my-service",
						"namespaceName": "test-ns",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "my-service.test-ns.svc:8080",
					Path:   "/metrics",
				},
			},
		},
	)
}

func TestServiceTargetsInvalidURL(t *testing.T) {
	t.Parallel()

	assert.Empty(
		t,
		serviceTargets(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-service",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "foobar",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: "http-app",
						Port: 80,
					},
					{
						Name: "http-metrics",
						Port: 8080,
					},
				},
			},
		}),
	)
}

func TestServiceTargetsNoPort(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		serviceTargets(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-service",
				Namespace: "test-ns",
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: "http-app",
						Port: 80,
					},
					{
						Name: "http-metrics",
						Port: 8080,
					},
				},
			},
		}),
		[]Target{
			{
				Name: "my-service",
				Object: Object{
					Name: "my-service",
					Kind: "service",
					Labels: labels.Set{
						"serviceName":   "my-service",
						"namespaceName": "test-ns",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "my-service.test-ns.svc:80",
					Path:   "/metrics",
				},
			},
			{
				Name: "my-service",
				Object: Object{
					Name: "my-service",
					Kind: "service",
					Labels: labels.Set{
						"serviceName":   "my-service",
						"namespaceName": "test-ns",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "my-service.test-ns.svc:8080",
					Path:   "/metrics",
				},
			},
		},
	)
}

func TestServiceTargetsPortLabel(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		serviceTargets(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-service",
				Namespace: "test-ns",
				Labels: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "8080",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: "http-app",
						Port: 80,
					},
					{
						Name: "http-metrics",
						Port: 8080,
					},
				},
			},
		}),
		[]Target{
			{
				Name: "my-service",
				Object: Object{
					Name: "my-service",
					Kind: "service",
					Labels: labels.Set{
						"serviceName":                "my-service",
						"namespaceName":              "test-ns",
						"label.prometheus.io/scrape": "true",
						"label.prometheus.io/port":   "8080",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "my-service.test-ns.svc:8080",
					Path:   "/metrics",
				},
			},
		},
	)
}

func TestProcessEventPodWithoutPodIP(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	err := retriever.Watch()
	if err != nil {
		t.Fatal(err)
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("seed"),
			Name: "test-pod",
			Labels: map[string]string{
				"prometheus.io/scrape": "true",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Ports: []corev1.ContainerPort{
					{
						Name:          "http-metrics",
						Protocol:      corev1.ProtocolTCP,
						ContainerPort: 8080,
					},
				},
			},
		}},
	}

	// Process the event. We expect no items to be cached
	event := watch.Event{Type: watch.Added, Object: pod}
	retriever.processEvent(event, false)
	actual, _ := retriever.targets.Load(string(pod.GetUID()))
	assert.Nil(t, actual)

	// The pod has been updated, and has a PodIP assigned
	pod.Status = corev1.PodStatus{PodIP: "10.10.10.10"}

	// We process the message again, and check if it now successfully caches the Pod
	event = watch.Event{Type: watch.Modified, Object: pod}
	retriever.processEvent(event, false)
	actual, _ = retriever.targets.Load(string(pod.GetUID()))
	assert.Equal(t, podTargets(pod), actual)
}

func TestProcessEvent(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	retriever := newFakeKubernetesTargetRetriever(client)
	err := retriever.Watch()
	if err != nil {
		t.Fatal(err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("seed"),
			Name: "test-pod",
			Labels: map[string]string{
				"prometheus.io/scrape": "true",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Ports: []corev1.ContainerPort{
					{
						Name:          "http-metrics",
						Protocol:      corev1.ProtocolTCP,
						ContainerPort: 8080,
					},
				},
			},
		}},
		Status: corev1.PodStatus{PodIP: "10.10.10.10"},
	}

	// Add the event.
	event := watch.Event{Type: watch.Added, Object: pod}
	retriever.processEvent(event, true)
	actual, _ := retriever.targets.Load(string(pod.GetUID()))
	assert.Equal(t, podTargets(pod), actual)

	// Modify the event without removing.
	pod.ObjectMeta.Labels = map[string]string{}
	event = watch.Event{Type: watch.Modified, Object: pod}
	retriever.processEvent(event, false)
	actual, _ = retriever.targets.Load(string(pod.GetUID()))
	assert.Equal(t, podTargets(pod), actual)

	// Verify `requireLabel` removes unlabeled object.
	retriever.processEvent(event, true)
	length := 0
	retriever.targets.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	if length != 0 {
		t.Fatal("failed to delete modified object")
	}

	// Add the event back in (without requiring a label).
	event = watch.Event{Type: watch.Added, Object: pod}
	retriever.processEvent(event, false)
	actual, _ = retriever.targets.Load(string(pod.GetUID()))
	assert.Equal(t, podTargets(pod), actual)

	// Delete the event.
	event = watch.Event{Type: watch.Deleted, Object: pod}
	retriever.processEvent(event, false)
	length = 0
	retriever.targets.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	if length != 0 {
		t.Fatal("failed to delete object")
	}

	// Add the event back in to check the Error type.
	retriever.targets.Store(string(pod.GetUID()), podTargets(pod))
	event = watch.Event{Type: watch.Error, Object: pod}
	retriever.processEvent(event, false)
	length = 0
	retriever.targets.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	if length != 0 {
		t.Fatal("failed to delete errored object")
	}
}

func TestPodTargetsPathAnnotationsOverrideLabels(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		podTargets(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pod",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"prometheus.io/path": "/metrics/1",
				},
				Labels: map[string]string{
					// annotation should override this.
					"prometheus.io/path": "/metrics/0",
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node-a",
				Containers: []corev1.Container{
					{
						Name: "app",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http-app",
								ContainerPort: 80,
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		}),
		[]Target{
			{
				Name: "my-pod",
				Object: Object{
					Name: "my-pod",
					Kind: "pod",
					Labels: labels.Set{
						"podName":                  "my-pod",
						"namespaceName":            "test-ns",
						"deploymentName":           "",
						"nodeName":                 "node-a",
						"label.prometheus.io/path": "/metrics/0",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "10.0.0.1:80",
					Path:   "/metrics/1",
				},
			},
		},
	)
}

func TestPodTargetsPathAnnotations(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		podTargets(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pod",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"prometheus.io/path": "/metrics/1",
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node-a",
				Containers: []corev1.Container{
					{
						Name: "app",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http-app",
								ContainerPort: 80,
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		}),
		[]Target{
			{
				Name: "my-pod",
				Object: Object{
					Name: "my-pod",
					Kind: "pod",
					Labels: labels.Set{
						"podName":        "my-pod",
						"namespaceName":  "test-ns",
						"deploymentName": "",
						"nodeName":       "node-a",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "10.0.0.1:80",
					Path:   "/metrics/1",
				},
			},
		},
	)
}

func TestPodTargetsPathLabel(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		podTargets(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pod",
				Namespace: "test-ns",
				Labels: map[string]string{
					"prometheus.io/path": "/metrics/1",
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node-a",
				Containers: []corev1.Container{
					{
						Name: "app",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http-app",
								ContainerPort: 80,
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		}),
		[]Target{
			{
				Name: "my-pod",
				Object: Object{
					Name: "my-pod",
					Kind: "pod",
					Labels: labels.Set{
						"podName":                  "my-pod",
						"namespaceName":            "test-ns",
						"deploymentName":           "",
						"nodeName":                 "node-a",
						"label.prometheus.io/path": "/metrics/1",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "10.0.0.1:80",
					Path:   "/metrics/1",
				},
			},
		},
	)
}

func TestServiceTargetsPathAnnotationsOverrideLabels(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		serviceTargets(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-service",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"prometheus.io/path": "/metrics/1",
				},
				Labels: map[string]string{
					// annotation should override this.
					"prometheus.io/path": "/metrics/0",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: "http-metrics",
						Port: 8080,
					},
				},
			},
		}),
		[]Target{
			{
				Name: "my-service",
				Object: Object{
					Name: "my-service",
					Kind: "service",
					Labels: labels.Set{
						"serviceName":              "my-service",
						"namespaceName":            "test-ns",
						"label.prometheus.io/path": "/metrics/0",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "my-service.test-ns.svc:8080",
					Path:   "/metrics/1",
				},
			},
		},
	)
}

func TestServiceTargetsPathAnnotations(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		serviceTargets(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-service",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"prometheus.io/path": "/metrics/1",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: "http-metrics",
						Port: 8080,
					},
				},
			},
		}),
		[]Target{
			{
				Name: "my-service",
				Object: Object{
					Name: "my-service",
					Kind: "service",
					Labels: labels.Set{
						"serviceName":   "my-service",
						"namespaceName": "test-ns",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "my-service.test-ns.svc:8080",
					Path:   "/metrics/1",
				},
			},
		},
	)
}

func TestServiceTargetsPathLabel(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(
		t,
		serviceTargets(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-service",
				Namespace: "test-ns",
				Labels: map[string]string{
					"prometheus.io/path": "/metrics/1",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: "http-metrics",
						Port: 8080,
					},
				},
			},
		}),
		[]Target{
			{
				Name: "my-service",
				Object: Object{
					Name: "my-service",
					Kind: "service",
					Labels: labels.Set{
						"serviceName":              "my-service",
						"namespaceName":            "test-ns",
						"label.prometheus.io/path": "/metrics/1",
					},
				},
				URL: url.URL{
					Scheme: "http",
					Host:   "my-service.test-ns.svc:8080",
					Path:   "/metrics/1",
				},
			},
		},
	)
}
