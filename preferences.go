/*
Copyright Tanla Solutions Ltd. 2019 All Rights Reserved.
This Chaincode is written for storing,retrieving,updating,
deleting(churnout) the preferences that are stored in DLT
and portOut the MSISDN on Successful Certificate verification.
*/

package main

import (
	"bytes"
	"encoding/json" //reading and writing JSON
	"strconv"       //import for msisdn validation
	"strings"

	"vendor/github.com/hyperledger/fabric/core/chaincode/shim"         // import for Chaincode Interface
	"vendor/github.com/hyperledger/fabric/core/chaincode/shim/ext/cid" // import for Client Identity
	pb "vendor/github.com/hyperledger/fabric/protos/peer"              // import for peer response
)

//Logger for Logging
var logger = shim.NewLogger("BATCH-PREFERENCES")

//Event Names
const EVTADDPREFERENCES = "ADD-PREFERENCES"
const EVTUPDATEPREFERENCES = "UPDATE-PREFERENCES"
const EVTDELPREFERENCES = "DELETE-PREFERENCES"
const EVTPORTOUT = "PORT-OUT"

//Output Structure for the output response
type Output struct {
	Data         string `json:"data"`
	ErrorDetails string `json:"error"`
}

//Event Payload Structure
type Event struct {
	Data string `json:"data"`
	Txid string `json:"txid"`
}

//Smart Contract structure
type CPM struct {
}

//=========================================================================================================
// Preference structure, with 13 properties.  Structure tags are used by encoding/json library
//=========================================================================================================
type Preference struct {
	ObjType           string `json:"obj"`
	Phone             string `json:"msisdn"`
	ServiceProvider   string `json:"svcprv"`
	RequestNumber     string `json:"reqno"`
	RegistrationMode  string `json:"rmode"`
	Category          string `json:"ctgr"`
	CommunicationMode string `json:"cmode"`
	DayType           string `json:"day"`
	DayTimeBand       string `json:"time"`
	Lrn               string `json:"lrn"`
	UpdateTs          string `json:"uts"`
	CreateTs          string `json:"cts"`
	UpdatedBy         string `json:"uby"`
}

//=========================================================================================================
// Init Chaincode
// The Init method is called when the Smart Contract "Preferences" is instantiated by the blockchain network
//=========================================================================================================

func (c *CPM) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Info("###### Preferences-Chaincode is Initialized #######")
	return shim.Success(nil)
}

// ========================================
// Invoke - Entry point for Invocations
// ========================================
func (dlp *CPM) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	logger.Infof("Preferences ChainCode Invoked, Function Name: " + string(function))
	switch function {
	case "sp": // add or update preference
		return dlp.setPreferences(stub, args)
	case "abp": //add batch preferences
		return dlp.batchPreferences(stub, args)
	case "dp": //churn out the preferences from DL
		return dlp.delPreferences(stub, args)
	case "po": //ownership transfer from donor to acceptor
		return dlp.portOut(stub, args)
	case "qp": //Rich Query to retrieve the Preferences from DL
		return dlp.queryPreferences(stub, args)
	default:
		logger.Errorf("Unknown Function Invoked, Available Function argument shall be any one of : sp,abp,dp,po,qp")
		return shim.Error("Available Functions: sp,abp,DP,po,qp")
	}
}

