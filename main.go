package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/tabwriter"
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
		if err != nil {
			log.Println(err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Println(err)
		}
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%-30s\t%-30s\t%s\t%5s\n", "SECRET", "NAMESPACE", "ISSUER", "REMAINING_LIFETIME (days)")

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

			remainingDays := int(time.Until(cert.NotAfter).Hours() / 24)
			issuer := cert.Issuer.String()
			// Skip if cert.Issuer is openshift ca
			fmt.Fprintf(w, "%-30s\t%-30s\t%s\t%5d\n", secret.Name, secret.Namespace, issuer, remainingDays)
		}
	}

	w.Flush()
}
