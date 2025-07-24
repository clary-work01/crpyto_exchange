package main

import (
	"encoding/json"
	"fmt"

	"github.com/clary-work01/crypto_exchange/orderbook"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	ex := NewExchange()

	e.POST("/order", ex.handlePlaceOrder)

	e.Start(":3000")

}

type Exchange struct {
	OrderBooks map[orderbook.Symbol]*orderbook.OrderBook
}

func NewExchange() *Exchange {
	orderbooks := make(map[orderbook.Symbol]*orderbook.OrderBook)
	orderbooks[orderbook.ETH] = orderbook.NewOrderBook(orderbook.ETH)

	return &Exchange{
		OrderBooks: orderbooks,
	}
}

type PlaceOrderRequest struct {
	Symbol   orderbook.Symbol
	Type     orderbook.OrderType
	Side     orderbook.OrderSide
	Price    float64
	Quantity float64
}

func (ex *Exchange) handlePlaceOrder(ctx echo.Context) error {
	var req PlaceOrderRequest

	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return err
	}

	fmt.Println(req.Symbol)
	buyOrder1 := &orderbook.Order{
		ID:       "buy1",
		Symbol:   req.Symbol,
		Side:     req.Side,
		Type:     req.Type,
		Price:    req.Price,
		Quantity: req.Quantity,
	}

	symbol := req.Symbol
	ob := ex.OrderBooks[symbol]
	trade := ob.PlaceOrder(buyOrder1)
	fmt.Println(trade)
	return ctx.JSON(200, "order placed")
}

// func (ex *Exchange) handleGetOrderBook(ctx echo.Context) error {
// 	symbol := ctx.Param("symbol")
// 	ob, ok := ex.OrderBooks[orderbook.Symbol(symbol)]

// 	if !ok {
// 		return ctx.JSON(http.StatusBadRequest, map[string]string{"msg": "symbol not found"})
// 	}
// }
