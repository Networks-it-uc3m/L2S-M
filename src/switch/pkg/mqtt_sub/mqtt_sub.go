package main

import (
	"encoding/json"
	"log"

	"github.com/Networks-it-uc3m/l2sm-switch/pkg/ovs"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Command struct {
	Action string     `json:"action"`
	Bridge ovs.Bridge `json:"bridge,omitempty"`
	Port   string     `json:"port,omitempty"`
	Vxlan  ovs.Vxlan  `json:"vxlan,omitempty"`
}

var broker = "tcp://mqtt_broker_ip:1883"
var topic = "ovs/commands"

func LaunchSubscriber() {
	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID("go_mqtt_client")
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		handleMessage(msg.Payload())
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// Block main routine forever
	select {}
}

func handleMessage(payload []byte) {
	var cmd Command
	err := json.Unmarshal(payload, &cmd)
	if err != nil {
		log.Printf("Error decoding command: %v", err)
		return
	}

	switch cmd.Action {
	case "create_bridge":
		bridge, err := ovs.NewBridge(cmd.Bridge)
		if err != nil {
			log.Printf("Error creating bridge: %v", err)
		} else {
			log.Printf("Bridge created: %+v", bridge)
		}
	case "add_port":
		bridge := ovs.FromName(cmd.Bridge.Name)
		err := bridge.AddPort(cmd.Port)
		if err != nil {
			log.Printf("Error adding port: %v", err)
		} else {
			log.Printf("Port added: %s", cmd.Port)
		}
	case "create_vxlan":
		bridge := ovs.FromName(cmd.Bridge.Name)
		err := bridge.CreateVxlan(cmd.Vxlan)
		if err != nil {
			log.Printf("Error creating VXLAN: %v", err)
		} else {
			log.Printf("VXLAN created: %+v", cmd.Vxlan)
		}
	default:
		log.Printf("Unknown action: %s", cmd.Action)
	}
}
