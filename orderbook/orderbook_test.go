package orderbook

import (
	"fmt"
	"testing"
	"time"
)

// 生成測試訂單ID
func generateOrderID(prefix string, id int) string {
	return fmt.Sprintf("%s_%d_%d", prefix, id, time.Now().UnixNano()%10000)
}

// 測試基本限價單撮合
func TestLimitOrderMatching(t *testing.T) {
	// 創建BTC/USDT訂單簿
	ob := NewOrderBook("BTCUSDT")

	fmt.Println("=== 測試1: 基本限價單撮合 ===")

	// 創建一些賣單 (Ask)
	askOrders := []*Order{
		{ID: generateOrderID("ASK", 1), Symbol: "BTCUSDT", Side: Ask, Type: Limit, Price: 50100, Quantity: 0.5},
		{ID: generateOrderID("ASK", 2), Symbol: "BTCUSDT", Side: Ask, Type: Limit, Price: 50200, Quantity: 1.0},
		{ID: generateOrderID("ASK", 3), Symbol: "BTCUSDT", Side: Ask, Type: Limit, Price: 50150, Quantity: 0.8},
		{ID: generateOrderID("ASK", 4), Symbol: "BTCUSDT", Side: Ask, Type: Limit, Price: 50300, Quantity: 2.0},
	}

	// 下賣單
	for _, order := range askOrders {
		trades := ob.PlaceOrder(order)
		fmt.Printf("下單: %s\n", order.String())
		if len(trades) > 0 {
			fmt.Printf("  成交: %d筆\n", len(trades))
			for _, trade := range trades {
				fmt.Printf("    %s\n", trade.String())
			}
		}
	}

	fmt.Println("\n賣單簿狀態:")
	printOrderBook(ob)

	// 創建一些買單 (Bid)
	bidOrders := []*Order{
		{ID: generateOrderID("BID", 1), Symbol: "BTCUSDT", Side: Bid, Type: Limit, Price: 50000, Quantity: 1.0},
		{ID: generateOrderID("BID", 2), Symbol: "BTCUSDT", Side: Bid, Type: Limit, Price: 49950, Quantity: 0.5},
		{ID: generateOrderID("BID", 3), Symbol: "BTCUSDT", Side: Bid, Type: Limit, Price: 50050, Quantity: 1.5},
	}

	fmt.Println("\n下買單:")
	for _, order := range bidOrders {
		trades := ob.PlaceOrder(order)
		fmt.Printf("下單: %s\n", order.String())
		if len(trades) > 0 {
			fmt.Printf("  成交: %d筆\n", len(trades))
			for _, trade := range trades {
				fmt.Printf("    %s\n", trade.String())
			}
		}
	}

	fmt.Println("\n當前訂單簿狀態:")
	printOrderBook(ob)

	fmt.Println("\n=== 測試2: 撮合交易 ===")

	// 下一個能撮合的買單
	matchingBuyOrder := &Order{
		ID:       generateOrderID("BUY_MATCH", 1),
		Symbol:   "BTCUSDT",
		Side:     Bid,
		Type:     Limit,
		Price:    50150, // 能與50100和50150的賣單撮合
		Quantity: 1.2,
	}

	fmt.Printf("下撮合買單: %s\n", matchingBuyOrder.String())
	trades := ob.PlaceOrder(matchingBuyOrder)

	if len(trades) > 0 {
		fmt.Printf("成功撮合 %d 筆交易:\n", len(trades))
		for _, trade := range trades {
			fmt.Printf("  %s\n", trade.String())
		}
	}

	fmt.Println("\n撮合後訂單簿狀態:")
	printOrderBook(ob)

	fmt.Println("\n=== 測試3: 市價單 ===")

	// 市價買單
	marketBuyOrder := &Order{
		ID:       generateOrderID("MARKET_BUY", 1),
		Symbol:   "BTCUSDT",
		Side:     Bid,
		Type:     Market,
		Quantity: 0.3,
	}

	fmt.Printf("下市價買單: %s\n", marketBuyOrder.String())
	trades = ob.PlaceOrder(marketBuyOrder)

	if len(trades) > 0 {
		fmt.Printf("市價單成交 %d 筆:\n", len(trades))
		for _, trade := range trades {
			fmt.Printf("  %s\n", trade.String())
		}
	}

	fmt.Println("\n市價單後訂單簿狀態:")
	printOrderBook(ob)

	fmt.Println("\n=== 測試4: 取消訂單 ===")

	// 取消一個訂單
	if len(askOrders) > 0 {
		cancelOrderID := askOrders[1].ID // 取消第二個賣單
		fmt.Printf("嘗試取消訂單: %s\n", cancelOrderID)
		success := ob.CancelOrder(cancelOrderID)
		fmt.Printf("取消結果: %t\n", success)

		fmt.Println("\n取消後訂單簿狀態:")
		printOrderBook(ob)
	}

	fmt.Println("\n=== 測試5: 獲取最佳價格 ===")
	bestBid, bestAsk, ok := ob.GetBestBidAsk()
	if ok {
		fmt.Printf("最佳買價: %.2f, 最佳賣價: %.2f\n", bestBid, bestAsk)
		fmt.Printf("價差: %.2f\n", bestAsk-bestBid)
	}

	fmt.Println("\n=== 測試6: 市場深度 ===")
	bids, asks := ob.GetDepth(5)

	fmt.Println("買單深度 (前5檔):")
	for i, bid := range bids {
		fmt.Printf("  %d. 價格: %.2f, 數量: %.4f, 訂單數: %d\n",
			i+1, bid.Price, bid.Quantity, len(bid.Orders))
	}

	fmt.Println("賣單深度 (前5檔):")
	for i, ask := range asks {
		fmt.Printf("  %d. 價格: %.2f, 數量: %.4f, 訂單數: %d\n",
			i+1, ask.Price, ask.Quantity, len(ask.Orders))
	}

	fmt.Println("\n=== 測試7: 大額訂單部分撮合 ===")

	// 下一個大額買單，測試部分撮合
	largeBuyOrder := &Order{
		ID:       generateOrderID("LARGE_BUY", 1),
		Symbol:   "BTCUSDT",
		Side:     Bid,
		Type:     Limit,
		Price:    50500, // 高價，能撮合所有賣單
		Quantity: 10.0,  // 大量
	}

	fmt.Printf("下大額買單: %s\n", largeBuyOrder.String())
	trades = ob.PlaceOrder(largeBuyOrder)

	fmt.Printf("大額訂單成交 %d 筆:\n", len(trades))
	for _, trade := range trades {
		fmt.Printf("  %s\n", trade.String())
	}

	fmt.Printf("大額訂單最終狀態: %s\n", largeBuyOrder.String())

	fmt.Println("\n最終訂單簿狀態:")
	printOrderBook(ob)

	fmt.Printf("\n=== 總成交統計 ===\n")
	fmt.Printf("總成交筆數: %d\n", len(ob.Trades))
	fmt.Printf("未成交訂單數: %d\n", len(ob.UnFilledOrders))

	// 計算總成交金額
	totalVolume := 0.0
	totalAmount := 0.0
	for _, trade := range ob.Trades {
		totalVolume += trade.Quantity
		totalAmount += trade.Price * trade.Quantity
	}
	fmt.Printf("總成交量: %.4f BTC\n", totalVolume)
	fmt.Printf("總成交額: %.2f USDT\n", totalAmount)
	if totalVolume > 0 {
		fmt.Printf("平均成交價: %.2f USDT\n", totalAmount/totalVolume)
	}
}

// 輔助函數 - 打印訂單簿狀態
func printOrderBook(ob *OrderBook) {
	bestBid, bestAsk, _ := ob.GetBestBidAsk()

	fmt.Printf("訂單簿 %s:\n", ob.Symbol)
	fmt.Printf("  最佳買價: %.2f\n", bestBid)
	fmt.Printf("  最佳賣價: %.2f\n", bestAsk)
	fmt.Printf("  買單數量: %d\n", ob.Bids.Len())
	fmt.Printf("  賣單數量: %d\n", ob.Asks.Len())
	fmt.Printf("  未成交訂單: %d\n", len(ob.UnFilledOrders))

	// 顯示前3檔買賣盤
	bids, asks := ob.GetDepth(3)

	if len(asks) > 0 {
		fmt.Println("  賣盤 (前3檔):")
		for _, ask := range asks {
			fmt.Printf("    %.2f -> %.4f\n", ask.Price, ask.Quantity)
		}
	}

	if len(bids) > 0 {
		fmt.Println("  買盤 (前3檔):")
		for _, bid := range bids {
			fmt.Printf("    %.2f -> %.4f\n", bid.Price, bid.Quantity)
		}
	}

	fmt.Println()
}
