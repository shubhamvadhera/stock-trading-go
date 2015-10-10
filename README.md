#Virtual Stock Trading System - GOLANG

This is a virtual stock trading system for whoever wants to learn how to invest in stocks.

The system uses real-time pricing via Yahoo finance API and supports USD currency only. The system has two features:

Buying stocks
Request
“stockSymbolAndPercentage”: string (E.g. “GOOG:55%,YHOO:45%”)
“budget” : float64
Response
“tradeId”: number
“stocks”: string (E.g. “GOOG:51:$500.25”, “YHOO:252:$31.40”)
“unvestedAmount”: float64

Command: go run client.go buy "GOOG:55%,YHOO:45%" 50000

Checking your portfolio (loss/gain)
Request
“tradeId”: number
Response
“stocks”: string (E.g. “GOOG:51:+$520.25”, “YHOO:252:-$30.40”) //+,- symbols indicate if currently in profit or loss
“currentMarketValue” : float32
“unvestedAmount”: float32

Command: go run client.go check <TradeId>

The system has 2 components: client and server.
server: the trading engine will have JSON-RPC interface for the above features.
client: the JSON-RPC client will take command line input and send requests the server.
