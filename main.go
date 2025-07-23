package main

import "fmt"

func main() {
	// 創建 BTC/USDT 訂單簿
	ob := NewOrderBook("BTCUSDT")

	// 創建一些測試訂單
	buyOrder1 := &Order{
		ID:       "buy1",
		Side:     Bid,
		Type:     Limit,
		Price:    50000,
		Quantity: 1.0,
	}

	// sellOrder1 := &Order{
	// 	ID:       "sell1",
	// 	Side:     Ask,
	// 	Type:     Limit,
	// 	Price:    50100,
	// 	Quantity: 0.5,
	// }

	// 下單
	fmt.Println("=== 下限價買單 ===")
	trades1 := ob.PlaceOrder(buyOrder1)
	fmt.Printf("成交記錄數量: %d\n", len(trades1))
	for _, trade := range trades1 {
		fmt.Printf("成交: 價格=%f, 數量=%f\n", trade.Price, trade.Quantity)
	}

	// fmt.Println("\n=== 下限價賣單 ===")
	// trades2 := ob.PlaceOrder(sellOrder1)
	// fmt.Printf("成交記錄數量: %d\n", len(trades2))
	// for _, trade := range trades2 {
	// 	fmt.Printf("成交: 價格=%f, 數量=%f\n", trade.Price, trade.Quantity)
	// }

	// 下市價單
	// marketBuy := &Order{
	// 	ID:       "market1",
	// 	Side:     Bid,
	// 	Type:     Market,
	// 	Quantity: 0.8,
	// }
	// fmt.Println("\n=== 下市價買單 ===")
	// trades4 := ob.PlaceOrder(marketBuy)
	// fmt.Printf("成交記錄數量: %d\n", len(trades4))
	// for _, trade := range trades4 {
	// 	fmt.Printf("成交: 價格=%f, 數量=%f\n", trade.Price, trade.Quantity)
	// }

}
