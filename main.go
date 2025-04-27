package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	podBuffer   []*corev1.Pod
	bufferMutex sync.Mutex
)

func setUpClientSetAndDynamicClient() (*kubernetes.Clientset, *dynamic.DynamicClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	fmt.Println("Connected to k8s API")
	return clientset, dynamicClient, nil
}

func isNodeReady(node *corev1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func isNodeUnderPressure(node *corev1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if (cond.Type == corev1.NodeDiskPressure || cond.Type == corev1.NodeMemoryPressure || cond.Type == corev1.NodePIDPressure) && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func mapFromFloatToNodeName(v float64) string {
	if v <= 1 {
		return "tokyo-worker1"
	} else if v <= 2 {
		return "masters-slave"
	} else if v <= 3 {
		return "masters-slave2"
	} else if v <= 4 {
		return "singapore-worker1"
	} else if v <= 5 {
		return "singapore-worker2"
	} else {
		return "singapore-worker3"
	}
}

func findBestNodesPSO(clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, pods []*corev1.Pod) []string {
	regionLatency := getRegionLatencies(dynamicClient)
	nodeLoad := getNodeLoad(dynamicClient)
	re := runPSO(regionLatency, nodeLoad, len(pods), 100, 10)
	fmt.Printf("PSO result (values): %v\n", re)
	nodeNames := make([]string, len(pods))
	for i, v := range re {
		nodeNames[i] = mapFromFloatToNodeName(v)
	}
	fmt.Printf("PSO result (node names): %v\n", nodeNames)
	return nodeNames
}

func schedulePodToNode(pod *corev1.Pod, nodeName string, clientset *kubernetes.Clientset) error {
	fmt.Println("scheduling pod to node: " + nodeName)

	binding := &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		Target: corev1.ObjectReference{
			Kind: "Node",
			Name: nodeName,
		},
	}
	err := clientset.CoreV1().Pods(pod.Namespace).Bind(context.Background(), binding, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Error when scheduling pods to node: %v\n", err.Error())
	}
	return err
}

func watchForPodsAndSchedule(clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient) {
	list := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		"app",
		fields.OneTermEqualSelector("spec.nodeName", ""),
	)

	options := cache.InformerOptions{
		ListerWatcher: list,
		ObjectType:    &corev1.Pod{},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*corev1.Pod)
				if pod.Spec.SchedulerName != "custom-scheduler" {
					return
				}
				bufferMutex.Lock()
				podBuffer = append(podBuffer, pod)
				bufferMutex.Unlock()
			},
		},
	}

	_, controller := cache.NewInformerWithOptions(options)

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(stop)

	// Periodically process the buffer to run PSO on multiple pods at once
	go func() {
		ticker := time.NewTicker(5 * time.Second) // tune this delay
		for range ticker.C {
			bufferMutex.Lock()
			if len(podBuffer) == 0 {
				bufferMutex.Unlock()
				continue
			}

			fmt.Printf("%s pods ready to be scheduled.\n", len(podBuffer))

			pods := append([]*corev1.Pod(nil), podBuffer...)
			podBuffer = []*corev1.Pod{}
			bufferMutex.Unlock()

			// Run PSO to find the best node assignments for these pods
			nodesAssignment := findBestNodesPSO(clientset, dynamicClient, pods)

			// Schedule each pod based on the PSO result
			for i, pod := range pods {
				nodeName := nodesAssignment[i]
				if err := schedulePodToNode(pod, nodeName, clientset); err != nil {
					fmt.Printf("Failed to schedule pod %s to node %s: %v\n", pod.Name, nodeName, err)
				}
			}
		}
	}()

	select {}
}

func main() {
	clientSet, dynamicClient, err := setUpClientSetAndDynamicClient()
	if err != nil {
		panic(err.Error())
	}

	if err := loadConfig(); err != nil {
		fmt.Println("Error when loading env var", err)
	}
	fmt.Printf("Config: %+v\n", config)

	watchForPodsAndSchedule(clientSet, dynamicClient)
}
