package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/crypto/ssh"
)

type Network struct {
	Name  string   `json:"name"`
	Ports []string `json:"ports"`
}

func main() {

	//var hostKey ssh.PublicKey
	// An SSH client is represented with a ClientConn.
	//
	// To authenticate with the remote server you must pass at least one
	// implementation of AuthMethod via the Auth field in ClientConfig,
	// and provide a HostKeyCallback.
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password("root123"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", "172.17.0.2:22", config)
	if err != nil {
		log.Fatal("Failed to dial: ", err)
	}
	defer client.Close()

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		log.Fatal("Failed to create session: ", err)
	}
	defer session.Close()

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run("/usr/bin/whoami"); err != nil {
		log.Fatal("Failed to run: " + err.Error())
	}
	fmt.Println(b.String())

	http.HandleFunc("/network", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getNetwork(w, r)
		case "POST":
			createNetwork(w, r)
		case "PUT":
			updateNetwork(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Server is running on port 8080...")
	http.ListenAndServe(":8080", nil)

}

func getNetwork(w http.ResponseWriter, r *http.Request) {
	//var network Network
	queryValues := r.URL.Query()
	name := queryValues.Get("name")
	//network = l2sm-get-network Name

	//json.NewEncoder(w).Encode(&network)
	fmt.Printf("GET %s", name)
}

func createNetwork(w http.ResponseWriter, r *http.Request) {
	var network Network
	if err := json.NewDecoder(r.Body).Decode(&network); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//l2sm-create-network network.Name
	w.WriteHeader(http.StatusCreated)
	fmt.Printf("POST")

}

func updateNetwork(w http.ResponseWriter, r *http.Request) {
	var network Network

	// l2sm-add-port network.Port
	// if err := json.NewDecoder(r.Body).Decode(&network); err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }
	// params := r.URL.Query()
	// name := params.Get("name")
	// port := params.Get("port")

	// for i, net := range networks {
	// 	if net.Name == name {
	// 		networks[i].Ports = append(networks[i].Ports, port)
	// 		w.WriteHeader(http.StatusOK)
	// 		return
	// 	}
	// }

	// http.Error(w, "Network not found", http.StatusNotFound)
	fmt.Printf("UPDATE")

}
