# L2S-M Pod Quarantine Example

This example creates one production network with four pods and one quarantine
network with stricter IDS rules. Three pods act as clients and one pod acts as a
server. One client starts a port scan after startup, and a
`QuarantinePodRequest` moves only that attacker pod from the production network
to the quarantine network.

Run the commands from the repository root.

## Deploy

Create the IDS rule ConfigMaps, both networks, and the four pods:

```bash
kubectl apply -f ./examples/quarantine/00-ids-rules.yaml
kubectl apply -f ./examples/quarantine/01-networks.yaml
kubectl apply -f ./examples/quarantine/02-pods.yaml
```

The attacker waits 45 seconds, then repeatedly scans the server:

```bash
kubectl logs -f quarantine-attacker
```

## Schedule the Quarantine Request

Apply the request after the attack has started. This command waits 90 seconds so
the IDS can observe the scan before the pod is quarantined:

```bash
sleep 90 && kubectl apply -f ./examples/quarantine/03-quarantine-attacker.yaml
```

The request selects the source network by labels and selects only the attacker
pod by the `security.l2sm/demo-role=attacker` label.

## Verify

Check the request status:

```bash
kubectl get quarantinepodrequest quarantine-demo-attacker -o yaml
```

Check that the attacker pod annotation now points at the quarantine network:

```bash
kubectl get pod quarantine-attacker -o go-template='{{ index .metadata.annotations "l2sm/networks" }}{{ "\n" }}'
```

The expected annotation contains `quarantine-demo-isolation`. The other two
clients and the server remain attached to `quarantine-demo-production`.

## Cleanup

```bash
kubectl delete -f ./examples/quarantine/03-quarantine-attacker.yaml --ignore-not-found
kubectl delete -f ./examples/quarantine/02-pods.yaml
kubectl delete -f ./examples/quarantine/01-networks.yaml
kubectl delete -f ./examples/quarantine/00-ids-rules.yaml
```
