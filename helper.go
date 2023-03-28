package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
)

var err error

func initKubeClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

	if _, err := os.Stat(kubeconfig); err == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	return clientset, nil
}

func checkTLSCertificates(ctx context.Context, clientset *kubernetes.Clientset) {
	// Create metric instrument
	meter := otel.Tracer("tlscheck")
	remainingLifetimeGauge := metric.Must(meter).NewInt64Gauge("tls.remaining_lifetime",
		metric.WithDescription("Remaining lifetime of TLS certificates in days"),
	)

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list namespaces: %v", err)
	}

	for _, namespace := range namespaces.Items {
		secrets, err := clientset.CoreV1().Secrets(namespace.Name).List(ctx, v1.ListOptions{})
		if err != nil {
			log.Printf("Failed to list secrets in namespace %s: %v", namespace.Name, err)
			continue
		}

		for _, secret := range secrets.Items {
			if secret.Type != "kubernetes.io/tls" {
				continue
			}

			certBytes := secret.Data["tls.crt"]

			block, _ := pem.Decode(certBytes)
			if block == nil {
				log.Printf("Failed to parse PEM data for secret %s/%s", namespace.Name, secret.Name)
				continue
			}

			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				log.Printf("Failed to parse certificate for secret %s/%s: %v", namespace.Name, secret.Name, err)
				continue
			}

			remainingDays := int(cert.NotAfter.Sub(time.Now()).Hours() / 24)

			// Record remaining lifetime using the metric instrument
			remainingLifetimeGauge.Record(ctx, int64(remainingDays),
				attribute.String("namespace", namespace.Name),
				attribute.String("secret_name", secret.Name),
			)

			fmt.Printf("%-30s %5d\n", secret.Name, remainingDays)
		}
	}
}
