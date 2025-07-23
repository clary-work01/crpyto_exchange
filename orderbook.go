package main

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
)

// 訂單方向 買or賣
type OrderSide int

const (
	Bid OrderSide = iota
	Ask
)

// 訂單類型 限價or市價
type OrderType int

const (
	Limit OrderType = iota
	Market
)

// 訂單
type Order struct {
	ID             string
	Symbol         string
	Side           OrderSide
	Type           OrderType
	Price          float64
	Quantity       float64
	FilledQuantity float64 // 已成交數量
	Timestamp      time.Time
}

// Remaining 返回剩餘未成交數量
func (o *Order) Remaining() float64 {
	return o.Quantity - o.FilledQuantity
}

// 一筆成交紀錄
type Trade struct {
	ID          string
	SellOrderId string
	BuyOrderId  string
	Price       float64
	Quantity    float64
	Timestamp   time.Time
}

// 價格層級 包含某價格的所有訂單
type PriceLevel struct {
	Price    float64
	Orders   []*Order
	Quantity float64 // 該價格層級的總量
}

func (p *PriceLevel) isEmpty() bool {
	return len(p.Orders) == 0 || p.Quantity == 0
}

// 買單堆:最大堆（價格由高到低）
type BidHeap []*PriceLevel

func (h BidHeap) Len() int {
	return len(h)
}
func (h BidHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}
func (h BidHeap) Less(i, j int) bool { // 最大堆（價格由高到低）
	return h[i].Price > h[j].Price
}
func (h *BidHeap) Push(x any) {
	*h = append(*h, x.(*PriceLevel))
}
func (h *BidHeap) Pop() any {
	old := *h
	n := len(old)

	item := old[n-1]
	*h = old[0 : n-1]
	return item
}
func (h *BidHeap) Peek() *PriceLevel {
	if len(*h) == 0 {
		return nil
	}
	return (*h)[0]
}

// 賣單堆:最小堆（價格由低到高）
type AskHeap []*PriceLevel

func (h AskHeap) Len() int {
	return len(h)
}
func (h AskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}
func (h AskHeap) Less(i, j int) bool { // 最小堆（價格由低到高）
	return h[i].Price < h[j].Price
}
func (h *AskHeap) Push(x any) {
	*h = append(*h, x.(*PriceLevel))
}
func (h *AskHeap) Pop() any {
	old := *h
	n := len(old)

	item := old[n-1]
	*h = old[0 : n-1]
	return item
}
func (h *AskHeap) Peek() *PriceLevel {
	if len(*h) == 0 {
		return nil
	}
	return (*h)[0]
}

// 訂單簿
type OrderBook struct {
	Symbol    string
	Bids      *BidHeap
	Asks      *AskHeap
	BidLevels map[float64]*PriceLevel
	AskLevels map[float64]*PriceLevel
	Orders    map[string]*Order
	mutex     sync.RWMutex
	Trades    []*Trade
}

func NewOrderBook(symbol string) *OrderBook {
	bidHeap := &BidHeap{}
	askHeap := &AskHeap{}
	heap.Init(bidHeap)
	heap.Init(askHeap)

	return &OrderBook{
		Symbol:    symbol,
		Bids:      bidHeap,
		Asks:      askHeap,
		BidLevels: make(map[float64]*PriceLevel),
		AskLevels: make(map[float64]*PriceLevel),
		Orders:    make(map[string]*Order),
	}
}

// 下單
func (ob *OrderBook) PlaceOrder(o *Order) []*Trade {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	if o.Type == Limit {
		return ob.ProcessLimitOrder(o)
	} else {
		return ob.ProcessMarketOrder(o)
	}
}

// 處理限價單
func (ob *OrderBook) ProcessLimitOrder(o *Order) []*Trade {
	trades := make([]*Trade, 0)

	if o.Side == Bid {
		// 買單，與最低價賣單撮合
		for o.Remaining() > 0 && ob.Asks.Len() > 0 {
			bestAsk := ob.Asks.Peek()

			if bestAsk.isEmpty() {
				heap.Pop(ob.Asks)
				delete(ob.AskLevels, bestAsk.Price)
				continue

			} else {
				trade := ob.matchOrders(o, bestAsk.Orders[0], bestAsk.Price)
				if trade != nil {
					trades = append(trades, trade)
				}
			}
		}
	} else {
		// 賣單 ，與最高價買單撮合
		for o.Remaining() > 0 && ob.Bids.Len() > 0 {
			bestBid := ob.Bids.Peek()

			if bestBid.isEmpty() {
				heap.Pop(ob.Bids)
				delete(ob.BidLevels, bestBid.Price)
				continue
			} else {
				trade := ob.matchOrders(o, bestBid.Orders[0], bestBid.Price)
				if trade != nil {
					trades = append(trades, trade)
				}
			}
		}
	}
	return trades
}

// 處理市價單
func (ob *OrderBook) ProcessMarketOrder(o *Order) []*Trade {
	trades := make([]*Trade, 0)
	return trades
}

// 撮合兩個訂單
func (ob *OrderBook) matchOrders(buyOrder, sellOrder *Order, price float64) *Trade {
	quantity := min(buyOrder.Remaining(), sellOrder.Remaining())

	buyOrder.FilledQuantity += quantity
	sellOrder.FilledQuantity += quantity

	// 創建成交記錄
	trade := &Trade{
		ID:          fmt.Sprintf("trade_%d", time.Now().UnixNano()),
		BuyOrderId:  buyOrder.ID,
		SellOrderId: sellOrder.ID,
		Price:       price,
		Quantity:    quantity,
		Timestamp:   time.Now(),
	}

	ob.Trades = append(ob.Trades, trade)
	return trade
}

// func (ob *OrderBook) matchOrders(buyOrder, sellOrder *Order, price float64) *Trade {
// 	quantity := min(buyOrder.Remaining(), sellOrder.Remaining())

// 	buyOrder.Filled += quantity
// 	sellOrder.Filled += quantity

// 	// 更新訂單狀態
// 	if buyOrder.IsFilled() {
// 		buyOrder.Status = Filled
// 		delete(ob.Orders, buyOrder.ID)
// 	} else {
// 		buyOrder.Status = Partial
// 	}

// 	if sellOrder.IsFilled() {
// 		sellOrder.Status = Filled
// 		delete(ob.Orders, sellOrder.ID)
// 	} else {
// 		sellOrder.Status = Partial
// 	}
// }
