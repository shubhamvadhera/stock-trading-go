package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"regexp"
	"strconv"
	"strings"
)

//Structs

//BuyRequest struct request from client
type BuyRequest struct {
	StockSymbolAndPercentage string
	Budget                   float64
}

//BuyResponse struct response to client
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

//------ Some internal structs----------//
//marshal buyRequest into symbols and values
type buyStock struct {
	symb  string
	value float64
}

type responseStock struct {
	symb        string
	numOfStocks int
	origVal     float64
}

type responseCheck struct {
	symb        string
	numOfStocks int
	currVal     float64
	pl          string
}

type serverRecord struct {
	rs      []responseStock
	balance float64
}

//response to JSON query struct
type jsonStruct struct {
	Query struct {
		Count   int    `json:"count"`
		Created string `json:"created"`
		Lang    string `json:"lang"`
		Results struct {
			Quote []struct {
				LastTradePriceOnly string `json:"LastTradePriceOnly"`
				Symbol             string `json:"symbol"`
			} `json:"quote"`
		} `json:"results"`
	} `json:"query"`
}

type jsonStruct1 struct {
	Query struct {
		Count   int    `json:"count"`
		Created string `json:"created"`
		Lang    string `json:"lang"`
		Results struct {
			Quote struct {
				LastTradePriceOnly string `json:"LastTradePriceOnly"`
				Symbol             string `json:"symbol"`
			} `json:"quote"`
		} `json:"results"`
	} `json:"query"`
}

var tradeIDStart int
var database map[int]serverRecord

//Transaction type struct
type Transaction struct{}

//Functions

//Check function to check portfolio
func (t *Transaction) Check(receive []byte, reply *[]byte) error {
	var checkRequestVar CheckRequest
	json.Unmarshal(receive, &checkRequestVar)
	fmt.Println("CheckRequest: ", string(receive))
	var response CheckResponse
	if _, ok := database[checkRequestVar.TradeID]; ok {
		sr := database[checkRequestVar.TradeID]
		//fmt.Println("Dev: func Check: sr.rs=", sr.rs)
		bs := responseStockToBuyStock(sr.rs)
		url := buildURL(bs)
		jsonData := urlToJSONStruct(url)
		m, _ := jsonStructToMap(jsonData)
		rc, cv := profitLoss(m, sr)
		response = makeCheckResponse(rc, cv, sr.balance)
	}
	out, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	fmt.Println("CheckResponse: ", string(out))
	*reply = out
	return nil
}

func makeCheckResponse(rc []responseCheck, currValue float64, unvAmt float64) CheckResponse {
	var cr CheckResponse
	for _, x := range rc {
		cr.Stocks += x.symb + ":" + strconv.Itoa(x.numOfStocks) + ":" + x.pl + "$" + strconv.FormatFloat(x.currVal, 'f', -1, 64) + ","
	}
	cr.Stocks = cr.Stocks[:len(cr.Stocks)-1]
	cr.CurrentMarketValue = currValue
	cr.UnvestedAmount = unvAmt
	return cr
}

func profitLoss(dataMap map[string]float64, sr serverRecord) ([]responseCheck, float64) {

	var rc = make([]responseCheck, len(sr.rs))
	var totalCurrValue float64
	for i, x := range sr.rs {
		rc[i].symb = x.symb
		rc[i].numOfStocks = x.numOfStocks
		rc[i].currVal = dataMap[x.symb]
		totalCurrValue += (rc[i].currVal * float64(rc[i].numOfStocks))
		if rc[i].currVal < x.origVal {
			rc[i].pl = "-"
		} else if rc[i].currVal > x.origVal {
			rc[i].pl = "+"
		}
	}
	return rc, totalCurrValue
}

func responseStockToBuyStock(rs []responseStock) []buyStock {
	//fmt.Println("Dev: func responseStockToBuyStock: len(rs)=", len(rs))
	var bs []buyStock
	if len(rs) == 1 {
		bs = make([]buyStock, 2)
	} else {
		bs = make([]buyStock, len(rs))
	}
	for i, r := range rs {
		bs[i].symb = r.symb
	}
	if len(rs) == 1 {
		bs[1].symb = "X.Y.Z"
	}
	return bs
}

//Buy function to perform Buy tasks
func (t *Transaction) Buy(receive []byte, reply *[]byte) error {
	var buyRequestVar BuyRequest
	json.Unmarshal(receive, &buyRequestVar)
	fmt.Println("BuyRequest: ", string(receive))
	var response BuyResponse
	buyStocks, er1 := buyRequestToBuyStock(buyRequestVar)
	if er1 == 1 {
		response.Stocks = "Sum of percentages not 100"
	} else if er1 == 2 {
		response.Stocks = "Request format invalid"
	} else {
		url := buildURL(buyStocks)
		jsonData := urlToJSONStruct(url)
		m, er2 := jsonStructToMap(jsonData)
		if er2 == false {
			responseStocks, unvestedAmount := stockBuyer(buyStocks, m, buyRequestVar.Budget)
			response = responseStocksToBuyResponse(responseStocks, unvestedAmount)
		} else {
			response.Stocks = "Invalid Symbol / Symbol not found"
		}
	}
	out, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	fmt.Println("BuyResponse: ", string(out))
	*reply = out
	return nil
}

