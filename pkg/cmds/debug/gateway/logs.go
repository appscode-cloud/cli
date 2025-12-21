/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gateway

import (
	"context"
	"os"
	"path"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"kmodules.xyz/client-go/meta"
)

func (g *gatewayOpts) collectOperatorLogs() error {
	labels := labels.SelectorFromSet(map[string]string{
		meta.NameLabelKey: catalogManager,
	}).String()
	pods, err := g.kubeClient.CoreV1().Pods("ace").List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels,
	})
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		isOperatorPod := false
		for _, container := range pod.Spec.Containers {
			if container.Name == catalogManager {
				isOperatorPod = true
			}
		}
		if isOperatorPod {
			err = g.writeLogs(pod.Name, pod.Namespace, catalogManager)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *gatewayOpts) collectGatewayLogs() error {
	labels := labels.SelectorFromSet(map[string]string{
		meta.NameLabelKey: "gateway",
	}).String()
	pods, err := g.kubeClient.CoreV1().Pods(g.hr.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels,
	})
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		isOperatorPod := false
		for _, container := range pod.Spec.Containers {
			if container.Name == envoyGateway {
				isOperatorPod = true
			}
		}
		if isOperatorPod {
			err = g.writeLogs(pod.Name, pod.Namespace, envoyGateway)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *gatewayOpts) collectEnvoyLogs() error {
	labels := labels.SelectorFromSet(map[string]string{
		meta.NameLabelKey: envoy,
	}).String()
	pods, err := g.kubeClient.CoreV1().Pods(g.hr.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels,
	})
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		isOperatorPod := false
		for _, container := range pod.Spec.Containers {
			if container.Name == envoy {
				isOperatorPod = true
			}
		}
		if isOperatorPod {
			err = g.writeLogs(pod.Name, pod.Namespace, envoy)
			if err != nil {
				return err
			}
			if err := g.collectConfigDump(pod.ObjectMeta); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *gatewayOpts) writeLogs(podName, ns, container string) error {
	req := g.kubeClient.CoreV1().Pods(ns).GetLogs(podName, &corev1.PodLogOptions{
		Container: container,
	})

	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return err
	}
	defer podLogs.Close() // nolint:errcheck

	logFile, err := os.Create(path.Join(g.dir, logsDir, podName+"_"+container+".log"))
	if err != nil {
		return err
	}
	defer logFile.Close() // nolint:errcheck

	buf := make([]byte, 1024)
	for {
		bytesRead, err := podLogs.Read(buf)
		if err != nil {
			break
		}
		_, _ = logFile.Write(buf[:bytesRead])
	}
	return nil
}
