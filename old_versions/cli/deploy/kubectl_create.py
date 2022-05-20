import subprocess
from subprocess import CalledProcessError
import sys
import json
import os

import pymysql

db = pymysql.connect(host="localhost",user="l2sm",password="l2sm;",db="L2SM")

cur = db.cursor()

def main(argv):

  #USE TEMPORARY FILE AS DESCRIPTOR TO BE USED
  newDescriptor = 'tmp-descriptor.yaml'

  #USE KUBECONFIG IF PRESENT
  if len(sys.argv) > 1:
    nativeDescriptor = argv[1]
    kubeConfig = argv[2]
  else:
    nativeDescriptor = argv
    kubeConfig = ''

  #CONTINUE IF VALID .YAML FORMAT
  if os.path.isfile(nativeDescriptor):
    if os.path.splitext(nativeDescriptor)[-1].lower() == '.yaml':

      #CHECK IF THE NODE WHERE THE POD WILL BE DEPLOYED IS VALID AND GET THE NAME
      try:
        node = get_node(nativeDescriptor)

      except Exception as e:
        print(e)
        db.close()
        quit()

      #CHECK IF THE NODE HAS FREE INTERFACES LEFT TO USE IN THE DATABASE
      nsql = "SELECT * FROM interfaces WHERE node = '%s' AND network = '-1'" % (node.strip())
      cur.execute(nsql)
      data = cur.fetchone()

      #TO DO: instanciar un nuevo NED -> mejor hacerlo al final si no quedan interfaces o ahora?
      if not data:
        print("l2sm could not deploy the pod: Node " + node.strip() + "has no free interfaces left")
        db.close()
        quit()

      interface_to_attach = data[0]

      try:
        #PROCCESS THE DESCRIPTOR TO SEE INCOMPATIBILITIES OR NULL VALUES
        newDescriptor = prepare_yaml(nativeDescriptor)
        #CREATE NEW DESCRIPTOR
        id, pod = process_file(newDescriptor, interface_to_attach)
        if kubeConfig == '':
          #subprocess.run(["kubectl", "get", "vn/" + network.strip()], stdout=subprocess.DEVNULL).check_returncode()
          subprocess.run(["kubectl", "create", "-f", newDescriptor], stdout=subprocess.DEVNULL).check_returncode()
        else:
          #subprocess.run(["kubectl", "--kubeconfig", kubeConfig, "get", "vn/" + network.strip()], stdout=subprocess.DEVNULL).check_returncode()
          subprocess.run(["kubectl", "--kubeconfig", kubeConfig, "create", "-f", newDescriptor], stdout=subprocess.DEVNULL).check_returncode()

        print('Scheduling pod ' + pod.strip() + ' at node ' + node.strip())

      except CalledProcessError:
        db.close()
        subprocess.run(["rm", newDescriptor], stdout=subprocess.DEVNULL)
        quit()
      except Exception as e:
        print(e)
        db.close()
        subprocess.run(["rm", newDescriptor], stdout=subprocess.DEVNULL)
        quit()

            #UPDATE DB VALUES
      try:
        sql = "UPDATE interfaces SET network = '%s', pod = '%s' WHERE interface = '%s' AND node = '%s'" % (id.strip(), pod.strip(), data[0], node.strip())
        cur.execute(sql)
              #If más de 2 interfaces, haz magia de Borhacker
        db.commit()
        subprocess.run(["rm", newDescriptor], stdout=subprocess.DEVNULL)
      except:
        print("Could not perform db update")
        subprocess.run(["rm", newDescriptor], stdout=subprocess.DEVNULL)
        db.rollback()

    else:
      print("Wrong file format: Currently l2sm only supports yaml files")

  else:
    print("Descriptor is not a file: Please introduce a valid descriptor")

#FUNCTION FOR RETRIEVING THE NODE NAME
def get_node(descriptor):
  with open(descriptor, 'r') as nodefile:
    if 'nodeName' not in nodefile.read():
      raise NameError("l2sm could not deploy the pod: The field nodeName is not present in the descriptor")
    nodefile.seek(0)
    for line in nodefile:
      if 'nodeName' in line:
        node = line.split(":", 1)[1]
        if node.strip() == '':
          raise Exception('l2sm could not deploy the pod: Field nodeName cannot be an empty value')
        nodefile.close()
        return node

#PROCCESS THE DESCRIPTOR TO SEE INCOMPATIBILITIES OR NULL VALUES
def prepare_yaml(fname):
    newDescriptor = 'tmp-descriptor.yaml'
    with open(fname, 'r') as infile:
      with open(newDescriptor, 'w') as outfile:
        for line in infile:
          analyseLine = line.strip()
          #LOOK FOR COMMENTS AND REMOVE THEM -> ONLY IF SYMBOL IS THE FIRST ONE OF THE NEW LINE
          if '#' in analyseLine:
            if analyseLine.find('#') == 0:
              pass
            else:
              outfile.write(line)
          else:
            outfile.write(line)

   #IF THE KUBERNETES ANNOTATION IS NOT PRESENT, STOP
    with open(newDescriptor, 'r') as prepfile:
      if 'l2sm.k8s.conf.io/virtual-networks:' not in prepfile.read():
        raise NameError("l2sm could not deploy the pod: no virtual network was defined in the descriptor (to deploy pods not attached to l2sm, use exec options instead)")
    prepfile.close()

   #IF NODENAME IS EMPTY, STOP
    with open(newDescriptor, 'r') as prepfile:
      if 'nodeName' not in prepfile.read():
        raise NameError("l2sm could not deploy the pod: The field nodeName is not present in the descriptor")
    prepfile.close()

    return newDescriptor

#GENERATE THE DESCRIPTOR TO BE USED IN THE KUBECTL
def process_file(onefile, interface_to_attach):
    # CHANGE VIRTUAL NETWORK VALUE FOR INTERFACE
    lines = []
    network = ''
    pod = ''
    id = '-1'

    with open(onefile, 'r') as file:
      for line in file:
        if 'l2sm.k8s.conf.io/virtual-networks' in line:
          network = line.split(":", 1)[1]
          #CHECK IF NETWORK IN THE DESCRIPTOR EXISTS
          sql = "SELECT id FROM networks WHERE network = '%s'" % (network.strip())
          cur.execute(sql)
          data = cur.fetchone()
          if not data:
            raise Exception("l2sm could not deploy the pod: Virtual network " + network.strip() + " does not exist in the cluster")
            newDescriptor.close()
            return id, pod
          #REPLACE THE VIRTUAL NETWORK ANNOTATION WITH THE MULTUS ONE
          id = data[0]
          #print(id)
          interface_in_file = ' ' + interface_to_attach + '\n'
          line = line.replace('l2sm.k8s.conf.io/virtual-networks:' + network, 'k8s.v1.cni.cncf.io/networks:' + interface_in_file)
        lines.append(line)

      # GET POD NAME VALUE
        if 'metadata:' in line:
          try:
            search = 1
            while search:
              newline = next(file)
              lines.append(newline)
              if 'name:' in newline:
                pod = newline.split(":", 1)[1]
                search = 0
          except:
            raise Exception('l2sm could not deploy the pod: Pod name is not present in the descriptor metadata')
            onefile.close()
            return id, pod

          if pod.strip() == '':
            raise Exception('l2sm could not deploy the pod: Pod name cannot be an empty value')
            onefile.close()
            return id, pod

    # WRITE TMP FILE
    with open(onefile, 'w') as out:
      for line in lines:
        out.write(line)

    return id, pod


if __name__== "__main__":
    main(sys.argv)
    db.close()

