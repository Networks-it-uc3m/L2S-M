import kopf
import os
import sys
import json
import subprocess
import secrets
import kubernetes
from subprocess import CalledProcessError
from random import randrange
from kubernetes import client, config
import pymysql
import random
import time
import requests
import re
import sys
from requests.auth import HTTPBasicAuth

databaseIP = "127.0.0.1"
baseControllerUrl = 'http://' + os.environ['CONTROLLER_IP'] + ':8181' + '/onos/v1'
def beginSessionController(baseControllerUrl,username,password):

  # Create a session with basic authentication
  auth = HTTPBasicAuth(username, password)

  session = requests.Session()
  session.auth = auth

  #Check if connection is possible
  response = session.get(baseControllerUrl + '/l2sm/networks/status')
  if response.status_code == 200:
    # Successful request
    print("Initialized session between operator and controller.")
    return session
  else:
    print("Could not initialize session with l2sm-controller")
    sys.exit()
    return None
  

  
session = beginSessionController(baseControllerUrl,"karaf","karaf")

def getSwitchId(cur, node):
    switchQuery = "SELECT * FROM switches WHERE node = '%s'" % (node)
    cur.execute(switchQuery)
    switchRecord = cur.fetchone()

    if switchRecord is not None:
        switchId = switchRecord[0]

        if switchId is not None:
            return switchId  # If openflowId is already set, return it

    # If openflowId is not set, make a request to get the information from the API
    response = session.get(baseControllerUrl + '/devices')
    devices = response.json().get('devices', [])

    for device in devices:
        if 'id' in device and 'annotations' in device and 'managementAddress' in device['annotations']:
            if device['annotations']['managementAddress'] == switchRecord[1]:
                openflowId = device['id']
                switchId = openflowId

                # Save the openflowId in the database
                updateQuery = "UPDATE switches SET openflowId = '%s' WHERE node = '%s'" % (switchId, node)
                cur.execute(updateQuery)

                return switchId  # Return the openflowId
    return None  # Return None if no matching device is found
  
#POPULATE DATABASE ENTRIES WHEN A NEW L2SM POD IS CREATED (A NEW NODE APPEARS)
@kopf.on.create('pods.v1', labels={'l2sm-component': 'l2sm-switch'})
def build_db(body, logger, annotations, **kwargs):
    db = pymysql.connect(host=databaseIP,user="l2sm",password="l2sm;",db="L2SM")
    cur = db.cursor()
    
    #CREATE TABLES IF THEY DO NOT EXIST
    table1 = "CREATE TABLE IF NOT EXISTS networks (network TEXT NOT NULL, id TEXT NOT NULL);"
    table2 = "CREATE TABLE IF NOT EXISTS interfaces (interface TEXT NOT NULL, node TEXT NOT NULL, network TEXT, pod TEXT);"
    table3 = "CREATE TABLE IF NOT EXISTS switches (openflowId TEXT, ip TEXT, node TEXT NOT NULL);"
    cur.execute(table1)
    cur.execute(table2)
    cur.execute(table3)
    db.commit()
    
    #MODIFY THE END VALUE TO ADD MORE INTERFACES
    values = []
    for i in range(1,11):
      values.append(['veth'+str(i), body['spec']['nodeName'], '-1', ''])
    sqlInterfaces = "INSERT INTO interfaces (interface, node, network, pod) VALUES (%s, %s, %s, %s)"
    cur.executemany(sqlInterfaces, values)
    db.commit()

    #ADD The switch identification to the database, without the of13 id yet, as it may not be connected yet.
    sqlSwitch = "INSERT INTO switches (node) VALUES ('" + body['spec']['nodeName'] + "')"
    cur.execute(sqlSwitch)
    db.commit()
    
    db.close()
    logger.info(f"Node {body['spec']['nodeName']} has been registered in the operator")

