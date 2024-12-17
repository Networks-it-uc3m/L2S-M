package l2network_test

import (
	"context"
	"flag"
	"fmt"
	"os"
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

// Constants (defined once)
const (
	NumCRs            = 10 // Number of CRs to create
	MaxCreateWorkers  = 5  // Number of concurrent creators
	MaxWatchWorkers   = 5  // Number of concurrent watchers
	MaxUpdateWorkers  = 5  // Number of concurrent update workers
	MaxDeleteWorkers  = 5  // Number of concurrent delete workers
	StatusTimeout     = 1 * time.Minute
	UpdateTimeout     = 2 * time.Minute
	InitialSpecType   = "vnet"
	UpdatedSpecType   = "vlink"
	L2NetworkGroup    = "l2sm.l2sm.k8s.local"
	L2NetworkVersion  = "v1"
	L2NetworkResource = "l2networks"
)

// Package-level variables for flags
var (
	kubeconfig *string
	namespace  string
)

// TestMain handles flag parsing and setup before running tests
func TestMain(m *testing.M) {
	// Define flags
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(&namespace, "namespace", "default", "Kubernetes namespace to create L2Networks in")

	// Parse flags
	flag.Parse()

	// Run tests
	exitCode := m.Run()

	// Exit
	os.Exit(exitCode)
}

// Helper function to get a pointer to an int64
func int64Ptr(i int64) *int64 {
	return &i
}

// Helper function to build dynamic client
func buildDynamicClient(t *testing.T) dynamic.Interface {
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
	return dynClient
}

// Helper function to create L2Network CRs
func createL2Networks(t *testing.T, dynClient dynamic.Interface, gvr schema.GroupVersionResource, ctx context.Context) []string {
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
							"type": InitialSpecType,
						},
					},
				}

				_, err := dynClient.Resource(gvr).Namespace(namespace).Create(ctx, l2network, metav1.CreateOptions{})
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

	return crNames
}

