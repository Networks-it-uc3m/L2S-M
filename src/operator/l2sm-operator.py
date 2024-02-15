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

database_ip = os.environ['DATABASE_IP']
database_username = os.environ['MYSQL_USER']
database_password = os.environ['MYSQL_PASSWORD']
database_name = os.environ['MYSQL_DATABASE']

base_controller_url = 'http://' + os.environ['CONTROLLER_IP'] + ':8181' + '/onos/v1'

print(base_controller_url)
print(database_ip)

def begin_session_controller(base_controller_url,username,password):

  # Create a session with basic authentication
  auth = HTTPBasicAuth(username, password)

  session = requests.Session()
  session.auth = auth

  #Check if connection is possible
  response = session.get(base_controller_url + '/l2sm/networks/status')
  if response.status_code == 200:
    # Successful request
    print("Initialized session between operator and controller.")
    return session
  else:
    print("Could not initialize session with l2sm-controller")
    sys.exit()
  

  
session = begin_session_controller(base_controller_url,"karaf","karaf")

def get_openflow_id(node_name):
    connection = pymysql.connect(host=database_ip,
                                user=database_username,
                                password=database_password,
                                database=database_name,
                                cursorclass=pymysql.cursors.DictCursor)
    try: 
        with connection.cursor() as cursor:
            switch_query = "SELECT id, openflowId, ip FROM switches WHERE node_name = %s"
            cursor.execute(switch_query, (node_name,))
            switch_record = cursor.fetchone()

            print(switch_record)
            if switch_record is not None:
                switchId = switch_record['openflowId']

                if switchId is not None:
                    return switchId  # If openflowId is already set, return it

            # If openflowId is not set, make a request to get the information from the API
            response = session.get(base_controller_url + '/devices')
            devices = response.json().get('devices', [])

            for device in devices:
                if 'id' in device and 'annotations' in device and 'managementAddress' in device['annotations']:
                    if device['annotations']['managementAddress'] == switch_record['ip']:
                        openflowId = device['id']
                        switchId = openflowId

                        # Save the openflowId in the database
                        updateQuery = "UPDATE switches SET openflowId = %s WHERE id = %s"
                        cursor.execute(updateQuery, (openflowId, switch_record['id']))
                        connection.commit()

                        return switchId  # Return the openflowId
            connection.commit()
    finally:
        connection.close()
                    
    return None  # Return None if no matching device is found
  
#POPULATE DATABASE ENTRIES WHEN A NEW L2SM SWITCH IS CREATED (A NEW NODE APPEARS)
@kopf.on.create('pods.v1', labels={'l2sm-component': 'l2sm-switch'})
def build_db(body, logger, annotations, **kwargs):
    connection = pymysql.connect(host=database_ip,
                             user=database_username,
                             password=database_password,
                             database=database_name,
                             cursorclass=pymysql.cursors.DictCursor)
    if 'spec' in body and 'nodeName' in body['spec']:
      try:
        with connection.cursor() as cursor:
          # Step 1: Check if the switch already exists
          select_switch_sql = "SELECT id FROM switches WHERE node_name = %s;"
          cursor.execute(select_switch_sql, body['spec']['nodeName'])
          result = cursor.fetchone()
          
          if result:
              # Switch exists, so update it (if needed)
              switch_id = result['id']
              # Example update (modify as needed)
              # update_switch_sql = "UPDATE switches SET openflowId = %s, IP = %s WHERE id = %s;"
              # cursor.execute(update_switch_sql, (newOpenflowId, newIP, switch_id))
          else:
              # Step 2: Insert a switch since it doesn't exist
              insert_switch_sql = "INSERT INTO switches (node_name, openflowId, IP) VALUES (%s, NULL, NULL);"
              cursor.execute(insert_switch_sql, body['spec']['nodeName'])
              switch_id = cursor.lastrowid
              # Step 3: Insert interfaces
              for i in range(1, 11):
                  interface_name = f"veth{i}"
                  # Consider adding a check here to see if the interface already exists for the switch
                  insert_interface_sql = "INSERT INTO interfaces (name, switch_id) VALUES (%s, %s);"
                  cursor.execute(insert_interface_sql, (interface_name, switch_id))
              
          # Commit the changes
          connection.commit()
      finally:
        connection.close()
      logger.info(f"Node {body['spec']['nodeName']} has been registered in the operator")
    else:
        raise kopf.TemporaryError("The Pod is not yet ready", delay=5)