@kopf.on.field('pods.v1', labels={'l2sm-component': 'l2sm-switch'}, field='status.podIP')
def update_db(body, logger, annotations, **kwargs):
    if 'status' in body and 'podIP' in body['status']:
      db = pymysql.connect(host=databaseIP,user="l2sm",password="l2sm;",db="L2SM")
      cur = db.cursor()
      updateQuery = "UPDATE switches SET ip = '%s', OpenFlowId = NULL WHERE node = '%s'" % (body['status']['podIP'], body['spec']['nodeName'])
      cur.execute(updateQuery)
      db.commit()
      db.close()
      logger.info(f"Updated switch ip")


#UPDATE DATABASE WHEN NETWORK IS CREATED, I.E: IS A MULTUS CRD WITH OUR DUMMY INTERFACE PRESENT IN ITS CONFIG
#@kopf.on.create('NetworkAttachmentDefinition', field="spec.config['device']", value='l2sm-vNet')
@kopf.on.create('NetworkAttachmentDefinition', when=lambda spec, **_: '"device": "l2sm-vNet"' in spec['config'])
def create_vn(spec, name, namespace, logger, **kwargs):
  
    db = pymysql.connect(host=databaseIP,user="l2sm",password="l2sm;",db="L2SM")
    cur = db.cursor()
    id = secrets.token_hex(32)
    sql = "INSERT INTO networks (network, id) VALUES ('%s', '%s')" % (name.strip(), id.strip())
    cur.execute(sql)
    db.commit()
    db.close()
    
    # Create the network in the controller, using a post request
    data = {
        "networkId": name.strip()
    }
    # json_payload = {
    #     "Content-Type": "application/json",
    #     "data": payload
    # }
    response = session.post(baseControllerUrl + '/l2sm/networks', json=data)

    # Check the response status
    if response.status_code == 204:
        logger.info(f"Network has been created")
        #print("Response:", response.json())
    else:
        # Handle errors
        logger.info(f"Network could not be created, check controller status")



#ASSIGN POD TO NETWORK (TRIGGERS ONLY IF ANNOTATION IS PRESENT)
@kopf.on.create('pods.v1', annotations={'k8s.v1.cni.cncf.io/networks': kopf.PRESENT})
def pod_vn(body, name, namespace, logger, annotations, **kwargs):
    #GET MULTUS INTERFACES IN THE DESCRIPTOR
    #IN QUARANTINE: SLOWER THAN MULTUS!!!!!
    time.sleep(random.uniform(0,0.8)) #Make sure the database is not consulted at the same time to avoid overlaping

    multusInt = annotations.get('k8s.v1.cni.cncf.io/networks').split(",")
    #VERIFY IF NETWORK IS PRESENT IN THE CLUSTER
    api = client.CustomObjectsApi()
    items = api.list_namespaced_custom_object('k8s.cni.cncf.io', 'v1', namespace, 'network-attachment-definitions').get('items')
    resources = []
    # NETWORK POSITION IN ANNOTATION
    network = []

    #FIND OUR NETWORKS IN MULTUS
    for i in items:
      if '"device": "l2sm-vNet"' in i['spec']['config']:
        resources.append(i['metadata']['name'])

    for k in range(len(multusInt)):
      multusInt[k] = multusInt[k].strip()
      if multusInt[k] in resources:
        network.append(k)

    #IF THERE ARE NO NETWORKS, LET MULTUS HANDLE THIS
    if not network:
      return

    #CHECK IF NODE HAS FREE VIRTUAL INTERFACES LEFT
    v1 = client.CoreV1Api()
    ret = v1.read_namespaced_pod(name, namespace)
    node = body['spec']['nodeName']

    db = pymysql.connect(host=databaseIP,user="l2sm",password="l2sm;",db="L2SM")
    nsql = "SELECT * FROM interfaces WHERE node = '%s' AND network = '-1'" % (node.strip())
    cur = db.cursor()
    cur.execute(nsql)
    data = cur.fetchall()
    if not data or len(data)<len(network):
      db.close()
      raise kopf.PermanentError("l2sm could not deploy the pod: Node " + node.strip() + "has no free interfaces left")

    #IF THERE IS ALREADY A MULTUS ANNOTATION, APPEND IT TO THE END.
    interface_to_attach = []
    network_array = []
    j = 0
    for interface in data[0:len(network)]:
      network_array.append(multusInt[network[j]])
      multusInt[network[j]] = interface[0].strip()
      interface_to_attach.append(interface[0].strip())
      j = j + 1

    ret.metadata.annotations['k8s.v1.cni.cncf.io/networks'] = ', '.join(multusInt)

    #PATCH NETWORK WITH ANNOTATION
    v1.patch_namespaced_pod(name, namespace, ret)

    #GET NETWORK ID'S
    #for j in items:
    #  if network in j['metadata']['name']:
    #    idsql = "SELECT id FROM networks WHERE network = '%s'" % (network.strip())
    #    cur.execute(idsql)
    #    retrieve = cur.fetchone()
    #    networkN = retrieve[0].strip()
    #    break

    switchId = getSwitchId(cur, node) # TODO: diferenciar caso en el que es un veth el conectado y el de que es una red de vdd.

    if switchId is None:
      logger.info(f"The l2sm switch is not connected to controller. Not connecting the pod")
      return
    vethPattern = re.compile(r'\d+$')
    portNumbers = [int(vethPattern.search(interface).group()) for interface in interface_to_attach]

    for m in range(len(network)):
      sql = "UPDATE interfaces SET network = '%s', pod = '%s' WHERE interface = '%s' AND node = '%s'" % (network_array[m], name, interface_to_attach[m], node)
      cur.execute(sql)

      payload = {
        "networkId": network_array[m],
        "networkEndpoints": [switchId + '/' + str(portNumbers[m])]
      }
    
     
      
      response = session.post(baseControllerUrl + '/l2sm/networks/port', json=payload)



    db.commit()
    db.close()
    
   
    


    # Check the response status
    if response.status_code == 200:
        # Successful request
        print("Request successful!")
    else:
        # Handle errors
        print(f"Error: {response.status_code}")
    logger.info(f"Pod {name} attached to network {network_array}")


