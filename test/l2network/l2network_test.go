package l2network_test

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	NumCRs           = 50
	MaxCreateWorkers = 10 // Number of concurrent creators
	MaxWatchWorkers  = 10 // Number of concurrent watchers
)

func TestCreateL2Networks(t *testing.T) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	var namespace string
	flag.StringVar(&namespace, "namespace", "default", "Kubernetes namespace to create L2Networks in")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		t.Fatalf("Error building kubeconfig: %v", err)
	}

	// Customize the rate limiter to allow more concurrent requests
	config.QPS = 100
	config.Burst = 200

	// Create a dynamic client with the customized config
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		t.Fatalf("Error creating dynamic client: %v", err)
	}

	l2networkGVR := schema.GroupVersionResource{
		Group:    "l2sm.l2sm.k8s.local",
		Version:  "v1",
		Resource: "l2networks",
	}

	ctx := context.Background()

	// CR Creation Phase
	startTime := time.Now()

	createCh := make(chan int, NumCRs)
	var createWg sync.WaitGroup
	var createMu sync.Mutex
	crNames := make([]string, 0, NumCRs)
	createErrCh := make(chan error, NumCRs)

	// Start create workers
	for i := 0; i < MaxCreateWorkers; i++ {
		createWg.Add(1)
		go func() {
			defer createWg.Done()
			for idx := range createCh {
				crName := fmt.Sprintf("l2network-%d", idx+1)
				l2network := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "l2sm.l2sm.k8s.local/v1",
						"kind":       "L2Network",
						"metadata": map[string]interface{}{
							"name":      crName,
							"namespace": namespace,
						},
						"spec": map[string]interface{}{
							"type": "vnet",
						},
					},
				}

				_, err := dynClient.Resource(l2networkGVR).Namespace(namespace).Create(ctx, l2network, metav1.CreateOptions{})
				if err != nil {
					createErrCh <- fmt.Errorf("failed to create L2Network %s: %v", crName, err)
					return
				}

				// Protect shared slice
				createMu.Lock()
				crNames = append(crNames, crName)
				createMu.Unlock()
			}
		}()
	}

	// Send creation tasks
	for i := 0; i < NumCRs; i++ {
		createCh <- i
	}
	close(createCh)

	// Wait for creators to finish
	createWg.Wait()
	close(createErrCh)

	// Check for creation errors
	if len(crNames) != NumCRs {
		for err := range createErrCh {
			t.Error(err)
		}
		t.Fatalf("Expected to create %d L2Networks, but created %d", NumCRs, len(crNames))
	}

	elapsed := time.Since(startTime)
	t.Logf("Created %d L2Network CRs in %s", NumCRs, elapsed)

	// Status Checking Phase using Watches
	checkStartTime := time.Now()
	statusTimeout := 1 * time.Minute // Adjust as needed

	checkCh := make(chan string, NumCRs)
	var checkWg sync.WaitGroup
	checkErrCh := make(chan error, NumCRs)

	// Start watch workers
	for i := 0; i < MaxWatchWorkers; i++ {
		checkWg.Add(1)
		go func() {
			defer checkWg.Done()
			for crName := range checkCh {
				// Set up a watch for the specific CR
				watcher, err := dynClient.Resource(l2networkGVR).Namespace(namespace).Watch(ctx, metav1.ListOptions{
					FieldSelector:  fmt.Sprintf("metadata.name=%s", crName),
					TimeoutSeconds: int64Ptr(int64(statusTimeout.Seconds())),
				})
				if err != nil {
					checkErrCh <- fmt.Errorf("failed to set up watch for %s: %v", crName, err)
					continue
				}

				available := false
				for event := range watcher.ResultChan() {
					if event.Type == watch.Modified || event.Type == watch.Added {
						cr, ok := event.Object.(*unstructured.Unstructured)
						if !ok {
							continue
						}

						status, found, err := unstructured.NestedString(cr.Object, "status", "internalConnectivity")
						if err != nil || !found {
							continue
						}

						if status == "Available" {
							available = true
							break
						}
					}
				}

				watcher.Stop()

				if !available {
					checkErrCh <- fmt.Errorf("L2Network %s did not become 'Available' within timeout", crName)
				}
			}
		}()
	}

	// Send check tasks
	for _, crName := range crNames {
		checkCh <- crName
	}
	close(checkCh)

	// Wait for checkers to finish
	checkWg.Wait()
	close(checkErrCh)

	// Check for status errors
	if len(checkErrCh) > 0 {
		for err := range checkErrCh {
			t.Error(err)
		}
		t.Fatalf("Not all L2Networks reached 'Available' status")
	}

	checkElapsed := time.Since(checkStartTime)
	t.Logf("All %d L2Network CRs are in status 'Available' after %s", NumCRs, checkElapsed)

	// Deletion Phase
	deleteStartTime := time.Now()
	deleteCh := make(chan string, NumCRs)
	var deleteWg sync.WaitGroup
	deleteErrCh := make(chan error, NumCRs)

	// Start delete workers
	for i := 0; i < MaxCreateWorkers; i++ { // Reusing MaxCreateWorkers for deletion
		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			for crName := range deleteCh {
				err := dynClient.Resource(l2networkGVR).Namespace(namespace).Delete(ctx, crName, metav1.DeleteOptions{})
				if err != nil {
					deleteErrCh <- fmt.Errorf("failed to delete L2Network %s: %v", crName, err)
				}
			}
		}()
	}

	// Send delete tasks
	for _, crName := range crNames {
		deleteCh <- crName
	}
	close(deleteCh)

	// Wait for deletions to finish
	deleteWg.Wait()
	close(deleteErrCh)

	// Check for deletion errors
	for err := range deleteErrCh {
		t.Error(err)
	}

	deleteElapsed := time.Since(deleteStartTime)
	t.Logf("Deleted %d L2Network CRs in %s", NumCRs, deleteElapsed)
}

func int64Ptr(i int64) *int64 {
	return &i
}