@kopf.on.field('pods.v1', labels={'l2sm-component': 'l2sm-switch'}, field='status.podIP')
def update_db(body, logger, annotations, **kwargs):
    if 'status' in body and 'podIP' in body['status']:
      connection = pymysql.connect(host=database_ip,
                             user=database_username,
                             password=database_password,
                             database=database_name,
                             cursorclass=pymysql.cursors.DictCursor)
      try:
        with connection.cursor() as cursor:
          updateQuery = "UPDATE switches SET ip = '%s', OpenFlowId = NULL WHERE node_name = '%s'" % (body['status']['podIP'], body['spec']['nodeName'])
          cursor.execute(updateQuery)
          connection.commit()
      finally:
        connection.close()
      logger.info(f"Updated switch ip")


#UPDATE DATABASE WHEN NETWORK IS CREATED, I.E: IS A MULTUS CRD WITH OUR L2SM INTERFACE PRESENT IN ITS CONFIG
#@kopf.on.create('NetworkAttachmentDefinition', field="spec.config['device']", value='l2sm-vNet')
@kopf.on.create('NetworkAttachmentDefinition', when=lambda spec, **_: '"type": "l2sm"' in spec['config'])
def create_vn(spec, name, namespace, logger, **kwargs):
    
    # Database connection setup
    connection = pymysql.connect(host=database_ip,
                                 user=database_username,
                                 password=database_password,
                                 database=database_name,
                                 cursorclass=pymysql.cursors.DictCursor)
    try:
        # Start database transaction
        with connection.cursor() as cursor:
            sql = "INSERT INTO networks (name, type) VALUES (%s, %s) ON DUPLICATE KEY UPDATE name = VALUES(name), type = VALUES(type)"
            cursor.execute(sql, (name.strip(), "vnet"))
      
        # Prepare data for the REST API call
        data = {"networkId": name.strip()}
        response = session.post(base_controller_url + '/l2sm/networks', json=data)
        
        # Check the response status
        if response.status_code == 204:
            # Commit database changes only if the network is successfully created in the controller
            connection.commit()
            logger.info(f"Network '{name}' has been successfully created in both the database and the L2SM controller.")
        else:
            # Roll back the database transaction if the network creation in the controller fails
            connection.rollback()
            logger.error(f"Failed to create network '{name}' in the L2SM controller. Database transaction rolled back.")
            
    except Exception as e:
        # Roll back the database transaction in case of any error
        connection.rollback()
        logger.error(f"An error occurred: {e}. Rolled back the database transaction.")
    finally:
        # Ensure the database connection is closed
        connection.close()



#ASSIGN POD TO NETWORK (TRIGGERS ONLY IF ANNOTATION IS PRESENT)
@kopf.on.create('pods.v1', annotations={'k8s.v1.cni.cncf.io/networks': kopf.PRESENT})
def pod_vn(body, name, namespace, logger, annotations, **kwargs):
    """Assign Pod to a network if a specific annotation is present."""
    

    # Avoid database overlap by introducing a random sleep time
    time.sleep(random.uniform(0, 0.8))

    multus_networks = extract_multus_networks(annotations)
    if not multus_networks:
        logger.info("No Multus networks specified. Exiting.")
        return

    existing_networks = get_existing_networks(namespace)
    target_networks = filter_target_networks(multus_networks, existing_networks)
    if not target_networks:
        logger.info("No target networks found. Letting Multus handle the network assignment.")
        return
    if 'spec' in body and 'nodeName' in body['spec']:
        node_name = body['spec']['nodeName']
           
        free_interfaces = get_free_interfaces(node_name)
        if len(free_interfaces) < len(target_networks):
            raise kopf.PermanentError(f"Node {node_name} has no free interfaces left")
        
        openflow_id = get_openflow_id(node_name)

        update_network_assignments(name, namespace, node_name, free_interfaces, target_networks, logger, openflow_id)
    else:
        raise kopf.TemporaryError("The Pod is not yet ready", delay=1)
   


