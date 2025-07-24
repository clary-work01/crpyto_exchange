package orderbook

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
)

// 鏈類型
type Symbol string

const (
	ETH Symbol = "ETH"
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

// 訂單狀態
type OrderStatus int

const (
	Pending OrderStatus = iota
	Filled
	Partial
	Cancelled
)

// 訂單
type Order struct {
	ID             string
	Symbol         Symbol
	Side           OrderSide
	Type           OrderType
	Status         OrderStatus
	Price          float64
	Quantity       float64
	FilledQuantity float64 // 已成交數量
	Timestamp      time.Time
}

// Remaining 返回剩餘未成交數量
func (o *Order) Remaining() float64 {
	return o.Quantity - o.FilledQuantity
}
func (o *Order) IsFilled() bool {
	return o.FilledQuantity >= o.Quantity
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
	return len(p.Orders) == 0 || p.Quantity <= 0
}

// AddOrder 添加訂單到價格層級
func (pl *PriceLevel) AddOrder(order *Order) {
	pl.Orders = append(pl.Orders, order)
	pl.Quantity += order.Remaining()
}

// 【修正】移除已成交的訂單並更新數量
func (pl *PriceLevel) RemoveFilledOrders() {
	newOrders := make([]*Order, 0)
	newQuantity := 0.0

	for _, order := range pl.Orders {
		if !order.IsFilled() {
			newOrders = append(newOrders, order)
			newQuantity += order.Remaining()
		}
	}

	pl.Orders = newOrders
	pl.Quantity = newQuantity
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
	Symbol         Symbol
	Bids           *BidHeap
	Asks           *AskHeap
	BidLevels      map[float64]*PriceLevel
	AskLevels      map[float64]*PriceLevel
	UnFilledOrders map[string]*Order
	mutex          sync.RWMutex
	Trades         []*Trade
}

func NewOrderBook(symbol Symbol) *OrderBook {
	bidHeap := &BidHeap{}
	askHeap := &AskHeap{}
	heap.Init(bidHeap)
	heap.Init(askHeap)

	return &OrderBook{
		Symbol:         symbol,
		Bids:           bidHeap,
		Asks:           askHeap,
		BidLevels:      make(map[float64]*PriceLevel),
		AskLevels:      make(map[float64]*PriceLevel),
		UnFilledOrders: make(map[string]*Order),
		Trades:         make([]*Trade, 0),
	}
}

// 下單
func (ob *OrderBook) PlaceOrder(o *Order) []*Trade {
	o.Status = Pending
	o.Timestamp = time.Now()

	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	if o.Type == Limit {
		return ob.processLimitOrder(o)
	} else {
		return ob.processMarketOrder(o)
	}
}

// 處理限價單
func (ob *OrderBook) processLimitOrder(o *Order) []*Trade {
	trades := make([]*Trade, 0)

	if o.Side == Bid {
		// 買單，先嘗試與賣單撮合
		for o.Remaining() > 0 && ob.Asks.Len() > 0 {
			bestAsk := ob.Asks.Peek()

			if bestAsk.isEmpty() {
				heap.Pop(ob.Asks)
				delete(ob.AskLevels, bestAsk.Price)
				continue
			}

			if o.Price >= bestAsk.Price {
				// 只有當買價 >= 賣價時才能撮合
				trade := ob.matchOrders(o, bestAsk.Orders[0], bestAsk.Price)

				if trade != nil {
					trades = append(trades, trade)
					ob.Trades = append(ob.Trades, trade)
				}
				// 撮合後清理已成交訂單並更新heap
				ob.cleanupPriceLevel(bestAsk, false)
			} else {
				// 價格不匹配，停止撮合
				break
			}
		}

		// 如果還有剩餘，加入買單簿
		if o.Remaining() > 0 {
			ob.AddBidToOrderBook(o)
		}
	} else {
		// 賣單，先嘗試與買單撮合
		for o.Remaining() > 0 && ob.Bids.Len() > 0 {
			bestBid := ob.Bids.Peek()

			if bestBid.isEmpty() {
				heap.Pop(ob.Bids)
				delete(ob.BidLevels, bestBid.Price)
				continue
			}

			if o.Price <= bestBid.Price {
				// 只有當買價 >= 賣價時才能撮合
				trade := ob.matchOrders(bestBid.Orders[0], o, bestBid.Price)

				if trade != nil {
					trades = append(trades, trade)
					// 【修正】將成交記錄添加到訂單簿
					ob.Trades = append(ob.Trades, trade)
				}
				// 撮合後清理已成交訂單並更新heap
				ob.cleanupPriceLevel(bestBid, true)
			} else {
				// 價格不匹配，停止撮合
				break
			}
		}

		// 如果還有剩餘，加入賣單簿
		if o.Remaining() > 0 {
			ob.AddAskToOrderBook(o)
		}
	}
	return trades
}

// 處理市價單
func (ob *OrderBook) processMarketOrder(o *Order) []*Trade {
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
					// 將成交記錄添加到訂單簿
					ob.Trades = append(ob.Trades, trade)
				}
			}
			// 撮合後清理已成交訂單並更新heap
			ob.cleanupPriceLevel(bestAsk, false)
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
					// 將成交記錄添加到訂單簿
					ob.Trades = append(ob.Trades, trade)
				}

			}
			// 撮合後清理已成交訂單並更新heap
			ob.cleanupPriceLevel(bestBid, false)
		}
	}

	// 市價單如果沒有完全成交，剩餘部分取消
	if o.Remaining() > 0 {
		o.Status = Cancelled
	}

	return trades
}

