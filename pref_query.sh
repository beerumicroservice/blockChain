#!/bin/bash
. setpeer.sh Airtel peer0 
export CHANNEL_NAME="preferencechannel"
peer chaincode invoke -o orderer.ucc.net:7050 --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA -C $CHANNEL_NAME -n pref -c '{"Args":["qp","{\"selector\":{\"obj\":\"Preferences\",\"msisdn\":\"9199528280\"}, \"use_index\" :\"preferencesSearchBymsisdn\"}"]}'

