

## Work in progress
## Components in inter-cluster scenario:

<p align="center">
  <img src="../assets/inter-cluster-arch.svg" width="600">
</p>

## Sequence Diagram

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





Se avisa al operador, y este avisa a ambos controladores, siendo estos los que se encargan de comprobar la firma. -> Y ver si hay autorización

Si no es autorizado, el intent del NED no se crea en el controlador, si es autorizado, se hace intent desde NED, la interfaz veth que corresponda con la que el operador solicita.



Que habría que implementar-> 
IDCO:
  Doy por hecho que el idco funciona. 
  Base de datos con: public keys asociados a users. permisos asociados a users. Usar plataforma externa o internamente se define en el controller? Hacer un portal de autorizaciones externo a ONOS?

L2SM-Switch: 
  interfaces veth adicionales que conecten a los NED

NED: 
  como l2sm switch, pero que pueden tener varios controladores. con hostNetwork, van generando interfaces en el host para conecarse con L2S-M switch. un cable por pod o un cable por red?

L2S-M Operator:
  Cuando encienda que sepa si está en modo inter o no por un argumento.
  Según lo descrito:
    evento cuando se crea red inter
    evento cuando se añade pod 


L2S-M Client:
  A través de este se crean los networks. Con docker por ejemplo? o programa instalado por línea de comandos?


Si alguien quiere unirse a la red, attachea al pod, utilizando 

intercluster:
	owner de la red en cada cluster crea la red con:
		provider (idco concreto): campo de nombre y campo de dominio
		nombre
		accessList (diferente en cada cluster): se guarda clave pública de cada usuario. (el usuario tiene su clave privada guardada en su pc por ej)
			clave: tiene identificador, hash, una firma digital.
		timestamp
	
	IDCO implementar: 
		cuando hay un request, ver si la clave está bien firmada.
		guardar la de redes:
			RED:
				Cluster1
					AccessList
				Cluster2
					AccessList
	
	
	1. Se crea red inter
	2 se pide al idco que red intra corresponde, (ahi se haria la utorizacion)
	3 se guarda el mappeo de red inter a intra, y se juntan esos pares desde el operador
	4 cuando alguien despliega, se usa la red intra
	
	de momento security parameters, dejarlo en abierto.
	hacer un dibujillo.
1. 
