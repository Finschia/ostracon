#!/bin/bash

IFS_BACK=$IFS
IFS=$'\r\n'

cd `dirname $0`

# updator validator with default parameter
commands=`go run make_val_set_change_tx.go --staking=10 --priv-key=${HOME}/.ostracon/config/priv_validator_key.json`
# remove validator tx
commands=`go run make_val_set_change_tx.go --staking=0`
# update validator tx
commands=`go run make_val_set_change_tx.go`
for command in ${commands[@]}; do
	if [[ "$command" =~ \# ]]; then
		echo $command
	else
		echo $command
		eval $command
		RET=$?
		echo ""
		if [ ${RET} -ne 0 ]; then
			echo "ERROR: Result Code of calling RPC: ${RET}"
			exit ${RET}
		fi
	fi
done

IFS=$IFS_BACK
