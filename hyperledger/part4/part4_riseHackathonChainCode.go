/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

var smartPayIndexStr = "_smartpayindex" //name for the key/value that will store a list of all known marbles
var paymentIndexStr = "_paymentindex"

// PaymentTransaction simple Payment Transaction Schema
type PaymentTransaction struct {
	TransactionID string `json:"transactionID"` //the fieldtags are needed to keep case from bouncing around
	DrawerID      string `json:"drawerID"`
	PayeeID       string `json:"payeeID"`
	Amount        int    `json:"amount"`
	Currency      string `json:"currency"`
}

// RemittanceTransaction simple Remittance Transation Schema
type RemittanceTransaction struct {
	TransactionID       string `json:"transactionID"` //the fieldtags are needed to keep case from bouncing around
	SourceID            string `json:"sourceID"`
	SourceCurrency      string `json:"sourceCurrency"`
	DestinationID       string `json:"destinationID"`
	DestinationCurrency string `json:"destinationCurrency"`
	Amount              int    `json:"amount"`
	ExchangeRate        int    `json:"ExchangeRate"`
}

// LendingTransacation simple Lending Transaction Schema
type LendingTransacation struct {
	TransactionID  string `json:"transactionID"` //the fieldtags are needed to keep case from bouncing around
	LendorID       string `json:"lendorID"`
	BorrowerID     string `json:"borrowerID"`
	LoanAmount     int    `json:"loanAmount"`
	Currency       string `json:"currency"`
	LoanRate       int    `json:"loanRate"`
	LoanReturnDate int64  `json:"loanReturnDate"`
}

// SmartPayTransaction simple SmartPay Transaction Schema
type SmartPayTransaction struct {
	TransactionID string                `json:"transactionID"` //user who created the open trade order
	PaymentTrans  PaymentTransaction    `json:"paymentTrans"`  //description of desired marble
	RemitTrans    RemittanceTransaction `json:"remitTrans"`    //array of marbles willing to trade away
	LendTrans     LendingTransacation   `json:"lentTrans"`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// ============================================================================================================================
// Init - reset all the things
// ============================================================================================================================
func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	var Aval int
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	// Initialize the chaincode
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Expecting integer value for asset holding")
	}

	// Write the state to the ledger
	err = stub.PutState("abc", []byte(strconv.Itoa(Aval))) //making a test var "abc", I find it handy to read/write to it right away to test the network
	if err != nil {
		return nil, err
	}

	var empty []string
	jsonAsBytes, _ := json.Marshal(empty) //marshal an emtpy array of strings to clear the index
	err = stub.PutState(smartPayIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	err = stub.PutState(paymentIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// ============================================================================================================================
// Run - Our entry point for Invocations - [LEGACY] obc-peer 4/25/2016
// ============================================================================================================================
func (t *SimpleChaincode) Run(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("run is running " + function)
	return t.Invoke(stub, function, args)
}

// ============================================================================================================================
// Invoke - Our entry point for Invocations
// ============================================================================================================================
func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" { //initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "delete" { //deletes an entity from its state
		res, err := t.Delete(stub, args) //lets make sure all open trades are still valid
		return res, err
	} else if function == "write" { //writes a value to the chaincode state
		return t.Write(stub, args)
	} else if function == "initPayment" { //create a new Payment
		return t.initPayment(stub, args)
	} else if function == "newEcrire" { //writes a value to the chaincode state
		return t.NewEcrire(stub, args)
	}
	fmt.Println("invoke did not find func: " + function) //error

	return nil, errors.New("Received unknown function invocation")
}

// ============================================================================================================================
// Query - Our entry point for Queries
// ============================================================================================================================
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" { //read a variable
		return t.read(stub, args)
	}
	fmt.Println("query did not find func: " + function) //error

	return nil, errors.New("Received unknown function query")
}

// ============================================================================================================================
// Read - read a variable from chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) read(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the var to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name) //get the var from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil //send it onward
}

