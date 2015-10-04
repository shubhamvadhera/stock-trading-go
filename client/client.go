package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/rpc/jsonrpc"
	"os"
	"strconv"
	"strings"
)

//BuyRequest struct for buy request to server
type BuyRequest struct {
	StockSymbolAndPercentage string
	Budget                   float64
}

//BuyResponse struct for response to buy from server
type BuyResponse struct {
	TradeID        int
	Stocks         string
	UnvestedAmount float64
}

//CheckRequest struct to check portfolio to server
type CheckRequest struct {
	TradeID int
}

//CheckResponse struct to receive portfolio from server
type CheckResponse struct {
	Stocks             string
	CurrentMarketValue float64
	UnvestedAmount     float64
}

func main() {
	a := os.Args[1:]
	if strings.ToLower(a[0]) == "buy" {
		sendBuy(a[1], a[2])
	} else if strings.ToLower(a[0]) == "check" {
		sendCheck(a[1])
	} else {
		defer func() {
			str := recover()
			fmt.Println(str)
		}()
		panic("Please enter proper keyword buy or check followed by necessary arguments")
	}
}

func sendCheck(argTradeIDString string) {
	var cr CheckRequest
	var er error
	cr.TradeID, er = strconv.Atoi(argTradeIDString)
	if er != nil {
		panic(er)
	}
	send, err := json.Marshal(cr)
	if err != nil {
		panic(err)
	}
	// Synchronous call
	client, err := net.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	var reply []byte
	c := jsonrpc.NewClient(client)
	err = c.Call("Transaction.Check", send, &reply)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(reply))
}

func sendBuy(argSymbolString string, argBudgetString string) {
	buyRequest := argToBuyRequest(argSymbolString, argBudgetString)
	send, err := json.Marshal(buyRequest)
	if err != nil {
		panic(err)
	}
	// Synchronous call
	client, err := net.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	var reply []byte
	c := jsonrpc.NewClient(client)
	err = c.Call("Transaction.Buy", send, &reply)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(reply))
}

//Marshal command line arguments into BuyRequest Struct
func argToBuyRequest(arg1 string, arg2 string) BuyRequest {
	var br BuyRequest
	var err error
	br.Budget, err = strconv.ParseFloat(arg2, 64)
	if err != nil {
		defer func() {
			str := recover()
			fmt.Println(str)
		}()
		panic("Please enter proper float value")
	}
	br.StockSymbolAndPercentage = arg1
	return br
}
