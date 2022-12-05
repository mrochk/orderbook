# Exchange
Market order book simulation, callable via an API.\
FIFO algorithm is used for matching orders.

## Usage
First, clone the repository, then:
```shell
cd orderbook
```
**Compile the project:**
```shell
make
```
or
```shell
go build -o bin/exchange
```
**Launch the server (runs on port 8080 by default):**
```shell
./bin/exchange <PORT>
```
## JSON API Endpoints
***
```
POST /init
```
*The user must run this command before using the order book.* \
*The user can also use it to reinitialize the order book without restarting the server.* \
**Body :**\
"mid_price" : float, The default midprice.
***
```
POST /limit_order
```
*Places a limit order at a certain price, if it is possible.* \
*Returns the placed order id to the user (used to cancel the order).* \
**Body :**\
"type" : bool, The type of the order (buy/sell), true for a buy order.\
"price" : float, The price the user wants to place the order at.\
"qty" : float, The quantity the user wants to buy or sell.
***

```
POST /market_order
```
*Executes a market order, if there is enough volume to.* \
**Body :**\
"type" : bool, The type of the order (buy/sell), true for a buy order.\
"qty" : float, The quantity the user wants to buy or sell.
***
```
POST /cancel_order
```
*Cancels the limit order corresponding to the order id provided by the user if it was not already executed.* \
**Body :**\
"id" : uuid, The order id, given when the user places a limit order.\
"price" : float, The price at which the order is sitting.
***

```
GET /get_data
```
*Returns the order book data.* 
