#!/bin/bash

# Define the SSH command and credentials
SSH_HOST="localhost"
SSH_PORT="8101"
SSH_USER="karaf"
SSH_PASS="karaf"
COMMAND1="bundle:install -s mvn:org.yaml/snakeyaml/1.26"
COMMAND2="bundle:install -s mvn:com.fasterxml.jackson.dataformat/jackson-dataformat-yaml/2.11.0"

# Install sshpass if it is not already installed
if ! command -v sshpass &> /dev/null
then
    echo "sshpass could not be found, trying to install it..."
    sudo apt-get install sshpass -y
fi

# Execute the commands over SSH
sshpass -p $SSH_PASS ssh -p $SSH_PORT -o StrictHostKeyChecking=no $SSH_USER@$SSH_HOST "$COMMAND1; $COMMAND2"
