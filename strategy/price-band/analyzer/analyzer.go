package analyzer

import (
	"context"
	"math"
	"time"

	"github.com/nax11/tinkoff_bot_public/api"
	investapi "github.com/nax11/tinkoff_bot_public/proto"
	"github.com/nax11/tinkoff_bot_public/strategy/price-band/models"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Provider interface {
	Analyze(ctx context.Context, figi string, from, to time.Time) (buyPrice, sellPrice float64, err error)
	AnalyzeFromSlice(ctx context.Context, candles []*investapi.HistoricCandle) (buyPrice, sellPrice float64, err error)
}

func NewAnalyzer(client *api.Client) *impl {
	return &impl{
		client: client,
	}
}

type impl struct {
	client *api.Client
}

func (i impl) Analyze(ctx context.Context, figi string, from, to time.Time) (buyPrice, sellPrice float64, err error) {
	req := investapi.GetCandlesRequest{
		Figi:     figi,
		From:     timestamppb.New(from),
		To:       timestamppb.New(to),
		Interval: investapi.CandleInterval_CANDLE_INTERVAL_5_MIN,
	}
	resp, err := i.client.MarketDataServiceClient.GetCandles(ctx, &req)
	if err != nil {
		return 0, 0, errors.Wrap(err, "fail get candles")
	}
	candles := resp.GetCandles()
	if candles == nil {
		return 0, 0, errors.New("the candles are empty")
	}

	return i.AnalyzeFromSlice(ctx, candles)
}

func (i impl) AnalyzeFromSlice(ctx context.Context, candles []*investapi.HistoricCandle) (buyPrice, sellPrice float64, err error) {
	buyPrice, sellPrice, err = pricesByHistory(candles)
	if err != nil {
		return 0, 0, errors.Wrap(err, "fail get prices by history")
	}
	return buyPrice, sellPrice, nil
}

func pricesByHistory(candles []*investapi.HistoricCandle) (buyPrice, sellPrice float64, err error) {
	maxPrices := models.AverageSlice{}
	minPrices := models.AverageSlice{}

	for i, candle := range candles {
		if !candle.IsComplete {
			continue
		}
		maxPrice, err := api.GetPrice(candle.High)
		if err != nil {
			return 0, 0, errors.Wrap(err, "fail convert High price of candle")
		}
		minPrice, err := api.GetPrice(candle.Low)
		if err != nil {
			return 0, 0, errors.Wrap(err, "fail convert Low price of candle")
		}
		maxPrices = append(maxPrices, maxPrice)
		minPrices = append(minPrices, minPrice)
		if i < 3 {
			continue
		}
		maxPrices = maxPrices[1:]
		minPrices = minPrices[1:]
	}

	buyPrice, sellPrice = getBuySellPrice(minPrices, maxPrices)
	return buyPrice, sellPrice, nil
}

func getBuySellPrice(minPrices, maxPrices models.AverageSlice) (buyPrice, sellPrice float64) {
	minBy := minPrices.AveragePrice()
	maxBy := maxPrices.AveragePrice()
	delta := maxBy - minBy
	deltaP := math.Round(delta*0.01*100) / 100

	buyPrice = math.Ceil((minBy+deltaP)*100) / 100
	sellPrice = math.Floor((maxBy-deltaP)*100) / 100
	return buyPrice, sellPrice
}
