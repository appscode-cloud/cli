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

package utils

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	firstUser      = "ace.user.1"
	firstOrg       = "2"
	defaultCluster = "generated-cluster"
	defaultContext = "generated-context"
	defaultUser    = "default"
	DefaultPath    = "/tmp/kubeconfig"
)

type KubeConfigGetter struct {
	scm     *runtime.Scheme
	config  *rest.Config
	kc      client.Client
	kubeCfg *clientcmdapi.Config
	secret  *corev1.Secret
}

func NewKubeconfigGetter(scm *runtime.Scheme, config *rest.Config) *KubeConfigGetter {
	kc, err := client.New(config, client.Options{Scheme: scm})
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	secret, err := getAuthSecret(kc)
	if err != nil {
		klog.Errorf("failed to get auth secret: %v", err)
		return nil
	}

	return &KubeConfigGetter{
		scm:     scm,
		config:  config,
		kc:      kc,
		kubeCfg: RESTConfigToKubeconfig(config),
		secret:  &secret,
	}
}

func RESTConfigToKubeconfig(cfg *rest.Config) *clientcmdapi.Config {
	kubeCfg := clientcmdapi.NewConfig()

	kubeCfg.Clusters[defaultCluster] = &clientcmdapi.Cluster{
		Server:                   cfg.Host,
		CertificateAuthorityData: cfg.CAData,
		InsecureSkipTLSVerify:    cfg.Insecure,
	}

	kubeCfg.AuthInfos[defaultUser] = &clientcmdapi.AuthInfo{
		Token:                 cfg.BearerToken,
		ClientCertificateData: cfg.CertData,
		ClientKeyData:         cfg.KeyData,
		Username:              cfg.Username,
		Password:              cfg.Password,
	}

	kubeCfg.Contexts[defaultContext] = &clientcmdapi.Context{
		Cluster:  defaultCluster,
		AuthInfo: defaultUser,
	}

	kubeCfg.CurrentContext = defaultContext

	return kubeCfg
}

func getAuthSecret(kc client.Client) (corev1.Secret, error) {
	var secretList corev1.SecretList
	err := kc.List(context.TODO(), &secretList, &client.ListOptions{Namespace: "open-cluster-management-cluster-auth"})
	if err != nil {
		return corev1.Secret{}, err
	}
	var secret corev1.Secret
	for _, s := range secretList.Items {
		// kubectl get secrets -n open-cluster-management-cluster-auth ace.user.1-token-j8dhhc
		//   annotations:
		//    kubernetes.io/service-account.name: ace.user.1
		if strings.HasPrefix(s.Name, firstUser) {
			val, exists := s.Annotations[corev1.ServiceAccountNameKey]
			if exists && val == firstUser {
				secret = s
			}
		}
	}

	if secret.Name == "" {
		return corev1.Secret{}, fmt.Errorf("secret not found")
	}
	return secret, nil
}

func (g *KubeConfigGetter) GetSpokeClient(spokeName string) (client.Client, error) {
	spokeKubeCfg := g.getSpokeKubeConfig(spokeName)
	return g.convertToClient(spokeKubeCfg)
}

func (g *KubeConfigGetter) getSpokeKubeConfig(spokeName string) *clientcmdapi.Config {
	kubeCfg := g.kubeCfg.DeepCopy()
	// clusters:
	//  - cluster:
	//      certificate-authority-data: <ca-from-above-secret>
	//      server: https://<hub-ip>:6443/apis/gateway.open-cluster-management.io/v1alpha1/clustergateways/<spoke-name>/proxy
	// users:
	//  - name: default
	//    user:
	//      as: ace.user.1
	//      as-user-extra:
	//        ace.appscode.com/org-id:
	//        - "2"
	//      token: <token-from-above-secret>
	kubeCfg.Clusters[defaultCluster].CertificateAuthorityData = g.secret.Data["ca.crt"]
	kubeCfg.Clusters[defaultCluster].Server = calculateServerURL(kubeCfg.Clusters[defaultCluster].Server, spokeName)
	kubeCfg.AuthInfos[defaultUser] = calculateUser(string(g.secret.Data["token"]))
	return kubeCfg
}

func calculateServerURL(cur string, spokeName string) string {
	getHost := func(server string) (string, error) {
		u, err := url.Parse(server)
		if err != nil {
			return "", err
		}

		host := u.Hostname()

		// If it's an IP, return as-is
		if net.ParseIP(host) != nil {
			return host, nil
		}

		// Otherwise it's DNS
		return host, nil
	}
	host, err := getHost(cur)
	if err != nil {
		_ = fmt.Errorf("err getting host: %v", err)
	}
	ret := fmt.Sprintf("https://%s:6443/apis/gateway.open-cluster-management.io/v1alpha1/clustergateways/%s/proxy", host, spokeName)
	fmt.Printf("Using host: %s\n", ret)
	return ret
}

func calculateUser(token string) *clientcmdapi.AuthInfo {
	user := clientcmdapi.NewAuthInfo()
	user.Token = token
	user.Impersonate = firstUser
	user.ImpersonateUserExtra = map[string][]string{
		kmapi.AceOrgIDKey: {firstOrg},
	}
	return user
}

func (g *KubeConfigGetter) convertToClient(kubeCfg *clientcmdapi.Config) (client.Client, error) {
	err := writeKubeconfig(DefaultPath, kubeCfg)
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.BuildConfigFromFlags("", DefaultPath)
	if err != nil {
		klog.Errorf("err building config: %v", err)
	}

	kc, err := client.New(config, client.Options{Scheme: g.scm})
	if err != nil {
		klog.Errorf("err creating client: %v", err)
	}

	return kc, nil
}

func writeKubeconfig(path string, cfg *clientcmdapi.Config) error {
	return clientcmd.WriteToFile(*cfg, path)
}