//setPreferences - Setting new preference or updating existing preference
// ==============================================================================
func (dlp *CPM) setPreferences(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var data map[string]interface{}
	err := json.Unmarshal([]byte(args[0]), &data)

	logger.Infof("data %v", data)

	if err != nil {
		logger.Errorf("setPreferences : Input arguments unmarhsaling Error : " + string(err.Error()))
		return shim.Error("setPreferences : Input arguments unmarhsaling Error : " + string(err.Error()))
	}
	certData, err := cid.GetX509Certificate(stub)
	if err != nil {
		logger.Errorf("setPreferences : Getting certificate Details Error : " + string(err.Error()))
		return shim.Error("setPreferences : Getting certificate Details Error : " + string(err.Error()))
	}

	if _, err := strconv.Atoi(data["msisdn"].(string)); err != nil {
		jsonResp = "{\"Error\":\"MSISDN is not numeric \"}"
		return shim.Error(jsonResp)
	}
	if len(data["msisdn"].(string)) < 10 {
		jsonResp = "{\"Error\":\"MSISDN is not a valid length \"}"
		return shim.Error(jsonResp)
	}
	if _, err := strconv.Atoi(data["lrn"].(string)); err != nil {
		jsonResp = "{\"Error\":\"LRN is not numeric \"}"
		return shim.Error(jsonResp)
	}
	Organizations := certData.Issuer.Organization
	value, err := stub.GetState(data["msisdn"].(string))
	if err != nil {
		logger.Errorf("setPreferences : GetState Failed for MSISDN : " + data["msisdn"].(string) + " Error : " + string(err.Error()))
		return shim.Error("setPreferences : GetState Failed for MSISDN : " + data["msisdn"].(string) + " Error : " + string(err.Error()))
	}
	if value == nil {
		if len(data) == 11 {
			PrfStruct := &Preference{}
			PrfStruct.ObjType = "Preferences"
			PrfStruct.Phone = data["msisdn"].(string)
			PrfStruct.ServiceProvider = data["svcprv"].(string)
			PrfStruct.RequestNumber = data["reqno"].(string)
			PrfStruct.RegistrationMode = data["rmode"].(string)
			PrfStruct.Category = data["ctgr"].(string)
			PrfStruct.CommunicationMode = data["cmode"].(string)
			PrfStruct.DayType = data["day"].(string)
			PrfStruct.DayTimeBand = data["time"].(string)
			PrfStruct.Lrn = data["lrn"].(string)
			PrfStruct.UpdateTs = data["uts"].(string)
			PrfStruct.CreateTs = data["cts"].(string)
			PrfStruct.UpdatedBy = Organizations[0]
			logger.Infof("msisdn is " + PrfStruct.Phone)
			PrfAsBytes, err := json.Marshal(PrfStruct)
			if err != nil {
				logger.Errorf("setPreferences : Marshalling Error : " + string(err.Error()))
				return shim.Error("setPreferences : Marshalling Error : " + string(err.Error()))
			}
			//Inserting DataBlock to BlockChain
			err = stub.PutState(PrfStruct.Phone, PrfAsBytes)
			if err != nil {
				logger.Errorf("setPreferences : PutState Failed Error : " + string(err.Error()))
				return shim.Error("setPreferences : PutState Failed Error : " + string(err.Error()))
			}
			logger.Infof("setPreferences : PutState Success : " + string(PrfAsBytes))
			//Txid := stub.GetTxID()
			eventbytes := Event{Data: string(PrfAsBytes), Txid: stub.GetTxID()}
			payload, err := json.Marshal(eventbytes)
			if err != nil {
				logger.Errorf("setPreferences : Event Payload Marshalling Error : " + string(err.Error()))
				return shim.Error("setPreferences : Event Payload Marshalling Error : " + string(err.Error()))
			}
			err2 := stub.SetEvent(EVTADDPREFERENCES, []byte(payload))
			if err2 != nil {
				logger.Errorf("setPreferences : Event Creation Error for EventID : " + string(EVTADDPREFERENCES))
				return shim.Error("setPreferences : Event Creation Error for EventID : " + string(EVTADDPREFERENCES))
			}
			logger.Infof("setPreferences : Event Payload data : " + string(payload))
			txid := stub.GetTxID()
			return shim.Success([]byte("setPreferences : Preferences data added Successfully for MSISDN : " + PrfStruct.Phone + " , TransactionID      " + txid))
		} else {
			jsonResp = "{\"msisdn\":\"value\",\"svcprv\":\"value\",\"reqno\":\"value\",\"rmode\":\"value\",\"ctgr\":\"value\",\"cmode\":\"value\",\"day\":\"value\",\"time\":\"value\",\"lrn\":\"value\",\"uts\":\"value\",\"cts\":\"value\"}"
			logger.Errorf("setPreferences : Incorrect Number Of Arguments, Expected json structure : " + string(jsonResp))
			return shim.Error("setPreferences : Incorrect Number Of Arguments, Expected json structure : " + string(jsonResp))
		}
	} else {
		if len(data) == 11 {
			var organizationName string
			var orgName string
			preference := Preference{}
			err := json.Unmarshal(value, &preference)
			if err != nil {
				logger.Errorf("setPreferences : Existing prefernce data Unmarhsaling Error : " + string(err.Error()))
				return shim.Error("setPreferences : Existing prefernce data Unmarhsaling Error : " + string(err.Error()))
			}
			orgName = preference.UpdatedBy
			organizationName = Organizations[0]
			if strings.Compare(orgName, organizationName) == 0 {
				PrfStruct := &Preference{}
				PrfStruct.ObjType = "Preferences"
				PrfStruct.Phone = data["msisdn"].(string)
				PrfStruct.ServiceProvider = data["svcprv"].(string)
				PrfStruct.RequestNumber = data["reqno"].(string)
				PrfStruct.RegistrationMode = data["rmode"].(string)
				PrfStruct.Category = data["ctgr"].(string)
				PrfStruct.CommunicationMode = data["cmode"].(string)
				PrfStruct.DayType = data["day"].(string)
				PrfStruct.DayTimeBand = data["time"].(string)
				PrfStruct.Lrn = data["lrn"].(string)
				PrfStruct.UpdateTs = data["uts"].(string)
				PrfStruct.CreateTs = data["cts"].(string)
				PrfStruct.UpdatedBy = Organizations[0]
				logger.Infof("msisdn is " + PrfStruct.Phone)
				PrfAsBytes, err := json.Marshal(PrfStruct)
				if err != nil {
					logger.Errorf("setPreferences : Marshaling Error : " + string(err.Error()))
					return shim.Error("setPreferences : Marshaling Error : " + string(err.Error()))
				}
				//Inserting DataBlock to BlockChain
				err = stub.PutState(PrfStruct.Phone, PrfAsBytes)
				if err != nil {
					logger.Errorf("setPreferences : PutState Failed Error : " + string(err.Error()))
					return shim.Error("setPreferences : PutState Failed Error : " + string(err.Error()))
				}
				logger.Infof("setPreferences : PutState Success : " + string(PrfAsBytes))
				eventbytes := Event{Data: string(PrfAsBytes), Txid: stub.GetTxID()}
				payload, err := json.Marshal(eventbytes)
				if err != nil {
					logger.Errorf("setPreferences : Event Payload marshaling Error : " + string(err.Error()))
					return shim.Error("setPreferences : Event Payload marshaling Error : " + string(err.Error()))
				}
				err2 := stub.SetEvent(EVTUPDATEPREFERENCES, []byte(payload))
				if err2 != nil {
					logger.Errorf("setPreferences : Event Creation Error for EventID : " + string(EVTUPDATEPREFERENCES))
					return shim.Error("setPreferences : Event Creation Error for EventID : " + string(EVTUPDATEPREFERENCES))
				}
				logger.Infof("Event published data: " + string(payload))
				txid := stub.GetTxID()
				return shim.Success([]byte("setPreferences : Preference data updated  successfully for MSISDN : " + PrfStruct.Phone + " , TransactionID      " + txid))
			} else {
				logger.Errorf("Unauthorized Access")
				return shim.Error("Unauthorized Access")

			}
		} else {
			jsonResp = "{\"msisdn\":\"value\",\"svcprv\":\"value\",\"reqno\":\"value\",\"rmode\":\"value\",\"ctgr\":\"value\",\"cmode\":\"value\",\"day\":\"value\",\"time\":\"value\",\"lrn\":\"value\",\"uts\":\"value\",\"cts\":\"value\"}"
			logger.Errorf("setPreferences : Incorrect Number Of Arguments, Expected json structure : " + string(jsonResp))
			return shim.Error("setPreferences : Incorrect Number Of Arguments, Expected json structure : " + string(jsonResp))
		}
	}
}

