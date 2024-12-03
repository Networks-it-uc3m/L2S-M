# L2S-M in a Inter-Cluster scenario

>**Note: Work in progress** :wrench::wrench:
> This feature and repository is under development, keep it in mind when testing the application. For a stable version, refer to the main branch in the [L2S-M official repository](https://github.com/Networks-it-uc3m/L2S-M). 

## How it works
### Components in inter-cluster scenario:

<p align="center">
  <img src="../assets/inter-cluster-arch.svg" width="600">
</p>

### Sequence Diagram

<p align="center">
  <img src="../assets/inter-cluster-diagram.svg" width="600">
</p>


## YAML examples:

### Inter cluster network example:

```yaml
apiVersion: l2sm.k8s.local/v1
kind: L2SMNetwork
metadata:
  name: spain-network
spec:
  type: inter-vnet
  config: |
    {
      "provider": {
        "name": "uc3m",
        "domain": "idco.uc3m.es"
      },
      "accessList": ["public-key-1", "public-key-2"]
    }
  signature: sxySO0jHw4h1kcqO/LMLDgOoOeH8dOn8vZWv4KMBq0upxz3lcbl+o/36JefpEwSlBJ6ukuKiQ79L4rsmmZgglk6y/VL54DFyLfPw9RJn3mzl99YE4qCaHyEBANSw+d5hPaJ/I8q+AMtjrYpglMTRPf0iMZQMNtMd0CdeX2V8aZOPCQP75PsZkWukPdoAK/++y1vbFQ6nQKagvpUZfr7Ecb4/QY+hIAzepm6N6lNiFNTgj6lGTrFK0qCVfRhMD+vXbBP6xzZjB2N1nIheK9vx7kvj3HORjZ+odVMa+AOU5ShSKpzXTvknrtcRTcWWmXPNUZLoq5k3U+z1g1OTFcjMdQ====

```

### Pod creation and attachment

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: spain-network-signature
type: Opaque
data:
  public-key.pem: <signature-using-private-key-1>
```


```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mypod
  annotations:
    l2sm/networks: spain-network
spec:
  containers:
  - name: ping
    image: busybox
  volumes:
  - name: inter-vnet-signature
    secret:
      secretName: spain-network-signature
```

