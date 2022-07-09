package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/consul/sdk/iptables"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWaitForAnnotation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		annotation string
		pod        func() corev1.Pod
		retries    uint64
		exists     bool
	}{
		{
			name:       "Pod with annotation already existing",
			annotation: "pod1",
			pod: func() corev1.Pod {
				pod1 := createPodWithAnnotation("pod1", "pod1", "")
				return *pod1
			},
			retries: 1,
			exists:  true,
		},
		{
			name:       "Pod without annotation",
			annotation: "pod2",
			pod: func() corev1.Pod {
				pod1 := createPodWithAnnotation("pod2", "", "")
				return *pod1
			},
			retries: 1,
			exists:  false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := waitForAnnotation(c.pod(), c.annotation, c.retries)
			require.Equal(t, c.exists, actual)
		})
	}
}

func TestParseAnnotation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		annotation string
		pod        func() corev1.Pod
		expected   iptables.Config
		err        error
	}{
		{
			name:       "Pod with iptables.Config annotation",
			annotation: annotationCNIProxyConfig,
			pod: func() corev1.Pod {
				// Use iptables.Config so that if the Config struct ever changes that the test is still valid
				cfg := iptables.Config{ProxyUserID: "1234"}
				j, err := json.Marshal(&cfg)
				if err != nil {
					t.Fatalf("could not marshal iptables config: %v", err)
				}
				pod1 := createPodWithAnnotation("pod1", annotationCNIProxyConfig, string(j))
				return *pod1
			},
			expected: iptables.Config{
				ProxyUserID: "1234",
			},
			err: nil,
		},
		{
			name:       "Pod without iptables.Config annotation",
			annotation: annotationCNIProxyConfig,
			pod: func() corev1.Pod {
				pod1 := createPodWithAnnotation("pod2", "", "")
				return *pod1
			},
			expected: iptables.Config{},
			err:      fmt.Errorf("could not find %s annotation for %s pod", annotationCNIProxyConfig, "pod2"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := parseAnnotation(c.pod(), c.annotation)
			require.Equal(t, c.expected, actual)
			require.Equal(t, c.err, err)
		})
	}
}

func TestSearchDNSIPFromEnvironment(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		prefix   string
		expected string
		pod      func(*corev1.Pod) *corev1.Pod
	}{
		{
			name:   "Pod with DNS set",
			prefix: "consul",
                        expected: "127.0.0.1",
			pod: func(pod *corev1.Pod) *corev1.Pod {
				pod.Spec.Containers[0].Env = []corev1.EnvVar{
					{
						Name:  "CONSUL_DNS_SERVICE_HOST",
						Value: "127.0.0.1",
					},
				}
				return pod
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := searchDNSIPFromEnvironment(*c.pod(minimalPod()), c.prefix)
			require.Equal(t, c.expected, actual)
		})
	}
}

// createPodWithAnnotation creates a pod with an annotation
func createPodWithAnnotation(name, key, value string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "default",
			Annotations: map[string]string{},
		},
	}
	if key != "" {
		// The content of the annotation is not important
		pod.Annotations[key] = value
	}
	return pod
}

func minimalPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   "default",
			Name:        "minimal",
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "web",
				},
				{
					Name: "web-side",
				},
			},
		},
	}
}