//======================================================
//batchPreferences for Uploading Bulk Preferences into DL
//=======================================================

func (dlp *CPM) batchPreferences(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var result []string
	var errorCount int
	errorCount = 0
	for i := 0; i < len(args); i++ {
		var errorCheck int
		var data map[string]interface{}
		errorCheck = 0
		logger.Infof(args[i])
		err := json.Unmarshal([]byte(args[i]), &data)
		if err != nil {
			logger.Errorf("batchPreferences : Input arguments unmarhsaling Error : " + string(err.Error()))
			return shim.Error("batchPreferences : Input arguments unmarhsaling Error : " + string(err.Error()))
		}
		certData, err := cid.GetX509Certificate(stub)
		if err != nil {
			logger.Errorf("batchPreferences : Getting certificate Details Error : " + string(err.Error()))
			return shim.Error("batchPreferences : Getting certificate Details Error : " + string(err.Error()))
		}
		Organizations := certData.Issuer.Organization
		if _, err := strconv.Atoi(data["msisdn"].(string)); err != nil {
			out := Output{Data: data["msisdn"].(string), ErrorDetails: "MobileNumber Contains Only Numeric Characters,It is Not Numeric"}
			edata, err := json.Marshal(out)
			if err != nil {
				logger.Errorf("batchPreferences : Marshalling Error : " + string(err.Error()))
				return shim.Error("batchPreferences : Marshalling Error : " + string(err.Error()))
			}
			result = append(result, string(edata))
			errorCount = errorCount + 1
			errorCheck = errorCheck + 1
		}
		if len(data["msisdn"].(string)) < 10 {
			out := Output{Data: data["msisdn"].(string), ErrorDetails: "MobileNumber is not Valid Length"}
			edata, err := json.Marshal(out)
			if err != nil {
				logger.Errorf("batchPreferences : Marshalling Error : " + string(err.Error()))
				return shim.Error("batchPreferences : Marshalling Error : " + string(err.Error()))
			}
			result = append(result, string(edata))
			errorCount = errorCount + 1
			errorCheck = errorCheck + 1

		}
		if _, err := strconv.Atoi(data["lrn"].(string)); err != nil {
			out := Output{Data: data["lrn"].(string), ErrorDetails: "Lrn Containts Only Numeric Charcters,It is Not Numeric"}
			edata, err := json.Marshal(out)
			if err != nil {
				logger.Errorf("batchPreferences : Marshalling Error : " + string(err.Error()))
				return shim.Error("batchPreferences : Marshalling Error : " + string(err.Error()))
			}
			result = append(result, string(edata))
			errorCount = errorCount + 1
			errorCheck = errorCheck + 1

		}
		if errorCheck != 0 {
			continue
		}
		value, err := stub.GetState(data["msisdn"].(string))
		if err != nil {
			logger.Errorf("batchPreferences : GetState Failed for MSISDN : " + data["msisdn"].(string) + " , Error : " + string(err.Error()))
			return shim.Error("batchPreferences : GetState Failed for MSISDN : " + data["msisdn"].(string) + " , Error : " + string(err.Error()))
		}
		if value == nil {
			if len(data) == 11 {
				PrfStruct := &Preference{}
				PrfStruct.ObjType = "Preferences"
				PrfStruct.Phone = data["msisdn"].(string)
				PrfStruct.ServiceProvider = data["svcprv"].(string)
				PrfStruct.RequestNumber = data["reqno"].(string)
				PrfStruct.RegistrationMode = data["rmode"].(string)
				PrfStruct.Category = data["ctgr"].(string)
				PrfStruct.CommunicationMode = data["cmode"].(string)
				PrfStruct.DayType = data["day"].(string)
				PrfStruct.DayTimeBand = data["time"].(string)
				PrfStruct.Lrn = data["lrn"].(string)
				PrfStruct.UpdateTs = data["uts"].(string)
				PrfStruct.CreateTs = data["cts"].(string)
				PrfStruct.UpdatedBy = Organizations[0]
				logger.Infof("msisdn is " + PrfStruct.Phone)
				PrfAsBytes, err := json.Marshal(PrfStruct)
				if err != nil {
					logger.Errorf("batchPreferences : Marshalling Error : " + string(err.Error()))
					return shim.Error("batchPreferences : Marshalling Error : " + string(err.Error()))
				}
				//Inserting DataBlock to BlockChain
				err = stub.PutState(PrfStruct.Phone, PrfAsBytes)
				if err != nil {
					logger.Errorf("batchPreferences : PutState Failed Error : " + string(err.Error()))
					return shim.Error("batchPreferences : PutState Failed Error : " + string(err.Error()))
				}
				logger.Infof("batchPreferences : PutState Success : " + string(PrfAsBytes))
				eventbytes := Event{Data: string(PrfAsBytes), Txid: stub.GetTxID()}
				payload, err := json.Marshal(eventbytes)
				if err != nil {
					logger.Errorf("batchPreferences : Event Payload Marshalling Error :" + string(err.Error()))
					return shim.Error("batchPreferences : Event Payload Marshalling Error :" + string(err.Error()))
				}
				err2 := stub.SetEvent(EVTADDPREFERENCES, []byte(payload))
				if err2 != nil {
					logger.Errorf("batchPreferences : Event Creation Error for EventID : " + string(EVTADDPREFERENCES))
					return shim.Error("batchPreferences : Event Creation Error for EventID : " + string(EVTADDPREFERENCES))
				}
				logger.Infof("batchPreferences : Event Payload data : " + string(payload))
			} else {
				jsonResp = "{\"msisdn\":\"value\",\"svcprv\":\"value\",\"reqno\":\"value\",\"rmode\":\"value\",\"ctgr\":\"value\",\"cmode\":\"value\",\"day\":\"value\",\"time\":\"value\",\"lrn\":\"value\",\"uts\":\"value\",\"cts\":\"value\"}"
				logger.Errorf("batchPreferences : Incorrect Number Of Arguments, Expected json structure : " + string(jsonResp))
				out := Output{Data: args[i], ErrorDetails: "IncorrectNumberOfArGumentsExceptin[11 keys]"}
				edata, err := json.Marshal(out)
				if err != nil {
					logger.Errorf("batchPreferences : Marshalling Error : " + string(err.Error()))
					return shim.Error("batchPreferences : Marshalling Error : " + string(err.Error()))
				}
				result = append(result, string(edata))
				errorCount = errorCount + 1
				continue
			}
		} else {

			if len(data) == 11 {
				var organizationName string
				var orgName string

				preference := Preference{}
				err := json.Unmarshal(value, &preference)
				if err != nil {
					logger.Errorf("batchPreferences : Unmarhsaling Error : " + string(err.Error()))
					return shim.Error("batchPreferences : Unmarhsaling Error : " + string(err.Error()))
				}
				orgName = preference.UpdatedBy
				organizationName = Organizations[0]
				if strings.Compare(orgName, organizationName) == 0 {
					PrfStruct := &Preference{}
					PrfStruct.ObjType = "Preferences"
					PrfStruct.Phone = data["msisdn"].(string)
					PrfStruct.ServiceProvider = data["svcprv"].(string)
					PrfStruct.RequestNumber = data["reqno"].(string)
					PrfStruct.RegistrationMode = data["rmode"].(string)
					PrfStruct.Category = data["ctgr"].(string)
					PrfStruct.CommunicationMode = data["cmode"].(string)
					PrfStruct.DayType = data["day"].(string)
					PrfStruct.DayTimeBand = data["time"].(string)
					PrfStruct.Lrn = data["lrn"].(string)
					PrfStruct.UpdateTs = data["uts"].(string)
					PrfStruct.CreateTs = data["cts"].(string)
					PrfStruct.UpdatedBy = Organizations[0]
					logger.Infof("msisdn is " + PrfStruct.Phone)
					PrfAsBytes, err := json.Marshal(PrfStruct)
					if err != nil {
						logger.Errorf("batchPreferences : Marshalling Error : " + string(err.Error()))
						return shim.Error("batchPreferences : Marshalling Error : " + string(err.Error()))
					}
					//Inserting DataBlock to BlockChain
					err = stub.PutState(PrfStruct.Phone, PrfAsBytes)
					if err != nil {
						logger.Errorf("batchPreferences : PutState Failed Error : " + string(err.Error()))
						return shim.Error("batchPreferences : PutState Failed Error : " + string(err.Error()))
					}
					logger.Infof("batchPreferences : PutState Success : " + string(PrfAsBytes))
					eventbytes := Event{Data: string(PrfAsBytes), Txid: stub.GetTxID()}
					payload, err := json.Marshal(eventbytes)
					if err != nil {
						logger.Errorf("batchPreferences : Event Payload Marshalling Error : " + string(err.Error()))
						return shim.Error("batchPreferences : Event Payload Marshalling Error : " + string(err.Error()))
					}
					err2 := stub.SetEvent(EVTUPDATEPREFERENCES, []byte(payload))
					if err2 != nil {
						logger.Errorf("batchPreferences : Event Creation Error for EventID : " + string(EVTUPDATEPREFERENCES))
						return shim.Error("batchPreferences : Event Creation Error for EventID : " + string(EVTUPDATEPREFERENCES))
					}
					logger.Infof("Event published data " + string(payload))
					//txid := stub.GetTxID()
					//return shim.Success([]byte("batchPreferences : Preferences data added Successfully for MSISDN : " + PrfStruct.Phone + " , TransactionID      " + txid))
				} else {
					logger.Errorf("Unauthorized Access")
					out := Output{Data: args[i], ErrorDetails: "Unaithorized Access"}
					edata, err := json.Marshal(out)
					if err != nil {
						logger.Errorf("batchPreferences : Marshalling Error : " + string(err.Error()))
						return shim.Error("batchPreferences : Marshalling Error : " + string(err.Error()))
					}
					result = append(result, string(edata))
					errorCount = errorCount + 1
					continue
				}
			} else {
				jsonResp = "{\"msisdn\":\"value\",\"svcprv\":\"value\",\"reqno\":\"value\",\"rmode\":\"value\",\"ctgr\":\"value\",\"cmode\":\"value\",\"day\":\"value\",\"time\":\"value\",\"lrn\":\"value\",\"uts\":\"value\",\"cts\":\"value\"}"
				logger.Errorf("batchPreferences : Incorrect Number Of Arguments, Expected json structure : " + string(jsonResp))
				out := Output{Data: args[i], ErrorDetails: "IncorrectNumberOfArGumentsExceptin[11 keys]"}
				edata, err := json.Marshal(out)
				if err != nil {
					logger.Errorf("batchPreferences : Marshalling Error : " + string(err.Error()))
					return shim.Error("batchPreferences : Marshalling Error : " + string(err.Error()))
				}
				result = append(result, string(edata))
				errorCount = errorCount + 1
				continue
			}
		}
	}
	logger.Infof("ErrorCount is " + string(errorCount))
	if errorCount == 0 {
		txid := stub.GetTxID()
		return shim.Success([]byte("batchPreferences : Batch Preferences data added Successfully. TransactionID : " + txid))
	} else {
		response := strings.Join(result, "|")
		return shim.Success([]byte("batchPreferences : Updating batch Error : " + string(response)))
	}

}

