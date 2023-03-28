package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMainLogic(t *testing.T) {
	// Create a fake Kubernetes clientset
	clientset := fake.NewSimpleClientset()

	// Create a test namespace
	namespace := "test-namespace"
	_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	// Create a test secret with a self-signed certificate
	secretName := "test-secret"
	_, err = clientset.CoreV1().Secrets(namespace).Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			"tls.crt": pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: createSelfSignedCert(),
			}),
			"tls.key": []byte("fake-key"),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	// Retrieve the secrets in the test namespace
	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list secrets in namespace %s: %v", namespace, err)
	}

	found := false
	for _, secret := range secrets.Items {
		if secret.Name == secretName {
			found = true
			certBytes := secret.Data["tls.crt"]

			block, _ := pem.Decode(certBytes)
			if block == nil {
				t.Fatalf("Failed to parse PEM data for secret %s/%s", namespace, secret.Name)
			}

			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				t.Fatalf("Failed to parse certificate for secret %s/%s: %v", namespace, secret.Name, err)
			}

			remainingDays := int(time.Until(cert.NotAfter).Hours() / 24)
			if remainingDays <= 0 {
				t.Fatalf("Invalid remaining days for secret %s/%s: %d", namespace, secret.Name, remainingDays)
			}
		}
	}

	if !found {
		t.Fatalf("Secret %s/%s not found in the namespace", namespace, secretName)
	}
}

func createSelfSignedCert() []byte {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "example.com",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		panic(err)
	}

	return certBytes
}
