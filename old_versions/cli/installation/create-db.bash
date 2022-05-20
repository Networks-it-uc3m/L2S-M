#!/bin/bash

echo MySQL user:
read user
echo MySQL pass:
read -s pass

mysql --user="$user" --password="$pass" --execute="CREATE DATABASE L2SM;" &> /dev/null
mysql --user="$user" --password="$pass" --database="L2SM" --execute="CREATE TABLE networks (network TEXT NOT NULL, id INT, metadata TEXT NOT NULL);" &> /dev/null
mysql --user="$user" --password="$pass" --database="L2SM" --execute="CREATE TABLE interfaces (interface TEXT NOT NULL, node TEXT NOT NULL, network INT, pod TEXT);" &> /dev/null

echo 'Introduce the node names of the cluster separated with ;'

read IN

IFS=';' read -ra NODES <<< "$IN"


for i in "${NODES[@]}"; do
for j in {1..10}; do
	mysql --user="$user" --password="$pass" --database="L2SM" --execute="INSERT INTO interfaces VALUES ('"vpod$j"', '"$i"', '-1', '');" &> /dev/null
done
done