// 撮合兩個訂單
func (ob *OrderBook) matchOrders(buyOrder, sellOrder *Order, price float64) *Trade {
	quantity := min(buyOrder.Remaining(), sellOrder.Remaining())

	buyOrder.FilledQuantity += quantity
	sellOrder.FilledQuantity += quantity

	// 更新訂單狀態
	if buyOrder.IsFilled() {
		buyOrder.Status = Filled
		delete(ob.UnFilledOrders, buyOrder.ID)
	} else {
		buyOrder.Status = Partial
	}
	if sellOrder.IsFilled() {
		sellOrder.Status = Filled
		delete(ob.UnFilledOrders, sellOrder.ID)
	} else {
		sellOrder.Status = Partial
	}

	// 創建成交記錄
	trade := &Trade{
		ID:          GenerateTradeID(),
		BuyOrderId:  buyOrder.ID,
		SellOrderId: sellOrder.ID,
		Price:       price,
		Quantity:    quantity,
		Timestamp:   time.Now(),
	}

	return trade
}

func (ob *OrderBook) AddBidToOrderBook(o *Order) {
	ob.UnFilledOrders[o.ID] = o

	if level, exists := ob.BidLevels[o.Price]; exists {
		level.AddOrder(o)
	} else {
		newLevel := &PriceLevel{
			Price:    o.Price,
			Orders:   []*Order{o},
			Quantity: o.Remaining(),
		}
		ob.BidLevels[o.Price] = newLevel
		heap.Push(ob.Bids, newLevel)
	}
}

func (ob *OrderBook) AddAskToOrderBook(o *Order) {
	ob.UnFilledOrders[o.ID] = o

	if level, exists := ob.AskLevels[o.Price]; exists {
		level.AddOrder(o)
	} else {
		newLevel := &PriceLevel{
			Price:    o.Price,
			Orders:   []*Order{o},
			Quantity: o.Remaining(),
		}
		ob.AskLevels[o.Price] = newLevel
		heap.Push(ob.Asks, newLevel)
	}
}

// 【新增】清理價格層級中的已成交訂單
func (ob *OrderBook) cleanupPriceLevel(level *PriceLevel, isBid bool) {
	level.RemoveFilledOrders()

	if level.isEmpty() {
		// 移除空的價格層級
		if isBid {
			heap.Pop(ob.Bids)
			delete(ob.BidLevels, level.Price)
		} else {
			heap.Pop(ob.Asks)
			delete(ob.AskLevels, level.Price)
		}
	}
}

// 【新增】取消訂單
func (ob *OrderBook) CancelOrder(orderID string) bool {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	order, exists := ob.UnFilledOrders[orderID]
	if !exists {
		return false
	}

	order.Status = Cancelled
	delete(ob.UnFilledOrders, orderID)

	// 從價格層級中移除該訂單
	var level *PriceLevel
	var isBid bool

	if order.Side == Bid {
		level = ob.BidLevels[order.Price]
		isBid = true
	} else {
		level = ob.AskLevels[order.Price]
		isBid = false
	}

	if level != nil {
		// 移除訂單
		newOrders := make([]*Order, 0)
		for _, o := range level.Orders {
			if o.ID != orderID {
				newOrders = append(newOrders, o)
			}
		}
		level.Orders = newOrders
		level.Quantity = 0
		for _, o := range newOrders {
			level.Quantity += o.Remaining()
		}

		ob.cleanupPriceLevel(level, isBid)
	}

	return true
}

// 【新增】獲取最佳買賣價
func (ob *OrderBook) GetBestBidAsk() (bestBid, bestAsk float64, ok bool) {
	ob.mutex.RLock()
	defer ob.mutex.RUnlock()

	if ob.Bids.Len() > 0 {
		bestBid = ob.Bids.Peek().Price
		ok = true
	}

	if ob.Asks.Len() > 0 {
		bestAsk = ob.Asks.Peek().Price
		ok = true
	}

	return
}

// 【新增】獲取市場深度
func (ob *OrderBook) GetDepth(levels int) (bids, asks []PriceLevel) {
	ob.mutex.RLock()
	defer ob.mutex.RUnlock()

	// 獲取買單深度
	bidCount := 0
	for i := 0; i < ob.Bids.Len() && bidCount < levels; i++ {
		level := (*ob.Bids)[i]
		if !level.isEmpty() {
			bids = append(bids, *level)
			bidCount++
		}
	}

	// 獲取賣單深度
	askCount := 0
	for i := 0; i < ob.Asks.Len() && askCount < levels; i++ {
		level := (*ob.Asks)[i]
		if !level.isEmpty() {
			asks = append(asks, *level)
			askCount++
		}
	}

	return
}

// 生成交易ID的輔助函數
func GenerateTradeID() string {
	return fmt.Sprintf("trade_%d", time.Now().UnixNano())
}

// min 輔助函數
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// 【新增】輔助函數 - 格式化訂單信息
func (o *Order) String() string {
	return fmt.Sprintf("Order[ID=%s, %s %s, Price=%.2f, Qty=%.4f, Filled=%.4f, Status=%s]",
		o.ID, GetSideName(o.Side), GetTypeName(o.Type),
		o.Price, o.Quantity, o.FilledQuantity, GetStatusName(o.Status))
}

// 【新增】輔助函數 - 格式化成交信息
func (t *Trade) String() string {
	return fmt.Sprintf("Trade[ID=%s, Price=%.2f, Qty=%.4f, Buy=%s, Sell=%s, Time=%s]",
		t.ID, t.Price, t.Quantity, t.BuyOrderId, t.SellOrderId,
		t.Timestamp.Format("15:04:05.000"))
}
