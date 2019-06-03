#!/bin/bash
. setpeer.sh Airtel peer0 
export CHANNEL_NAME="preferencechannel"
peer chaincode invoke -o orderer.ucc.net:7050 --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA -C $CHANNEL_NAME -n pref -c '{"Args":["sp","{\"cmode\":\"10,11\",\"ctgr\":\"1,2,3,4,5\",\"cts\":\"1556083755\",\"day\":\"31,32\",\"lrn\":\"3333\",\"msisdn\":\"9199528288\",\"reqno\":\"1002155353448664489\",\"rmode\":\"2\",\"svcprv\":\"AI\",\"time\":\"21,22\",\"uts\":\"1556083755\"}"]}'