def extract_multus_networks(annotations):
  """Extract and return Multus networks from annotations."""
  return [network.strip() for network in annotations.get('k8s.v1.cni.cncf.io/networks').split(",")]

def get_existing_networks(namespace):
    """Return existing networks in the namespace."""
    api = client.CustomObjectsApi()
    networks = api.list_namespaced_custom_object('k8s.cni.cncf.io', 'v1', namespace, 'network-attachment-definitions').get('items')
    return [network['metadata']['name'] for network in networks if '"type": "l2sm"' in network['spec']['config']]

def filter_target_networks(multus_networks, existing_networks):
    """Filter and return networks that are both requested and exist."""
    return [network for network in multus_networks if network in existing_networks]

def get_free_interfaces(node_name):
    """Query the database for free interfaces on a node."""
    connection = pymysql.connect(host=database_ip,
                                user=database_username,
                                password=database_password,
                                database=database_name,
                                cursorclass=pymysql.cursors.DictCursor)
    try:
        with connection.cursor() as cursor:
            sql = """
                    SELECT i.id, i.name
                    FROM interfaces i
                    JOIN switches s ON i.switch_id = s.id
                    WHERE i.network_id IS NULL AND s.node_name = %s;
                    """            
            cursor.execute(sql, (node_name.strip(),))
            free_interfaces = cursor.fetchall()
    finally:
        connection.close()
    return free_interfaces

def update_pod_annotation(pod_name, namespace, interfaces):
    """Update the Pod's annotation with assigned interfaces."""
    v1 = client.CoreV1Api()
    pod = v1.read_namespaced_pod(pod_name, namespace)
    pod_annotations = pod.metadata.annotations or {}
    pod_annotations['k8s.v1.cni.cncf.io/networks'] = ', '.join(interfaces)
    v1.patch_namespaced_pod(pod_name, namespace, {'metadata': {'annotations': pod_annotations}})

def update_network_assignments(pod_name, namespace, node_name, free_interfaces, target_networks, logger, openflow_id):
    """Update the network assignments in the database and controller."""
    connection = pymysql.connect(host=database_ip,
                                user=database_username,
                                password=database_password,
                                database=database_name,
                                cursorclass=pymysql.cursors.DictCursor)
    try:
        assigned_interfaces = []
        with connection.cursor() as cursor:
            for i, interface in enumerate(free_interfaces[:len(target_networks)]):
                update_interface_assignment(cursor, interface['id'], target_networks[i], pod_name, node_name)
                assigned_interfaces.append(interface['name'])

                # Assuming function get_openflow_id and session.post logic are implemented elsewhere
                if openflow_id:
                    port_number = extract_port_number(interface['name'])
                    post_network_assignment(openflow_id, port_number, target_networks[i])
                            
            update_pod_annotation(pod_name, namespace, assigned_interfaces)

        connection.commit()
    finally:
        connection.close()
    logger.info(f"Pod {pod_name} attached to networks {', '.join(target_networks)}")

# Assuming these functions are implemented as per original logic
def update_interface_assignment(cursor, interface_id, network_name, pod_name, node_name):
    """Update a single interface's network assignment in the database."""
    # First, find the network_id from the network name
    cursor.execute("SELECT id FROM networks WHERE name = %s", (network_name,))
    network = cursor.fetchone()
    if network is None:
        raise ValueError(f"Network {network_name} does not exist")
    network_id = network['id']

    # Update the interface with the network_id and pod name
    sql = """
    UPDATE interfaces
    SET pod = %s, network_id = %s
    WHERE id = %s;
    """
    cursor.execute(sql, (pod_name, network_id, interface_id))

