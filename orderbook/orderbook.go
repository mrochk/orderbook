package orderbook

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

/*
An order is defined by an unique id, a certain
quantity the user wants to buy, and a timestamp.
*/
type Order struct {
	id        uuid.UUID
	qty       float64
	timestamp int64
	next      *Order
}

/*
Returns a ptr to a new order with a quantity
defined by the user.
*/
func newOrder(qty float64) *Order {
	return &Order{
		id:        uuid.New(),
		qty:       qty,
		timestamp: time.Now().UnixNano(),
		next:      nil,
	}
}

/*
We use a queue to store the orders so we
don't need to re-sort the orders by timestamp
whenever we delete one.
*/
type OrderQueue struct {
	rear  *Order
	front *Order
}

/*
Returns a ptr to a new OrderQueue, rear
and front are initialized to nil.
*/
func newOrderQueue() *OrderQueue {
	return &OrderQueue{
		rear:  nil,
		front: nil,
	}
}

/*
A limit is defined by his Price, the orders that
are sitting at this Price, and his Volume (the
sum of the quantity of all the orders it contains).
*/
type Limit struct {
	Price  float64
	Volume float64
	orders *OrderQueue
}

/*
Returns a ptr to a new limit sitting at a Price
defined by the user, and containing no orders.
*/
func newLimit(Price float64) *Limit {
	return &Limit{
		Price:  Price,
		Volume: 0,
		orders: newOrderQueue(),
	}
}

/*
Add the order to the limit orders queue.
*/
func (l *Limit) addOrder(o *Order) {
	if l.orders.rear != nil {
		l.orders.rear.next = o
		l.orders.rear = l.orders.rear.next
	} else {
		l.orders.rear = o
		l.orders.front = l.orders.rear
	}
	l.Volume += o.qty
}

/*
Deletes the order having this id from the
order queue and keeps them ordered.
*/
func (l *Limit) deleteOrder(id uuid.UUID) error {
	if l.orders.front.id == id {
		l.Volume -= l.orders.front.qty
		l.orders.front = l.orders.front.next
		return nil
	}
	temp := l.orders.front
	for temp.next != nil {
		if temp.next.id == id {
			l.Volume -= temp.next.qty
			temp.next = temp.next.next
			return nil
		}
		temp = temp.next
	}
	return errors.New("no order having this id in this limit")
}

/*
The order-book is simply a collection of buy and
sell limits sitting at certain Prices and containing
orders, we use slices and maps to access them.
*/
type OrderBook struct {
	BuyLimits     []*Limit
	SellLimits    []*Limit
	buyLimitsMap  map[float64]*Limit
	sellLimitsMap map[float64]*Limit
	Price         float64 // Price of the limit at which the last order was executed.
}

/*
Returns a ptr to a new empty order-book,
before it can be used, it needs to be
initialized with the Init() function.
*/
func New() *OrderBook {
	return &OrderBook{
		BuyLimits:     []*Limit{},
		SellLimits:    []*Limit{},
		buyLimitsMap:  make(map[float64]*Limit),
		sellLimitsMap: make(map[float64]*Limit),
		Price:         0,
	}
}

/*
Sets the order-book default midPrice by adding
an empty buy and sell limit at this Price, both
containing 0 orders, the user can also configure
the name of the asset traded.
*/
func (ob *OrderBook) Init(midPrice float64) {
	var (
		l = newLimit(midPrice)
		o = newOrder(0)
	)
	l.addOrder(o)
	ob.addLimit(true, l)
	ob.addLimit(false, l)
}

/*
Resets the orderbook by deleting all the
orders and limits it contains (thanks to
the Golang garbage collector).
*/
func (ob *OrderBook) Reset() {
	ob.BuyLimits = []*Limit{}
	ob.SellLimits = []*Limit{}
	ob.buyLimitsMap = make(map[float64]*Limit)
	ob.sellLimitsMap = make(map[float64]*Limit)
	ob.Price = 0
}

func (ob *OrderBook) GetData() ([]*Limit, []*Limit) {
	return ob.BuyLimits, ob.SellLimits
}

