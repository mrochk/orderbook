package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrochk/exchange/orderbook"
)

type InitParams struct {
	MidPrice float64 `json:"mid_price"`
}

func handleInit(c *gin.Context) (InitParams, error) {
	var params InitParams
	return params, c.BindJSON(&params)
}

type LimitOrderParams struct {
	Type  bool    `json:"type"`
	Price float64 `json:"price"`
	Qty   float64 `json:"qty"`
}

func handleLimitOrder(c *gin.Context) (LimitOrderParams, error) {
	var params LimitOrderParams
	return params, c.BindJSON(&params)
}

type CancelOrderParams struct {
	ID    uuid.UUID `json:"id"`
	Price float64   `json:"price"`
}

func handleCancelOrder(c *gin.Context) (CancelOrderParams, error) {
	var params CancelOrderParams
	return params, c.BindJSON(&params)
}

type MarketOrderParams struct {
	Type bool    `json:"type"`
	Qty  float64 `json:"qty"`
}

func handleMarketOrder(c *gin.Context) (MarketOrderParams, error) {
	var params MarketOrderParams
	return params, c.BindJSON(&params)
}

type LimitData struct {
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
}

type Data struct {
	BuyLimitsData  []LimitData `json:"buy_limits"`
	SellLimitsData []LimitData `json:"sell_limits"`
	Price          float64     `json:"price"`
	Spread         float64     `json:"spread"`
	MidPrice       float64     `json:"midprice"`
}

func newData() Data {
	return Data{
		BuyLimitsData:  make([]LimitData, 0),
		SellLimitsData: make([]LimitData, 0),
	}
}

func New(ob *orderbook.OrderBook) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.POST("/init", func(ctx *gin.Context) {
		p, err := handleInit(ctx)
		if err == nil {
			ob.Init(p.MidPrice)
		}
	})

	router.POST("/limit_order", func(ctx *gin.Context) {
		p, err := handleLimitOrder(ctx)
		if err == nil {
			id, err := ob.PlaceLimitOrder(p.Type, p.Price, p.Qty)
			if err == nil {
				var resp struct {
					ID uuid.UUID `json:"order_id"`
				}
				resp.ID = id
				ctx.JSON(http.StatusOK, resp)
			} else {
				fmt.Println(err)
			}
		}
	})

	router.POST("/cancel_order", func(ctx *gin.Context) {
		p, err := handleCancelOrder(ctx)
		if err == nil {
			err := ob.CancelLimitOrder(p.ID, p.Price)
			if err != nil {
				fmt.Println(err)
			}
		}
	})

	router.POST("/market_order", func(ctx *gin.Context) {
		p, err := handleMarketOrder(ctx)
		if err == nil {
			err := ob.PlaceMarketOrder(p.Type, p.Qty)
			if err != nil {
				fmt.Println(err)
			}
		}
	})

	router.GET("/get_data", func(ctx *gin.Context) {
		var limitData LimitData
		d := newData()
		for _, limit := range ob.BuyLimits {
			limitData.Price = limit.Price
			limitData.Volume = limit.Volume
			d.BuyLimitsData = append(d.BuyLimitsData, limitData)
		}
		for _, limit := range ob.SellLimits {
			limitData.Price = limit.Price
			limitData.Volume = limit.Volume
			d.SellLimitsData = append(d.SellLimitsData, limitData)
		}
		ctx.JSON(200, d)
	})

	return router
}