//internal: converts buy request to buystock
func buyRequestToBuyStock(r BuyRequest) ([]buyStock, int) {
	checkString := r.StockSymbolAndPercentage
	if !strings.Contains(checkString, ",") {
		checkString += ",X.Y.Z:0%"
	}
	string2 := strings.Split(checkString, ",")
	//fmt.Println("len(string2): ", len(string2))
	var buyStocks = make([]buyStock, len(string2))
	var totpercent float64
	var err int
	for i, stock := range string2 {
		exp := regexp.MustCompile("\\b(\\w|\\.|\\d)+:\\d+\\.?(\\d+)?%")
		matched := exp.MatchString(stock)
		if !matched {
			err = 2
		} else {
			temp := strings.Split(stock, ":")
			buyStocks[i].symb = strings.ToUpper(temp[0])
			var temp2 float64
			temp2, _ = strconv.ParseFloat(temp[1][:len(temp[1])-1], 64)
			totpercent += temp2
			buyStocks[i].value = temp2 * 0.01 * r.Budget
		}
	}
	if totpercent != 100 {
		err = 1
	}
	return buyStocks, err
}

//function to build URL string
func buildURL(bs []buyStock) string {
	urlP1 := "https://query.yahooapis.com/v1/public/yql?q=select%20LastTradePriceOnly%2CSymbol%20from%20yahoo.finance.quote%20where%20symbol%20in"
	urlP3 := "&format=json&env=store%3A%2F%2Fdatatables.org%2Falltableswithkeys&callback="
	urlP2 := "("
	for _, x := range bs {
		urlP2 += "\"" + x.symb + "\","
	}
	urlP2 = urlP2[:len(urlP2)-1]
	urlP2 += ")"
	url := urlP1 + urlP2 + urlP3
	//fmt.Println("Dev: func buildURL: URL=", url)
	return url
}

func urlToJSONStruct(url string) jsonStruct {
	res, err := http.Get(url)
	defer res.Body.Close()
	if err != nil {
		panic(err)
	}
	jsonDataFromHTTP, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var data jsonStruct
	json.Unmarshal([]byte(jsonDataFromHTTP), &data)
	//fmt.Println("Dev: func urlToJSONStruct: data=", data)
	//fmt.Println("Dev: func urlToJSONStruct: data=", data)
	return data
}

func stockBuyer(bs []buyStock, dataMap map[string]float64, budget float64) ([]responseStock, float64) {
	var rs = make([]responseStock, len(dataMap))
	//fmt.Println("Dev: len(dataMap): ", len(dataMap))
	var unvestedAmount float64
	var bsnew []buyStock
	if len(bs) == 2 && bs[1].symb == "X.Y.Z" {
		bsnew = make([]buyStock, 1)
		bsnew[0] = bs[0]
	} else {
		bsnew = bs
	}
	for i, x := range bsnew {
		stockValue := dataMap[x.symb]
		rs[i].numOfStocks = int(x.value / stockValue)
		//fmt.Println("Dev: line 245 i=", i)
		rs[i].symb = x.symb
		rs[i].origVal = stockValue
		unvestedAmount += math.Mod(x.value, stockValue)
	}
	return rs, unvestedAmount
}

func jsonStructToMap(js jsonStruct) (map[string]float64, bool) {
	m := make(map[string]float64)
	var err bool
	//fmt.Println("Dev: line 258 jsonstruct=", js)
	for _, x := range js.Query.Results.Quote {
		if x.LastTradePriceOnly == "" && x.Symbol != "X.Y.Z" {
			err = true
		}
		if x.Symbol != "X.Y.Z" {
			m[x.Symbol], _ = strconv.ParseFloat(x.LastTradePriceOnly, 64)
		}
	}
	//fmt.Println("Dev: line 257 map=", m)
	return m, err
}

func responseStocksToBuyResponse(rs []responseStock, unvestedAmount float64) BuyResponse {
	var br BuyResponse
	tradeIDStart++
	br.TradeID = tradeIDStart
	for _, x := range rs {
		br.Stocks += x.symb + ":" + strconv.Itoa(x.numOfStocks) + ":$" + strconv.FormatFloat(x.origVal, 'f', -1, 64) + ","
	}
	br.Stocks = br.Stocks[:len(br.Stocks)-1]
	br.UnvestedAmount = unvestedAmount
	//record this trade in server database
	var sr serverRecord
	sr.rs = rs
	sr.balance = unvestedAmount
	database[tradeIDStart] = sr
	return br
}

//Main function
func main() {
	database = make(map[int]serverRecord)
	trans := new(Transaction)
	server := rpc.NewServer()
	server.Register(trans)
	server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	listener, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	for {
		if conn, err := listener.Accept(); err != nil {
			log.Fatal("accept error: " + err.Error())
		} else {
			log.Printf("new connection established\n")
			go server.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}
}