/*
Places a limit buy or sell order at a certain Price
and of a certain quantity. Returns the order id and
an error if it was not possible to place it.
*/
func (ob *OrderBook) PlaceLimitOrder(buyOrder bool, Price float64, qty float64) (uuid.UUID, error) {
	var (
		order         = newOrder(qty)
		midPrice, err = ob.getMidPrice()
	)

	if err != nil {
		return order.id, err
	} else if Price <= 0 {
		return order.id, errors.New("can't place order if Price <= 0")
	} else if qty <= 0 {
		return order.id, errors.New("can't place order if quantity <= 0")
	} else if buyOrder && Price > midPrice {
		return order.id, errors.New("can't place a buy limit order higher than midPrice")
	} else if !buyOrder && Price < midPrice {
		return order.id, errors.New("can't place a sell limit order lower than midPrice")
	}
	if buyOrder {
		// If there is no limit to place this order in.
		// We create a new limit at the corresponding Price.
		// We add our order to it.
		// We add the limit to our orderbook.
		// Finally, we sort the limits to get the highest Price 1st.
		if ob.buyLimitsMap[Price] == nil {
			limit := newLimit(Price)
			limit.addOrder(order)
			ob.addLimit(true, limit)
		} else {
			// If there is a limit to place it.
			// We first append it to the corresponding map.
			// After appending a new order we need to re sort
			// the slice to get the oldest orders 1st (FIFO).
			// Finally we sort the orders by timestamp in the bg.
			ob.buyLimitsMap[Price].addOrder(order)
		}
	} else {
		if ob.sellLimitsMap[Price] == nil {
			limit := newLimit(Price)
			limit.addOrder(order)
			ob.addLimit(false, limit)
		} else {
			ob.sellLimitsMap[Price].addOrder(order)
		}
	}

	if ob.canDeleteLimit(true) && ob.BuyLimits[0].Volume == 0 {
		ob.deleteLimit(true, 0, ob.BuyLimits[0].Price)
	} else if ob.canDeleteLimit(false) && ob.SellLimits[0].Volume == 0 {
		ob.deleteLimit(false, 0, ob.SellLimits[0].Price)
	}

	return order.id, nil
}

/*
Cancels a limit order if it was not already executed.
*/
func (ob *OrderBook) CancelLimitOrder(id uuid.UUID, Price float64) error {
	midPrice, err := ob.getMidPrice()
	if err != nil {
		return err
	}
	// If Price is lower than midPrice, the user wants to cancel a sell order.
	// We need to find the limit corresponding to the Price.
	// When we find the limit, we delete the order inside of it.
	// If limit Volume = 0 we can delete it from order book.
	if Price < midPrice {
		err = ob.buyLimitsMap[Price].deleteOrder(id)
		if err != nil {
			return err
		}
		if ob.canDeleteLimit(true) && ob.buyLimitsMap[Price].Volume == 0 {
			for i, limit := range ob.BuyLimits {
				if limit.Price == Price {
					ob.deleteLimit(true, i, Price)
				}
			}
		}
		return nil
	} else {
		err = ob.sellLimitsMap[Price].deleteOrder(id)
		if err != nil {
			return err
		}
		if ob.canDeleteLimit(false) && ob.sellLimitsMap[Price].Volume == 0 {
			for i, limit := range ob.SellLimits {
				if limit.Price == Price {
					fmt.Println(i)
					ob.deleteLimit(false, i, Price)
				}
			}
		}
		return nil
	}
}

/*
Executes a market buy or sell order of a certain
quantity that must be <= than the order-book total
buy or sell limits Volume.
*/
func (ob *OrderBook) PlaceMarketOrder(buyOrder bool, qty float64) error {
	// First, we delete the limits entirely while we can fill them.
	// Then, we delete the limit orders entirely while we can fill them.
	// Finally, we fill the last limit order partially.
	if qty <= 0 {
		return errors.New("error, market order qty <= 0")
	}
	if buyOrder {
		if qty >= ob.getTotalVolume(false) {
			return errors.New("can't execute market order : order qty > total Volume")
		}
		for len(ob.SellLimits) > 0 && qty >= ob.SellLimits[0].Volume {
			qty -= ob.SellLimits[0].Volume
			ob.Price = ob.SellLimits[0].Price
			ob.deleteLimit(false, 0, ob.SellLimits[0].Price)
		}
		for len(ob.SellLimits) > 0 && qty >= ob.SellLimits[0].orders.front.qty {
			ob.Price = ob.SellLimits[0].Price
			ob.SellLimits[0].Volume -= ob.SellLimits[0].orders.front.qty
			qty -= ob.SellLimits[0].orders.front.qty
			ob.SellLimits[0].orders.front = ob.SellLimits[0].orders.front.next
		}
		if qty != 0 {
			ob.Price = ob.SellLimits[0].Price
			ob.SellLimits[0].orders.front.qty -= qty
			ob.SellLimits[0].Volume -= qty
		}
	} else {
		if qty >= ob.getTotalVolume(true) {
			return errors.New("can't execute market order : order qty > total Volume")
		}
		for len(ob.BuyLimits) > 0 && qty >= ob.BuyLimits[0].Volume {
			qty -= ob.BuyLimits[0].Volume
			ob.Price = ob.BuyLimits[0].Price
			ob.deleteLimit(true, 0, ob.BuyLimits[0].Price)
		}
		for len(ob.BuyLimits) > 0 && qty >= ob.BuyLimits[0].orders.front.qty {
			ob.Price = ob.BuyLimits[0].Price
			ob.BuyLimits[0].Volume -= ob.BuyLimits[0].orders.front.qty
			qty -= ob.BuyLimits[0].orders.front.qty
			ob.BuyLimits[0].orders.front = ob.BuyLimits[0].orders.front.next
		}
		if qty != 0 {
			ob.Price = ob.BuyLimits[0].Price
			ob.BuyLimits[0].orders.front.qty -= qty
			ob.BuyLimits[0].Volume -= qty
		}
	}
	return nil
}