#UPDATE DATABASE WHEN POD IS DELETED
@kopf.on.delete('pods.v1', annotations={'k8s.v1.cni.cncf.io/networks': kopf.PRESENT})
def dpod_vn(name, logger, **kwargs):
    db = pymysql.connect(host=databaseIP,user="l2sm",password="l2sm;",db="L2SM")
    cur = db.cursor()
    sql = "UPDATE interfaces SET network = '-1', pod = '' WHERE pod = '%s'" % (name)
    cur.execute(sql)
    db.commit()
    db.close()
    logger.info(f"Pod {name} removed")

#UPDATE DATABASE WHEN NETWORK IS DELETED
@kopf.on.delete('NetworkAttachmentDefinition', when=lambda spec, **_: '"device": "l2sm-vNet"' in spec['config'])
def delete_vn(spec, name, logger, **kwargs):
    db = pymysql.connect(host=databaseIP,user="l2sm",password="l2sm;",db="L2SM")
    cur = db.cursor()
    sql = "DELETE FROM networks WHERE network = '%s'" % (name)
    cur.execute(sql)
    
    
    response = session.delete(baseControllerUrl + '/l2sm/networks/' + name)
    
    if response.status_code == 204:
        # Successful request
      logger.info(f"Network has been deleted")
      db.commit()
    else:
        # Handle errors
      logger.info(f"Error: {response.status_code}")
    db.close()

#DELETE DATABASE ENTRIES WHEN A NEW L2SM SWITCH IS DELETED (A NEW NODE GETS OUT OF THE CLUSTER)
@kopf.on.delete('pods.v1', labels={'l2sm-component': 'l2sm-switch'})
def remove_node(body, logger, annotations, **kwargs):
    db = pymysql.connect(host=databaseIP,user="l2sm",password="l2sm;",db="L2SM")
    cur = db.cursor()
    sql = "DELETE FROM interfaces WHERE node = '%s'" % (body['spec']['nodeName'])
    switchSql = "DELETE FROM switches WHERE node = '%s'" % (body['spec']['nodeName'])
    cur.execute(sql)
    cur.execute(switchSql)
    db.commit()
    db.close()
    logger.info(f"Node {body['spec']['nodeName']} has been deleted from the cluster")