//=============================================================================================================
//delPreferences for Removing or to churn out preference from DL based on MSISDN on successful certificate check
//==============================================================================================================

func (dlp *CPM) delPreferences(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		logger.Errorf("delPreferences : Incorrect Number Of Arguments, MSISDN Expected.")
		return shim.Error("delPreferences : Incorrect Number Of Arguments, MSISDN Expected.")
	}
	value, err := stub.GetState(args[0])
	if err != nil {
		logger.Errorf("delPreferences : GetState Failed for MSISDN : " + string(args[0]) + " , Error : " + string(err.Error()))
		return shim.Error("delPreferences : GetState Failed for MSISDN :" + string(args[0]) + " , Error : " + string(err.Error()))
	}
	if value == nil {
		logger.Info("delPreferences : No Existing preferences for MSISDN : " + string(args[0]))
		return shim.Success([]byte("delPreferences : No Existing preferences for MSISDN : " + string(args[0])))
	} else {
		var organizationName string
		var orgName string
		preference := Preference{}
		err := json.Unmarshal(value, &preference)
		if err != nil {
			logger.Errorf("delPreferences : Unmarshaling Error : " + string(err.Error()))
			return shim.Error("delPreferences : Unmarshaling Error : " + string(err.Error()))
		}
		certData, err := cid.GetX509Certificate(stub)
		if err != nil {
			logger.Errorf("delPreferences : Getting certificate Details Error : " + string(err.Error()))
			return shim.Error("delPreferences : Getting certificate Details Error : " + string(err.Error()))
		}
		Organizations := certData.Issuer.Organization
		orgName = preference.UpdatedBy
		organizationName = Organizations[0]
		if strings.Compare(orgName, organizationName) == 0 {
			err = stub.DelState(args[0])
			if err != nil {
				logger.Error("delPreferences : Removing Preferences from DLT error for MSISDN " + string(args[0]) + " , Error : " + string(err.Error()))
				return shim.Error("delPreferences : Removing Preferences from DLT error for MSISDN " + string(args[0]) + " , Error : " + string(err.Error()))
			}
			eventbytes := Event{Data: string(args[0]), Txid: stub.GetTxID()}
			payload, err := json.Marshal(eventbytes)
			if err != nil {
				logger.Errorf("delPreferences : Event Payload Marshalling Error : " + string(err.Error()))
				return shim.Error("delPreferences : Event Payload Marshalling Error : " + string(err.Error()))
			}
			err2 := stub.SetEvent(EVTDELPREFERENCES, []byte(payload))
			if err2 != nil {
				logger.Errorf("delPreferences : Event Creation Error for EventID : " + string(EVTDELPREFERENCES))
				return shim.Error("delPreferences : Event Creation Error for EventID : " + string(EVTDELPREFERENCES))
			}
			logger.Infof("delPreferences : Event Payload Data : " + string(args[0]))

		} else {
			logger.Errorf("Unauthorized access")
			return shim.Error("Unauthorized access")
		}
	}
	txid := stub.GetTxID()
	return shim.Success([]byte("Preferences is deleled from dlt for msisdn is " + string(args[0]) + "with TransactionID is " + string(txid)))
}

