package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

var (
	remainingLifetimeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tls_remaining_lifetime",
			Help: "Remaining lifetime of TLS certificates in days",
		},
		[]string{"namespace", "secret_name"},
	)
)

func init() {
	prometheus.MustRegister(remainingLifetimeGauge)
}

func initKubernetesClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: api.Cluster{Server: ""}}).ClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func checkTLSCertificates(ctx context.Context, clientset *kubernetes.Clientset) {
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list namespaces: %v", err)
	}

	for _, namespace := range namespaces.Items {
		secrets, err := clientset.CoreV1().Secrets(namespace.Name).List(ctx, v1.ListOptions{})
		if err != nil {
			log.Fatalf("Failed to list secrets in namespace %s: %v", namespace.Name, err)
		}

		for _, secret := range secrets.Items {
			if secret.Type == "kubernetes.io/tls" {
				certData, ok := secret.Data["tls.crt"]
				if !ok {
					continue
				}

				block, _ := pem.Decode(certData)
				if block == nil {
					log.Printf("Failed to decode PEM data for secret %s/%s", namespace.Name, secret.Name)
					continue
				}

				cert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					log.Printf("Failed to parse certificate for secret %s/%s: %v", namespace.Name, secret.Name, err)
					continue
				}

				remainingLifetime := float64(cert.NotAfter.Sub(time.Now()).Hours()) / 24

				remainingLifetimeGauge.With(prometheus.Labels{
					"namespace":   namespace.Name,
					"secret_name": secret.Name,
				}).Set(remainingLifetime)
			}
		}
	}
}
