package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"l2sm.local/ovs-switch/pkg/ovs"
)

// Mock implementation of ovs.Bridge for testing purposes
type MockBridge struct {
	Name       string
	Controller string
	Protocol   string
	DatapathId string
}

func (b *MockBridge) AddPort(port string) error {
	return nil
}

func (b *MockBridge) String() string {
	return fmt.Sprintf("MockBridge{Name: %s, Controller: %s, Protocol: %s, DatapathId: %s}", b.Name, b.Controller, b.Protocol, b.DatapathId)
}

// Override ovs.NewBridge for testing
var NewBridge = func(b ovs.Bridge) (ovs.Bridge, error) {
	return ovs.Bridge{
		Name:       b.Name,
		Controller: b.Controller,
		Protocol:   b.Protocol,
		DatapathId: b.DatapathId,
	}, nil
}

func TestTakeArguments(t *testing.T) {
	// Backup original command line arguments
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		args        []string
		expectedErr error
	}{
		{[]string{"cmd", "-n_veths", "5", "-controller_ip", "192.168.1.1", "-switch_name", "switch1"}, nil},
		{[]string{"cmd", "-n_veths", "5", "-controller_ip", ""}, errors.New("controller IP is not defined")},
	}

	for _, test := range tests {
		os.Args = test.args
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		_, _, _, err := takeArguments()
		if err != nil && err.Error() != test.expectedErr.Error() {
			t.Errorf("Expected error: %v, got: %v", test.expectedErr, err)
		}
	}
}

func TestInitializeSwitch(t *testing.T) {
	// Mock exec.Command for testing
	oldExecCommand := exec.Command
	defer func() { exec.Command = oldExecCommand }()

	exec.Command = func(name string, arg ...string) *exec.Cmd {
		cmd := oldExecCommand("echo", "192.168.1.1")
		return cmd
	}

	tests := []struct {
		switchName   string
		controllerIP string
		expectedErr  error
	}{
		{"switch1", "192.168.1.1", nil},
		{"switch1", "invalid-ip", nil},
	}

	for _, test := range tests {
		bridge, err := initializeSwitch(test.switchName, test.controllerIP)
		if err != nil && err.Error() != test.expectedErr.Error() {
			t.Errorf("Expected error: %v, got: %v", test.expectedErr, err)
		}
		if bridge == nil {
			t.Errorf("Expected bridge to be initialized, got nil")
		}
	}
}

func TestGenerateDatapathID(t *testing.T) {

	datapathID := generateDatapathID("pod")
	fmt.Println(datapathID)
}
