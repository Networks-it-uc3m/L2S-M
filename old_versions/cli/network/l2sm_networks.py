import os
import sys
import json
import subprocess
import secrets
from subprocess import CalledProcessError
from random import randrange
from prettytable import from_db_cursor, PrettyTable
from kubernetes import client, config

import pymysql

db = pymysql.connect(host="localhost",user="l2sm",password="l2sm;",db="L2SM")

cur = db.cursor()

def main(argv):
  #CREATE OPTION
  if argv[1] == 'create':
    if os.path.isfile(argv[2]):
      if os.path.splitext(argv[2])[-1].lower() == '.yaml':
        try:
          #WITHOUT KUBECONFIG
          if len(argv) == 3:
            create_network(argv[2], '')
          #WITH KUBECONFIG
          elif len(argv) == 4:
            create_network(argv[2], argv[3])
          else:
            print("Something went wrong: introduce the proper arguments")
        except CalledProcessError:
          db.close()
          subprocess.run(["rm", "tmp.yaml"], stdout=subprocess.DEVNULL)
        except Exception as e:
          print(e)
          subprocess.run(["rm", "tmp.yaml"], stdout=subprocess.DEVNULL)

      else:
        print("Wrong file format: Currently l2sm only supports yaml files")
    else:
      print("Descriptor is not a file: Please introduce a valid descriptor")

  #SHOW OPTION
  elif argv[1] == 'show':
#    show_networks(argv[2])
    if len(argv) == 3:
      show_networks(argv[2], '')
    elif len(argv) == 4:
      show_networks(argv[2], argv[3])

  elif argv[1] == 'delete':
    try:
      #DELETE OPTION 
      if len(argv) == 3:
        delete_network(argv[2], '')
      elif len(argv) == 4:
        delete_network(argv[2], argv[3])
      else:
        print("Something went wrong: introduce the proper arguments")
    except CalledProcessError:
      db.close()
    except Exception as e:
      print(e)


#BUILD A NETWORK FROM A K8S DESCRIPTOR
def create_network(onefile, kubeConfig):
  #GET NETWORK NAME
  with open(onefile, 'r') as file:
    with open('tmp.yaml', 'w') as newFile:
      for line in file:
        newFile.write(line)
        # GET THE NETWORK NAME
        if 'spec:' in line:
          search = 1
          while search:
            newline = next(file)
            newFile.write(newline)
            if 'name:' in newline:
              network = newline.split(":", 1)[1]
              search = 0
        # GET KUBERNETTES VN RESOURCE NAME 
        if 'metadata:' in line:
          search = 1
          while search:
            newline = next(file)
            newFile.write(newline)
            if 'name:' in newline:
              metadata = newline.split(":", 1)[1]
              search = 0

      # GENERATE RANDOM ID NUMBER
      #id = randrange(999999999)
      id = secrets.token_hex(32)
      newFile.write('\n  id: ' + id)

  #CHECK IF NETWORK EXISTS IN DATABASE
  nsql = "SELECT * FROM networks WHERE network = '%s'" % (network.strip())
  cur.execute(nsql)
  data=cur.fetchall()
  if data:
    print("Network " + network.strip() + " exists")
    subprocess.run(["rm", "tmp.yaml"], stdout=subprocess.DEVNULL)
    return

  #CREATE NETWORK IN MASTER (IF REQUIRED, USE KUBECONFIG FILE)
  if kubeConfig == '':
    subprocess.run(["kubectl", "create", "-f" + "tmp.yaml"], stdout=subprocess.DEVNULL).check_returncode()
  else:
    subprocess.run(["kubectl", "create", "--kubeconfig", kubeConfig, "-f" + "tmp.yaml"], stdout=subprocess.DEVNULL).check_returncode()

  subprocess.run(["rm", "tmp.yaml"], stdout=subprocess.DEVNULL)

  #CREATE IN THE DATABASE THE NEW VALUES
  sql = "INSERT INTO networks (network, id, metadata) VALUES ('%s', '%s', '%s')" % (network.strip(), id.strip(), metadata.strip())
  cur.execute(sql)
  db.commit()
  db.close()
  print("Network " + network.strip() + " has been created")

def show_networks(network, kubeConfig):
  table = ''
  output = ''
  #SHOW ALL NETWORKS
  if network == 'all':
    sql = "SELECT id FROM networks"
  #SHOW ONLY THE SELECTED NETWORK
  else:
    sql = "SELECT id FROM networks WHERE network = '%s'" % (network)
  cur.execute(sql)
  data = cur.fetchall()

  if not data:
    if network ==  'all':
      print("No networks have been defined in the cluster")
    else:
      print("Network " + network + " has not been created in the cluster")
    return

  #PRINT THE NETWORK (TO DO: SHOW POD ID'S)
  for id in data:
    sqln = "SELECT * FROM networks WHERE id = '%s'" % (id[0])
    cur.execute(sqln)
    output = from_db_cursor(cur).get_string(fields=['network', 'id'])
    print(output)
    sql2 = "SELECT * FROM interfaces WHERE network = '%s'" % (id[0])
    cur.execute(sql2)
    table_data = cur.fetchall()
    output = PrettyTable()
    output.field_names = ["network", "pod", "pod id"]
    if kubeConfig == '':
      config.load_kube_config()
    else:
      config.load_kube_config(kubeConfig)
    v1 = client.CoreV1Api()
    ret = v1.list_pod_for_all_namespaces(watch=False)

    for entry in table_data:
      for i in ret.items:
        if i.metadata.name == entry[3]:
          id = i.metadata.uid
      output.add_row([entry[2], entry[3], id])
    print(output)

def delete_network(network, kubeConfig):
  #CHECK IF NETWORK TO BE DELETED IS PRESENT
  sql = "SELECT id FROM networks WHERE network = '%s'" % (network)
  cur.execute(sql)
  data = cur.fetchall()
  if not data:
    print("Network " + network + " has not been created")
    return
  #RETRIEVE THE METADATA NAME OF THE NETWORK TO BE DELETED (SO IT CAN BE REMOVED FROM K8S)
  for id in data:
    sqln = "SELECT * FROM interfaces WHERE network = '%s'" % (id[0])
    cur.execute(sqln)
    pods = cur.fetchall()
    mqln = "SELECT metadata FROM networks WHERE id = '%s'" % (id[0])
    cur.execute(mqln)
    metadata = cur.fetchone()
    #DELETE THE VN RESOURCE FROM K8S
    if not pods:
      if kubeConfig == '':
        subprocess.run(["kubectl", "delete", " vn/ " + metadata[0]], stdout=subprocess.DEVNULL).check_returncode()
      else:
        subprocess.run(["kubectl", "delete", "--kubeconfig", kubeConfig, "vn/" + metadata[0]], stdout=subprocess.DEVNULL).check_returncode()
      #REMOVE ENTRY FROM THE DATABASE
      delsql = "DELETE FROM networks WHERE id = '%s'" % (id[0])
      cur.execute(delsql)
      db.commit()
      db.close()
      print("Network " + network.strip() + " has been deleted")

    else:
      print("Network " + network.strip() + " could not be removed, since there are pods still attached")
      return


if __name__== "__main__":
    main(sys.argv)
