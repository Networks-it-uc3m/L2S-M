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

ip = "127.0.0.1"

#POPULATE DATABASE ENTRIES WHEN A NEW L2SM POD IS CREATED (A NEW NODE APPEARS)
@kopf.on.create('pods.v1', labels={'l2sm-component': 'l2-ps'})
def build_db(body, logger, annotations, **kwargs):
    db = pymysql.connect(host=ip,user="l2sm",password="l2sm;",db="L2SM")
    cur = db.cursor()
    #CREATE TABLES IF THEY DO NOT EXIST
    table1 = "CREATE TABLE IF NOT EXISTS networks (network TEXT NOT NULL, id TEXT NOT NULL);"
    table2 = "CREATE TABLE IF NOT EXISTS interfaces (interface TEXT NOT NULL, node TEXT NOT NULL, network TEXT, pod TEXT);"
    cur.execute(table1)
    cur.execute(table2)
    db.commit()
    values = []
    #MODIFY THE END VALUE TO ADD MORE INTERFACES
    for i in range(1,5):
      values.append(['vpod'+str(i), body['spec']['nodeName'], '-1', ''])
    sql = "INSERT INTO interfaces (interface, node, network, pod) VALUES (%s, %s, %s, %s)"
    cur.executemany(sql, values)
    db.commit()
    db.close()
    logger.info(f"Node {body['spec']['nodeName']} has been registered in the operator")

#UPDATE DATABASE WHEN NETWORK IS CREATED
@kopf.on.create('virtual-networks')
def create_vn(spec, name, namespace, logger, **kwargs):
    db = pymysql.connect(host=ip,user="l2sm",password="l2sm;",db="L2SM")
    cur = db.cursor()
    id = secrets.token_hex(32)
    sql = "INSERT INTO networks (network, id) VALUES ('%s', '%s')" % (name.strip(), id.strip())
    cur.execute(sql)
    db.commit()
    db.close()
    logger.info(f"Network has been created")


#ASSIGN POD TO NETWORK (TRIGGERS ONLY IF ANNOTATION IS PRESENT)
@kopf.on.create('pods.v1', annotations={'l2sm.k8s.conf.io/virtual-networks': kopf.PRESENT})
def pod_vn(body, name, namespace, logger, annotations, **kwargs):
    #GET NETWORK IN THE DESCRIPTOR
    #IN QUARANTINE: SLOWER THAN MULTUS!!!!!
    time.sleep(random.uniform(0,0.8)) #Make sure the database is not consulted at the same time to avoid overlaping

    network = annotations.get('l2sm.k8s.conf.io/virtual-networks').split(",")
    #VERIFY IF NETWORK IS PRESENT IN THE CLUSTER
    api = client.CustomObjectsApi()
    items = api.list_namespaced_custom_object('l2sm.k8s.conf.io', 'v1', namespace, 'virtual-networks').get('items')
    resources = []
    for i in items:
      resources.append(i['metadata']['name'])

    for k in range(len(network)):
      network[k] = network[k].strip()
      if network[k] not in resources:
        raise kopf.PermanentError("The pod could not be attached the network since network " + network[k] + " was not defined in the cluster")


    #CHECK IF NODE HAS FREE VIRTUAL INTERFACES LEFT
    v1 = client.CoreV1Api()
    ret = v1.read_namespaced_pod(name, namespace)
    node = body['spec']['nodeName']

    db = pymysql.connect(host=ip,user="l2sm",password="l2sm;",db="L2SM")
    nsql = "SELECT * FROM interfaces WHERE node = '%s' AND network = '-1'" % (node.strip())
    cur = db.cursor()
    cur.execute(nsql)
    data = cur.fetchall()
    if not data or len(data)<len(network):
      db.close()
      raise kopf.PermanentError("l2sm could not deploy the pod: Node " + node.strip() + "has no free interfaces left")

    #IF THERE IS ALREADY A MULTUS ANNOTATION, APPEND IT TO THE END.
    interface_to_attach = []
    for interface in data[0:len(network)]:
      interface_to_attach.append(interface[0].strip())
      if 'k8s.v1.cni.cncf.io/networks' not in ret.metadata.annotations:
        ret.metadata.annotations['k8s.v1.cni.cncf.io/networks'] = interface[0].strip()
      else:
        ret.metadata.annotations['k8s.v1.cni.cncf.io/networks'] = ret.metadata.annotations['k8s.v1.cni.cncf.io/networks'] + ", " + interface[0].strip()

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

    for m in range(len(network)):
      sql = "UPDATE interfaces SET network = '%s', pod = '%s' WHERE interface = '%s' AND node = '%s'" % (network[m], name, interface_to_attach[m], node)
      cur.execute(sql)

    db.commit()
    db.close()
    #HERE GOES SDN, THIS IS WHERE THE FUN BEGINS
    logger.info(f"Pod {name} attached to network {network}")


#UPDATE DATABASE WHEN POD IS DELETED
@kopf.on.delete('pods.v1', annotations={'l2sm.k8s.conf.io/virtual-networks': kopf.PRESENT})
def dpod_vn(name, logger, **kwargs):
    db = pymysql.connect(host=ip,user="l2sm",password="l2sm;",db="L2SM")
    cur = db.cursor()
    sql = "UPDATE interfaces SET network = '-1', pod = '' WHERE pod = '%s'" % (name)
    cur.execute(sql)
    db.commit()
    db.close()
    logger.info(f"Pod {name} removed")

#UPDATE DATABASE WHEN NETWORK IS DELETED
@kopf.on.delete('virtual-networks')
def delete_vn(spec, name, logger, **kwargs):
    db = pymysql.connect(host=ip,user="l2sm",password="l2sm;",db="L2SM")
    cur = db.cursor()
    sql = "DELETE FROM networks WHERE network = '%s'" % (name)
    cur.execute(sql)
    db.commit()
    db.close()
    logger.info(f"Network has been deleted")

#DELETE DATABASE ENTRIES WHEN A NEW L2SM POD IS DELETED (A NEW NODE GETS OUT OF THE CLUSTER)
@kopf.on.delete('pods.v1', labels={'l2sm-component': 'l2-ps'})
def remove_node(body, logger, annotations, **kwargs):
    db = pymysql.connect(host=ip,user="l2sm",password="l2sm;",db="L2SM")
    cur = db.cursor()
    sql = "DELETE FROM interfaces WHERE node = '%s'" % (body['spec']['nodeName'])
    cur.execute(sql)
    db.commit()
    db.close()
    logger.info(f"Node {body['spec']['nodeName']} has been deleted from the cluster")