//=====================================================
//portOut for Ownership transfer from Donor to acceptor
//=====================================================

func (dlp *CPM) portOut(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string

	logger.Infof("data %v", args)
	logger.Infof("data %v", args[0])
	if len(args) != 3 {
		logger.Errorf("portOut : Incorrect number of arguments, Excepted 3 [msisdn,serviceprovide,updatedtime]")
		return shim.Error("portOut : Incorrect number of arguments, Excepted 3  [msisdn,serviceprovide,updatedtime]")
	}
	if _, err := strconv.Atoi(args[0]); err != nil {
		jsonResp = "{\"Error\":\"MSISDN is not numeric \"}"
		logger.Infof("datass %s", err)
		return shim.Error(jsonResp)
	}
	if len(args[0]) < 10 {
		jsonResp = "{\"Error\":\"MSISDN is not a valid length \"}"
		return shim.Error(jsonResp)
	}
	value, err := stub.GetState(args[0])
	if err != nil {
		logger.Errorf("portOut : GetState Failed for MSISDN : " + string(args[0]) + " , Error : " + string(err.Error()))
		return shim.Error("portOut : GetState Failed for MSISDN : " + string(args[0]) + " , Error : " + string(err.Error()))
	}
	if value == nil {
		logger.Info("portOut : No Existing preferences for MSISDN : " + string(args[0]))
		return shim.Success([]byte("portOut : No Existing preferences for MSISDN : " + string(args[0])))
	} else {
		var organizationName string
		var orgName string
		preference := Preference{}
		err := json.Unmarshal(value, &preference)
		if err != nil {
			logger.Errorf("portOut : Existing prefernce data Unmarhsaling Error : " + string(err.Error()))
			return shim.Error("portOut : Existing prefernce data Unmarhsaling Error : " + string(err.Error()))
		}
		certData, err := cid.GetX509Certificate(stub)
		if err != nil {
			logger.Errorf("portOut : Getting certificate Details Error : " + string(err.Error()))
			return shim.Error("portOut : Getting certificate Details Error : " + string(err.Error()))
		}
		Organizations := certData.Issuer.Organization
		orgName = preference.UpdatedBy
		organizationName = Organizations[0]
		if strings.Compare(orgName, organizationName) == 0 {
			PrfStruct := &Preference{}
			PrfStruct.ObjType = "Preferences"
			PrfStruct.Phone = args[0]
			PrfStruct.ServiceProvider = args[1]
			PrfStruct.RequestNumber = preference.RequestNumber
			PrfStruct.RegistrationMode = preference.RegistrationMode
			PrfStruct.Category = preference.Category
			PrfStruct.CommunicationMode = preference.CommunicationMode
			PrfStruct.DayType = preference.DayType
			PrfStruct.DayTimeBand = preference.DayTimeBand
			PrfStruct.Lrn = preference.Lrn
			PrfStruct.UpdateTs = args[2]
			PrfStruct.CreateTs = preference.CreateTs
			PrfStruct.UpdatedBy = Organizations[0]
			logger.Infof("msisdn is " + PrfStruct.Phone)
			PrfAsBytes, err := json.Marshal(PrfStruct)
			if err != nil {
				logger.Errorf("portOut : Marshalling Error : " + string(err.Error()))
				return shim.Error("portOut : Marshalling Error : " + string(err.Error()))
			}
			//Inserting DataBlock to BlockChain
			err = stub.PutState(PrfStruct.Phone, PrfAsBytes)
			if err != nil {
				logger.Errorf("portOut : PutState Failed Error : " + string(err.Error()))
				return shim.Error("portOut : PutState Failed Error : " + string(err.Error()))
			}
			logger.Infof("portOut : PutState Success : " + string(PrfAsBytes))
			eventbytes := Event{Data: string(PrfAsBytes), Txid: stub.GetTxID()}
			payload, err := json.Marshal(eventbytes)
			if err != nil {
				logger.Errorf("portOut : Event Payload Marshalling Error : " + string(err.Error()))
				return shim.Error("portOut : Event Payload Marshalling Error : " + string(err.Error()))
			}
			err = stub.SetEvent(EVTPORTOUT, []byte(payload))
			if err != nil {
				logger.Errorf("portOut : Event Creation Error for EventID : " + string(EVTPORTOUT))
				return shim.Error("portOut : Event Creation Error for EventID : " + string(EVTPORTOUT))
			}
			logger.Infof("portOut : Event Payload data : " + string(payload))
			txid := stub.GetTxID()
			return shim.Success([]byte("portOut : PutState Success for MSISDN : " + PrfStruct.Phone + " , TransactionID : " + txid))
		} else {
			logger.Errorf("Unauthorized Access")
			return shim.Error("Unauthorized Access")
		}
	}
}

//======================================================================================
//queryPreferences RichQuery for Obtaining Preference data
//======================================================================================

func (dlp *CPM) queryPreferences(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("queryPreferences : Incorrect number of arguments, Expected 1 [Query String]")
	}
	queryString := args[0]
	logger.Info(args[0])
	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		logger.Errorf("queryPreferences : getQueryResultForQueryString Failed Error : " + string(err.Error()))
		return shim.Error("queryPreferences : getQueryResultForQueryString Failed Error : " + string(err.Error()))
	}
	return shim.Success(queryResults)
}

func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {
	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// ===========================================================================================
// constructQueryResponseFromIterator constructs a JSON array containing query results from
// a given result iterator
// ===========================================================================================
func constructQueryResponseFromIterator(resultsIterator shim.StateQueryIteratorInterface) (*bytes.Buffer, error) {
	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return &buffer, nil
}

// ===================================================================================
//main function for the preference ChainCode
// ===================================================================================
func main() {
	err := shim.Start(new(CPM))
	logger.SetLevel(shim.LogDebug)
	if err != nil {
		logger.Error("Error Starting Cpm Chaincode is " + string(err.Error()))
	} else {
		logger.Info("Starting CPM Chaincode")
	}
}
