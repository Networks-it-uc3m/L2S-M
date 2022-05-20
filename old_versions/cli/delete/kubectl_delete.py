import subprocess
from subprocess import CalledProcessError
import sys
import json
import os
from kubernetes import client, config

import pymysql

#CONNECT TO THE DATABASE
db = pymysql.connect(host="localhost",user="l2sm",password="l2sm;",db="L2SM")

cur = db.cursor()

def main(argv):

  #CHECK IF KUBECONFIG IS PRESENT 
  if len(sys.argv) > 2:
    pod = argv[1]
    kubeConfig = argv[2]

  else:
    pod = argv[1]
    kubeConfig = ''

  try:
    delete_pod(pod, kubeConfig)
  except CalledProcessError:
    db.close()
  except Exception as e:
    print(e)


def delete_pod(pod, kubeConfig):

  #QUERY TO SEARCH FOR THEN POD TO BE DELETED
  sql = "SELECT pod FROM interfaces WHERE pod = '%s'" % (pod)
  cur.execute(sql)
  data = cur.fetchone()

  if not data:
    raise NameError("l2sm could not remove the pod: Pod " + pod.strip() + " does not exist")

  else:
    if kubeConfig == '':
      #USE K8S API TO RETRIEVE THE NODE WHERE THE POD HAS BEEN DEPLOYED
      config.load_kube_config()
      v1 = client.CoreV1Api()
      ret = v1.list_pod_for_all_namespaces(watch=False)
      for i in ret.items:
        if i.metadata.name == pod:
          node = i.spec.node_name

      subprocess.run(["kubectl", "delete", "pods/" + pod], stdout=subprocess.DEVNULL).check_returncode()

    else:
      config.load_kube_config(kubeConfig)
      v1 = client.CoreV1Api()
      ret = v1.list_pod_for_all_namespaces(watch=False)
      for i in ret.items:
        if i.metadata.name == pod:
          node = i.spec.node_name

      #DELETE THE POD USING THE KUBECTL TOOL
      subprocess.run(["kubectl", "--kubeconfig", kubeConfig, "delete", "pods/" + pod], stdout=subprocess.DEVNULL).check_returncode()

  #UPDATE THE DATABASE TO REMOVE THE POD
  dsql = "UPDATE interfaces SET network = '-1', pod = '' WHERE pod = '%s'" % (pod)
  cur.execute(dsql)
  db.commit()
  print("Pod " + pod.strip() + " has been deleted")


if __name__== "__main__":
    main(sys.argv)
    db.close()

