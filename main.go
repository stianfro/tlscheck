package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
	}

	clientset, err := initKubernetesClient(kubeconfigPath)
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Register and start the metrics server
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Println("Metrics server listening on :2112")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	// Continuously update the metrics
	go func() {
		for {
			checkTLSCertificates(ctx, clientset)
		}
	}()

	// Wait for an interrupt signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	log.Println("Shutting down...")
}