// ============================================================================================================================
// Delete - remove a key/value pair from state
// ============================================================================================================================
func (t *SimpleChaincode) Delete(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	name := args[0]
	err := stub.DelState(name) //remove the key from chaincode state
	if err != nil {
		return nil, errors.New("Failed to delete state")
	}

	//get the smartPay index
	smartPayTransactionAsBytes, err := stub.GetState(smartPayIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get SmartPayTransaction index")
	}
	var smartPayIndex []string
	json.Unmarshal(smartPayTransactionAsBytes, &smartPayIndex) //un stringify it aka JSON.parse()

	//remove marble from index
	for i, val := range smartPayIndex {
		fmt.Println(strconv.Itoa(i) + " - looking at " + val + " for " + name)
		if val == name { //find the correct marble
			fmt.Println("Found SmartPay Transaction")
			smartPayIndex = append(smartPayIndex[:i], smartPayIndex[i+1:]...) //remove it
			for x := range smartPayIndex {                                    //debug prints...
				fmt.Println(string(x) + " - " + smartPayIndex[x])
			}
			break
		}
	}
	jsonAsBytes, _ := json.Marshal(smartPayIndex) //save new index
	err = stub.PutState(smartPayIndexStr, jsonAsBytes)
	return nil, nil
}

// ============================================================================================================================
// Write - write variable into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) Write(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var name, value string // Entities
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the variable and value to set")
	}

	name = args[0] //rename for funsies
	value = args[1]
	err = stub.PutState(name, []byte(value)) //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// ============================================================================================================================
// Ecrire - Prepend 9999: and write variable into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) NewEcrire(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var name, value string // Entities
	var err error
	fmt.Println("running Ecrire()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the variable and value to set")
	}

	name = args[0] //rename for funsies
	value = "SmartPayTransactions:" + args[1]
	err = stub.PutState(name, []byte(value)) //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// ============================================================================================================================
// Init Payment - create a new marble, store into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) initPayment(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var err error
	//   0       1          2          3       4
	// "asdf", "blue", "35", "bob"
	// TransId  DrawerID   PayeeID   Amount   Currency

	if len(args) != 5 {
		return nil, errors.New("Incorrect number of arguments. Expecting 5")
	}

	//input sanitation
	fmt.Println("- start init marble")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("3rd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return nil, errors.New("4th argument must be a non-empty string")
	}
	if len(args[4]) <= 0 {
		return nil, errors.New("5th argument must be a non-empty string")
	}
	transID := args[0]
	drawerID := strings.ToLower(args[1])
	payeeID := strings.ToLower(args[2])
	amount, err := strconv.Atoi(args[3])
	if err != nil {
		return nil, errors.New("3rd argument must be a numeric string")
	}
	currency := strings.ToLower(args[4])

	//check if Payment already exists
	paymentAsBytes, err := stub.GetState(transID)
	if err != nil {
		return nil, errors.New("Failed to get Transaction name")
	}
	res := PaymentTransaction{}
	json.Unmarshal(paymentAsBytes, &res)
	if res.TransactionID == transID {
		fmt.Println("This Payment Transaction arleady exists: " + transID)
		fmt.Println(res)
		return nil, errors.New("This PaymentTranaction arleady exists") //all stop a marble by this name exists
	}

	//build the Payment json string manually
	str := `{"transactionID": "` + transID + `", "drawerID": "` + drawerID + `, "payeeID": "` + payeeID + `", "amount": ` + strconv.Itoa(amount) + transID + `", "currency": "` + currency + `"}`
	err = stub.PutState(transID, []byte(str)) //store marble with id as key
	if err != nil {
		return nil, err
	}

	//get the Payment index
	paymentAsBytes, err = stub.GetState(paymentIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get marble index")
	}
	var paymentIndex []string
	json.Unmarshal(paymentAsBytes, &paymentIndex) //un stringify it aka JSON.parse()

	//append
	paymentIndex = append(paymentIndex, transID) //add marble name to index list
	fmt.Println("! Payment index: ", paymentIndex)
	jsonAsBytes, _ := json.Marshal(paymentIndex)
	err = stub.PutState(paymentIndexStr, jsonAsBytes) //store name of marble

	fmt.Println("- End initPayment")
	return nil, nil
}