/*
Add the limit to the order-book and re-sorts
the slice.
*/
func (ob *OrderBook) addLimit(buyLimit bool, l *Limit) {
	if buyLimit {
		ob.buyLimitsMap[l.Price] = l
		ob.BuyLimits = append(ob.BuyLimits, l)
		sort.Sort(byHighestPrice(ob.BuyLimits))
	} else {
		ob.sellLimitsMap[l.Price] = l
		ob.SellLimits = append(ob.SellLimits, l)
		sort.Sort(byLowestPrice(ob.SellLimits))
	}
}

/*
We must not delete a limit if it's the last remaining
in the corresponding slice, because it would break
the getMidPrice function.
*/
func (ob *OrderBook) canDeleteLimit(buyLimit bool) bool {
	if buyLimit {
		return len(ob.BuyLimits) > 1
	}
	return len(ob.SellLimits) > 1
}

/*
Deletes the limit and re-sorts the slice.
*/
func (ob *OrderBook) deleteLimit(buyLimit bool, pos int, Price float64) {
	if buyLimit {
		ob.BuyLimits[pos] = ob.BuyLimits[len(ob.BuyLimits)-1]
		ob.BuyLimits = ob.BuyLimits[:len(ob.BuyLimits)-1]
		delete(ob.buyLimitsMap, Price)
		sort.Sort(byHighestPrice(ob.BuyLimits))
	} else {
		ob.SellLimits[pos] = ob.SellLimits[len(ob.SellLimits)-1]
		ob.SellLimits = ob.SellLimits[:len(ob.SellLimits)-1]
		delete(ob.sellLimitsMap, Price)
		sort.Sort(byLowestPrice(ob.SellLimits))
	}
}

/*
Returns the order-book midPrice, it needs
to have at least one buy and sell limit to work.
*/
func (ob *OrderBook) getMidPrice() (float64, error) {
	if len(ob.BuyLimits) > 0 && len(ob.SellLimits) > 0 {
		return (ob.BuyLimits[0].Price + ob.SellLimits[0].Price) / 2, nil
	}
	return 0.0, errors.New("order-book as 0 buy or sell limits")
}

/*
Returns the order-book spread, it needs to
have at least one buy and sell limit to work.
*/
func (ob *OrderBook) getSpread() (float64, error) {
	if len(ob.BuyLimits) > 0 && len(ob.SellLimits) > 0 {
		return (ob.SellLimits[0].Price - ob.BuyLimits[0].Price), nil
	} else {
		return 0.0, nil
	}
}

/*
Returns the market buy or sell limits total Volume.
*/
func (ob *OrderBook) getTotalVolume(buyLimits bool) float64 {
	totalVol := 0.0
	if buyLimits {
		for _, limit := range ob.BuyLimits {
			totalVol += limit.Volume
		}
	} else {
		for _, limit := range ob.SellLimits {
			totalVol += limit.Volume
		}
	}
	return totalVol
}

// Sorting Limits

type byLowestPrice []*Limit

func (limits byLowestPrice) Len() int {
	return len(limits)
}

func (limits byLowestPrice) Swap(i, j int) {
	limits[i], limits[j] = limits[j], limits[i]
}

func (limits byLowestPrice) Less(i, j int) bool {
	return limits[i].Price < limits[j].Price
}

//

type byHighestPrice []*Limit

func (limits byHighestPrice) Len() int {
	return len(limits)
}

func (limits byHighestPrice) Swap(i, j int) {
	limits[i], limits[j] = limits[j], limits[i]
}

func (limits byHighestPrice) Less(i, j int) bool {
	return limits[i].Price > limits[j].Price
}

// Printing

func (ob *OrderBook) String() string {
	return fmt.Sprintf("\nORDER BOOK\nBUYS %+v \nSELLS %+v \nBMAP %+v \nSMAP %+v \nPrice %.2f\n", ob.BuyLimits, ob.SellLimits, ob.buyLimitsMap, ob.sellLimitsMap, ob.Price)
}

func (o *Order) String() string {
	return fmt.Sprintf("[QTY %.2f]", o.qty)
}

func (l *Limit) String() string {
	return fmt.Sprintf("\n{Price %.2f VOL %.2f ORDERS %s}", l.Price, l.Volume, l.getOrders())
}

func (l *Limit) getOrders() string {
	temp := l.orders.front
	s := ""
	for temp != nil {
		s += fmt.Sprintf("%+v ", temp.String())
		temp = temp.next
	}
	return s
}
