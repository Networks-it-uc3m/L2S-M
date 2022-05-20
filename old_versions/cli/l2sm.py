import os
import sys
import json
import subprocess
from subprocess import CalledProcessError
from random import randrange
from prettytable import from_db_cursor

import argparse
import pymysql


#network function
def network_man(args):
  arguments = ["python3", "./network/l2sm_networks.py"]
  arguments.append(args.action)
  arguments.append(args.argument)
  if args.kubeconfig != '':
    arguments.append(args.kubeconfig)
#  print(arguments)
  subprocess.run(arguments)

#deploy function
def deploy_man(args):
  arguments = ["python3", "./deploy/kubectl_create.py"]
  arguments.append(args.file)
  if args.kubeconfig != '':
    arguments.append(args.kubeconfig)
#  print(arguments)
  subprocess.run(arguments)

#delete function
def delete_man(args):
  arguments = ["python3", "./delete/kubectl_delete.py"]
  arguments.append(args.pod)
  if args.kubeconfig != '':
    arguments.append(args.kubeconfig)
#  print(arguments)
  subprocess.run(arguments)

#execute kubectl function NOTE: always use '' before the options or other functions may not work
def exec_man(args):
  arguments = ["kubectl"]
  # Variable to introduce the new arguments
  var = list()
  # Steparate the string into a new list
  for line in args.options:
    line = line.strip()
    var.extend(line.split())
  # These operations should only be performed using L2S-M
  if "create" in var or "apply" in var or "delete" in var:
    print("Operation not permitted: Creating and deleting resources are limited to l2sm operations")
    return
  arguments.extend(var)
  if args.kubeconfig != '':
    arguments.append("--kubeconfig")
    arguments.append(args.kubeconfig)
  subprocess.run(arguments)




def main(argv):
  #Create Parser 
  parser = argparse.ArgumentParser()
#  parser.add_argument("option", help= "Option to be used by l2sm", choices=['deploy', 'delete', 'exec'])
  subparsers = parser.add_subparsers(help="Special options for l2sm")

  #Add network parser
  parser_networks = subparsers.add_parser('network', help='Operate with network options')
  parser_networks.add_argument("action", choices=['create', 'show', 'delete'], help = "Network management action")
  parser_networks.add_argument("argument", help = "File or network to operate with")
  parser_networks.add_argument("--kubeconfig", help="If needed, use kubeConfig file to contact the Kubernetes cluster", action="store", default='')
  # define the function to use with network
  parser_networks.set_defaults(func=network_man)

  #Add deploy parser
  parser_networks = subparsers.add_parser('deploy', help='Deploy a pod to be connected to l2sm networks')
  parser_networks.add_argument("file", help = "File to be deployed")
  parser_networks.add_argument("--kubeconfig", help="If needed, use kubeConfig file to contact the Kubernetes cluster", action="store", default='')
  # define the function to use with network
  parser_networks.set_defaults(func=deploy_man)

  #Add delete parser
  parser_networks = subparsers.add_parser('delete', help='Delete a pod connected to l2sm networks')
  parser_networks.add_argument("pod", help = "Pod to be deleted")
  parser_networks.add_argument("--kubeconfig", help="If needed, use kubeConfig file to contact the Kubernetes cluster", action="store", default='')
  # define the function to use with network
  parser_networks.set_defaults(func=delete_man)

  #Add exec parser
  parser_exec = subparsers.add_parser('exec', help='Operate with kubectl')
  parser_exec.add_argument("options", help = "Use the same notation as it would be used in kubectl", nargs='*')
  parser_exec.add_argument("--kubeconfig", help="If needed, use kubeConfig file to contact the Kubernetes cluster", action="store", default='')
  # define the function to use with exec
  parser_exec.set_defaults(func=exec_man)


  # Get arguments
  args = parser.parse_args()
 # print(args)

  args.func(args)


if __name__== "__main__":
    main(sys.argv)

