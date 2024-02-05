


Ejemplo de network inter: 

```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: sample-inter-network
spec:
  config: '{
      "cniVersion": "0.3.0",
      "type": "l2sm",
      "device": "l2sm-vNet",
      "kind": {
        "inter": {
          "provider": {
            "name": "<idco-name>",
            "domain": "<domain-name>"
          },
          "accessList": ["<public-key-1>","<public-key-2>",...,"<public-key-N>"] 
          ]
        }
      }
    }'
```
Hay un NED conectado al L2S-M switch del nodo master con 10 interfaces veth (como el NED es hostNetwork, nos podemos permitir crear las interfaces y conectarlas directamente -> Necesario que l2sm-switch se despliegue más tarde).

Se crea esta red en cada clúster usando el L2S-M k8s Client. Es necesario para esto:
    - Que cada cluster tenga previamente un cluster role para poder dar permisos de crear network attachment definitions.

Se crea un network con el mismo nombre dentro del host. Operador avisa al idco de que se ha conectado a la interfaz, se dice cual es el veth empleado. El idco anota, pero no añade aún.

Se crea un secret dentro del cluster con la firma de la public key. 

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sample-one-authorization-key
type: Opaque
data:
  public-key.pem: <firma>
```

Se attachea al pod, en el campo de spec. Quedaría así:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mypod
  annotations:
    k8s.v1.cni.cncf.io/networks: "sample-inter-network"
spec:
  containers:
  - name: ping
    image: busybox
  volumes:
  - name: authorization-key-volume
    secret:
      secretName: sample-one-authorization-key
```




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