def extract_port_number(interface_name):
    """Extract and return the port number from an interface name."""
    return int(re.search(r'\d+$', interface_name).group())

def post_network_assignment(openflow_id, port_number, network_name):
    """Post network assignment to the controller."""
    payload = {
        "networkId": network_name,
        "networkEndpoints": [f"{openflow_id}/{port_number}"]
    }
    response = session.post(f"{base_controller_url}/l2sm/networks/port", json=payload)
    if response.status_code != 204:
        raise Exception(f"Network assignment failed with status code: {response.status_code}")
     


#UPDATE DATABASE WHEN POD IS DELETED
@kopf.on.delete('pods.v1', annotations={'k8s.v1.cni.cncf.io/networks': kopf.PRESENT})
def dpod_vn(name, logger, **kwargs):
    connection = pymysql.connect(host=database_ip,
                                user=database_username,
                                password=database_password,
                                database=database_name,
                                cursorclass=pymysql.cursors.DictCursor)
    try:
        with connection.cursor() as cursor:
            sql = "UPDATE interfaces SET network_id = NULL, pod = NULL WHERE pod = '%s'" % (name)
            cursor.execute(sql)
            connection.commit()
    finally:
        connection.close()
        logger.info(f"Pod {name} removed")

#UPDATE DATABASE WHEN NETWORK IS DELETED
@kopf.on.delete('NetworkAttachmentDefinition', when=lambda spec, **_: '"type": "l2sm"' in spec['config'])
def delete_vn(spec, name, logger, **kwargs):
    connection = pymysql.connect(host=database_ip,
                                user=database_username,
                                password=database_password,
                                database=database_name,
                                cursorclass=pymysql.cursors.DictCursor)
    try:
        with connection.cursor() as cursor:
             # First, set network_id to NULL in interfaces for the network being deleted
            update_interfaces_sql = """
            UPDATE interfaces
            SET network_id = NULL
            WHERE network_id = (SELECT id FROM networks WHERE name = %s AND type = 'vnet');
            """
            cursor.execute(update_interfaces_sql, (name,))

            # Then, delete the network from networks table
            delete_network_sql = "DELETE FROM networks WHERE name = %s AND type = 'vnet';"
            cursor.execute(delete_network_sql, (name,))
            
   
    
            response = session.delete(base_controller_url + '/l2sm/networks/' + name)
            
            if response.status_code == 204:
                # Successful request
                logger.info(f"Network has been deleted in the SDN Controller")
                connection.commit()
            else:
                # Handle errors
                logger.info(f"Error: {response.status_code}")
    finally:
        connection.close()
        logger.info(f"Network {name} removed")

#DELETE DATABASE ENTRIES WHEN A NEW L2SM SWITCH IS DELETED (A NEW NODE GETS OUT OF THE CLUSTER)
@kopf.on.delete('pods.v1', labels={'l2sm-component': 'l2sm-switch'})
def remove_node(body, logger, annotations, **kwargs):
    connection = pymysql.connect(host=database_ip,
                                user=database_username,
                                password=database_password,
                                database=database_name,
                                cursorclass=pymysql.cursors.DictCursor)
    try:
        with connection.cursor() as cursor:
            sql = """
            DELETE FROM interfaces
            WHERE switch_id = (SELECT id FROM switches WHERE node_name = '%s');
            """
            switchSql = "DELETE FROM switches WHERE node_name = '%s';"
            cursor.execute(sql,body['spec']['nodeName'])
            cursor.execute(switchSql,body['spec']['nodeName'])
            connection.commit()
    finally:
        connection.close()
    logger.info(f"Node {body['spec']['nodeName']} has been deleted from the cluster")

