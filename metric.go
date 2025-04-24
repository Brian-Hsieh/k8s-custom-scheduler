package main

import (
	"context"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type RegionLatency struct {
	singapore float64
	sydney    float64
	tokyo     float64
}

type NodeLoad struct {
	w0 float64
	w1 float64
	w2 float64
	w3 float64
	w4 float64
	w5 float64
}

func setUpDynamicClient() (*dynamic.DynamicClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	fmt.Println("Connected to k8s API")
	return dynamicClient, nil
}

func getRegionLatencies(dynamicClient *dynamic.DynamicClient) RegionLatency {
	// fallback values
	latencies := RegionLatency{
		singapore: 1000,
		sydney:    1000,
		tokyo:     1000,
	}

	datadogMetricGVR := schema.GroupVersionResource{
		Group:    "datadoghq.com",
		Version:  "v1alpha1",
		Resource: "datadogmetrics",
	}

	regions := []string{"latency-singapore", "latency-sydney", "latency-tokyo"}

	for _, region := range regions {
		metric, err := dynamicClient.Resource(datadogMetricGVR).Namespace("datadog").Get(context.TODO(), region, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error fetching metric for %s: %v\n", region, err)
			continue
		}

		latency, found, err := unstructured.NestedString(metric.Object, "status", "currentValue")
		if err != nil || !found {
			fmt.Printf("Latency value not found for %s\n", region)
			continue
		}

		value, err := strconv.ParseFloat(latency, 64)
		if err != nil {
			fmt.Printf("Error when converting string to float: %v\n", err)
			continue
		}

		switch region {
		case "latency-singapore":
			latencies.singapore = value
		case "latency-sydney":
			latencies.sydney = value
		case "latency-tokyo":
			latencies.tokyo = value
		}

		fmt.Printf("Latency for %s: %.2f\n", region, value)
	}

	return latencies
}

func getNodeLoad(dynamicClient *dynamic.DynamicClient) NodeLoad {
	// fallback values

	loads := NodeLoad{
		w0: 50,
		w1: 50,
		w2: 50,
		w3: 50,
		w4: 50,
		w5: 50,
	}

	datadogMetricGVR := schema.GroupVersionResource{
		Group:    "datadoghq.com",
		Version:  "v1alpha1",
		Resource: "datadogmetrics",
	}

	nodes := []string{"tokyo-worker1", "masters-slave", "masters-slave2", "singapore-worker1", "singapore-worker2", "singapore-worker3"}

	for _, node := range nodes {
		metric, err := dynamicClient.Resource(datadogMetricGVR).Namespace("datadog").Get(context.TODO(), node, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error fetching metric for %s: %v\n", node, err)
			continue
		}

		load, found, err := unstructured.NestedString(metric.Object, "status", "currentValue")
		if err != nil || !found {
			fmt.Printf("Load value not found for %s\n", node)
			continue
		}

		value, err := strconv.ParseFloat(load, 64)
		if err != nil {
			fmt.Printf("Error when converting string to float: %v\n", err)
			continue
		}

		switch node {
		case "tokyo-worker1":
			loads.w0 = value
		case "masters-slave":
			loads.w1 = value
		case "masters-slave2":
			loads.w2 = value
		case "singapore-worker1":
			loads.w3 = value
		case "singapore-worker2":
			loads.w4 = value
		case "singapore-worker3":
			loads.w5 = value
		}

		fmt.Printf("Load for %s: %.2f\n", node, value)
	}

	return loads
}