// Helper function to watch and verify CR status
func verifyCRStatus(t *testing.T, dynClient dynamic.Interface, gvr schema.GroupVersionResource, ctx context.Context, crNames []string) {
	startTime := time.Now()

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
				watcher, err := dynClient.Resource(gvr).Namespace(namespace).Watch(ctx, metav1.ListOptions{
					FieldSelector:  fmt.Sprintf("metadata.name=%s", crName),
					TimeoutSeconds: int64Ptr(int64(StatusTimeout.Seconds())),
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

	elapsed := time.Since(startTime)
	t.Logf("All %d L2Network CRs are in status 'Available' after %s", NumCRs, elapsed)
}

// Helper function to update L2Network CRs and measure update duration
func updateL2Networks(t *testing.T, dynClient dynamic.Interface, gvr schema.GroupVersionResource, ctx context.Context, crNames []string) {
	startTime := time.Now()
	updateField := "spec.type"
	newFieldValue := UpdatedSpecType // New value to apply

	updateCh := make(chan string, NumCRs)
	var updateWg sync.WaitGroup
	updateErrCh := make(chan error, NumCRs)
	// To store individual update durations
	updateDurations := make(map[string]time.Duration)
	var updateMu sync.Mutex

	// Start update workers
	for i := 0; i < MaxUpdateWorkers; i++ {
		updateWg.Add(1)
		go func() {
			defer updateWg.Done()
			for crName := range updateCh {
				updateStart := time.Now()

				// Fetch the latest version of the CR
				cr, err := dynClient.Resource(gvr).Namespace(namespace).Get(ctx, crName, metav1.GetOptions{})
				if err != nil {
					updateErrCh <- fmt.Errorf("failed to get L2Network %s for update: %v", crName, err)
					continue
				}

				// Modify the desired field
				if err := unstructured.SetNestedField(cr.Object, newFieldValue, "spec", "type"); err != nil {
					updateErrCh <- fmt.Errorf("failed to set new field for L2Network %s: %v", crName, err)
					continue
				}

				// Apply the update
				_, err = dynClient.Resource(gvr).Namespace(namespace).Update(ctx, cr, metav1.UpdateOptions{})
				if err != nil {
					updateErrCh <- fmt.Errorf("failed to update L2Network %s: %v", crName, err)
					continue
				}

				// Start watching for the field to be updated
				watcher, err := dynClient.Resource(gvr).Namespace(namespace).Watch(ctx, metav1.ListOptions{
					FieldSelector:  fmt.Sprintf("metadata.name=%s", crName),
					TimeoutSeconds: int64Ptr(int64(UpdateTimeout.Seconds())),
				})
				if err != nil {
					updateErrCh <- fmt.Errorf("failed to set up watch for update of %s: %v", crName, err)
					continue
				}

				updated := false
				for event := range watcher.ResultChan() {
					if event.Type == watch.Modified || event.Type == watch.Added {
						updatedCR, ok := event.Object.(*unstructured.Unstructured)
						if !ok {
							continue
						}

						updatedValue, found, err := unstructured.NestedString(updatedCR.Object, "spec", "type")
						if err != nil || !found {
							continue
						}

						if updatedValue == newFieldValue {
							updated = true
							updateDuration := time.Since(updateStart)
							// Protect the map
							updateMu.Lock()
							updateDurations[crName] = updateDuration
							updateMu.Unlock()
							break
						}
					}
				}

				watcher.Stop()

				if !updated {
					updateErrCh <- fmt.Errorf("L2Network %s did not update '%s' to '%s' within timeout", crName, updateField, newFieldValue)
				}
			}
		}()
	}

	// Send update tasks
	for _, crName := range crNames {
		updateCh <- crName
	}
	close(updateCh)

	// Wait for update workers to finish
	updateWg.Wait()
	close(updateErrCh)

	// Check for update errors
	if len(updateDurations) != NumCRs {
		for err := range updateErrCh {
			t.Error(err)
		}
		t.Fatalf("Expected to update %d L2Networks, but updated %d", NumCRs, len(updateDurations))
	}

	updateElapsed := time.Since(startTime)
	t.Logf("Updated %d L2Network CRs in %s", NumCRs, updateElapsed)

	// Optionally, log individual update durations
	for crName, duration := range updateDurations {
		t.Logf("L2Network %s updated in %s", crName, duration)
	}
}

// Helper function to delete L2Network CRs
func deleteL2Networks(t *testing.T, dynClient dynamic.Interface, gvr schema.GroupVersionResource, ctx context.Context, crNames []string) {
	startTime := time.Now()

	deleteCh := make(chan string, NumCRs)
	var deleteWg sync.WaitGroup
	deleteErrCh := make(chan error, NumCRs)

	// Start delete workers
	for i := 0; i < MaxDeleteWorkers; i++ { // Reusing MaxDeleteWorkers for deletion
		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			for crName := range deleteCh {
				err := dynClient.Resource(gvr).Namespace(namespace).Delete(ctx, crName, metav1.DeleteOptions{})
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

	deleteElapsed := time.Since(startTime)
	t.Logf("Deleted %d L2Network CRs in %s", NumCRs, deleteElapsed)
}

// TestCreateAndConfigureL2Networks tests creation, update, and deletion of L2Network CRs
func TestCreateAndConfigureL2Networks(t *testing.T) {
	dynClient := buildDynamicClient(t)

	l2networkGVR := schema.GroupVersionResource{
		Group:    L2NetworkGroup,
		Version:  L2NetworkVersion,
		Resource: L2NetworkResource,
	}

	ctx := context.Background()

	// CR Creation Phase
	crNames := createL2Networks(t, dynClient, l2networkGVR, ctx)

	// Status Checking Phase
	verifyCRStatus(t, dynClient, l2networkGVR, ctx, crNames)

	// Update Phase
	updateL2Networks(t, dynClient, l2networkGVR, ctx, crNames)

	// Deletion Phase
	deleteL2Networks(t, dynClient, l2networkGVR, ctx, crNames)
}
