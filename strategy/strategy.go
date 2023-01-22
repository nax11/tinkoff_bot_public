package strategy

import (
	"context"
	"time"

	"github.com/nax11/tinkoff_bot_public/api"
	investapi "github.com/nax11/tinkoff_bot_public/proto"
)

type StartegyMap map[string]func(client *api.Client) Strategy

type Strategy interface {
	Name() string
	Run(ctx context.Context, tradeParams TradeParams) error
}

type TradeParams struct {
	AccountID        string
	Figi             string
	OperationLots    int64
	MaxDealSum       float64                  //maximum amount per deal
	DealLimit        float64                  //max limit of deals
	Interval         investapi.CandleInterval //?
	AnalyzePeriod    time.Duration
	DealPeriod       time.Duration
	SimulateDayTrade bool
	SimulateLotQty   int64
	ReportData       *ReportParams
}

type ReportParams struct {
	AnalyzedData []TikCandle
}

type TikCandle struct {
	MarketHighPrice     float64
	MarketLowPrice      float64
	CalculatedSellPrice float64
	CalculatedBuyPrice  float64
}
