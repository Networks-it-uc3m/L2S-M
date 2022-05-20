#!/bin/bash
file=$1

while IFS=";" read -r vtep ip
do
	# Ignores lines starting with '#' and empty lines
	if [[ $vtep == \#* ]] ; then
		continue
	fi

	bridge fdb append to 00:00:00:00:00:00 dst $ip dev $vtep

done < $file
