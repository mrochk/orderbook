# Exchange
Market order book simulation, callable via an API.\
FIFO algorithm is used for matching orders.

## Usage
**Compile the project:**
First, clone the repository, then:
```shell
cd orderbook
```
then
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
## API Endpoints
*The API documentation hasn't been written yet.*

```
POST /init
```

```
POST /limit_order
```

```
POST /cancel_order
```

```
POST /market_order
```

```
GET /get_data
```
