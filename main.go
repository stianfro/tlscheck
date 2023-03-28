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
)

func main() {
	var config *rest.Config
	var err error

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

	if _, err := os.Stat(kubeconfig); err == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		log.Fatalf("Failed to read kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list namespaces: %v", err)
	}

	fmt.Printf("%-30s %5s\n", "CERTIFICATE_NAME", "REMAINING_LIFETIME (days)")
	fmt.Println("---------------------------------------------------")

	for _, namespace := range namespaces.Items {
		secrets, err := clientset.CoreV1().Secrets(namespace.Name).List(context.Background(), v1.ListOptions{})
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
			fmt.Printf("%-30s %5d\n", secret.Name, remainingDays)
		}
	}
}
