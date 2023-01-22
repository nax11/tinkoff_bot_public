package priceband

import (
	"context"
	"time"

	"github.com/nax11/tinkoff_bot_public/api"
	investapi "github.com/nax11/tinkoff_bot_public/proto"
	"github.com/nax11/tinkoff_bot_public/strategy"
	"github.com/nax11/tinkoff_bot_public/strategy/price-band/analyzer"
	"github.com/nax11/tinkoff_bot_public/strategy/price-band/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewStrategy(client *api.Client) strategy.Strategy {
	return &priceBandImpl{
		client:        client,
		analyzer:      analyzer.NewAnalyzer(client),
		simulateSlice: make(map[int64]*investapi.HistoricCandle),
	}
}

type priceBandImpl struct {
	client        *api.Client
	analyzer      analyzer.Provider
	simulateSlice map[int64]*investapi.HistoricCandle
}

func (p priceBandImpl) Name() string {
	return "PriceBand"
}

func (p priceBandImpl) Run(ctx context.Context, params strategy.TradeParams) error {
	err := p.validate(params)
	if err != nil {
		return err
	}

	share, err := p.client.GetShare(ctx, params.Figi)
	if err != nil {
		return err
	}

	log := logrus.WithFields(logrus.Fields{
		"strategy":   p.Name(),
		"instrument": share,
	})

	log.Info("Run strategy")

	if params.SimulateDayTrade {
		err = p.simulateStrategy(ctx, params, share)
		if err != nil {
			return err
		}
		return nil
	} else {
		err = p.performStrategy(ctx, params, share)
		if err != nil {
			return err
		}
	}

	return p.Run(ctx, params)
}

func (p priceBandImpl) performStrategy(ctx context.Context, params strategy.TradeParams, share *investapi.Share) error {

	from := time.Now().Add(params.AnalyzePeriod * (-1))
	to := time.Now()
	buyPrice, sellPrice, err := p.analyzer.Analyze(ctx, share.Figi, from, to)
	if err != nil {
		return err
	}

	qty := api.CalcLotCount(params.MaxDealSum, buyPrice, share.Lot, params.OperationLots)
	if qty < 1 {
		return errors.New("available lot count is less than 1")
	}

	if ok, err := p.buy(ctx, params.AccountID, share, buyPrice, qty); err != nil || !ok {
		if err == nil {
			return errors.New("buy operation terminated")
		}
		return err
	}

	if ok, err := p.sell(ctx, params.AccountID, share, sellPrice); err != nil || !ok {
		if err == nil {
			return errors.New("sell operation terminated")
		}
		return err
	}
	return nil
}

func (p priceBandImpl) simulateStrategy(ctx context.Context, params strategy.TradeParams, share *investapi.Share) error {
	nowTime := time.Now()
	from := time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day()-1, 0, 0, 0, 0, nowTime.Location())
	to := time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), 0, 0, 0, 0, nowTime.Location())
	req := investapi.GetCandlesRequest{
		Figi:     share.Figi,
		From:     timestamppb.New(from),
		To:       timestamppb.New(to),
		Interval: investapi.CandleInterval_CANDLE_INTERVAL_5_MIN,
	}
	resp, err := p.client.MarketDataServiceClient.GetCandles(ctx, &req)
	if err != nil {
		return errors.Wrap(err, "fail get candles")
	}
	qty := share.Lot * int32(params.SimulateLotQty)

	orders := []models.SimulateOrder{}
	buySuccess := 0
	sellSuccess := 0

	queueQty := int(params.AnalyzePeriod / (time.Minute * 5))
	candles := resp.GetCandles()
	lastPrice := 0.0

	if candles != nil {
		params.ReportData.AnalyzedData = []strategy.TikCandle{}
		queue := []*investapi.HistoricCandle{}
		for _, candle := range candles {
			max, _ := api.GetPrice(candle.GetHigh())
			min, _ := api.GetPrice(candle.GetLow())

			for i, order := range orders {
				if !order.IsPurchased {
					if order.BuyPrice >= min &&
						order.BuyPrice <= max {
						orders[i].IsPurchased = true
						buySuccess++
					}
					continue
				}
				if !order.IsSold {
					if order.SellPrice >= min &&
						order.SellPrice <= max {
						orders[i].IsSold = true
						sellSuccess++
					}
					continue
				}
			}
			repData := strategy.TikCandle{
				MarketHighPrice: max,
				MarketLowPrice:  min,
			}
			if len(queue) > queueQty {
				buy, sell, err := p.analyzer.AnalyzeFromSlice(ctx, queue)
				if err != nil {
					return errors.Wrap(err, "AnalyzeFromSlice with error")
				}
				repData.CalculatedBuyPrice = buy
				repData.CalculatedSellPrice = sell

				order := models.SimulateOrder{
					BuyPrice:  buy,
					BuySum:    buy * float64(qty),
					Qty:       qty,
					SellPrice: sell,
					SellSum:   sell * float64(qty),
					Profit:    sell*float64(qty) - buy*float64(qty),
				}
				if order.Profit < 10 {
					logrus.WithFields(
						logrus.Fields{
							"buy":    buy,
							"sell":   sell,
							"qty":    qty,
							"profit": order.Profit,
						},
					).Info("Period skipped")
					continue
				}
				orders = append(orders, order)

				logrus.WithFields(
					logrus.Fields{
						"buy":    buy,
						"sell":   sell,
						"qty":    qty,
						"profit": order.Profit,
					},
				).Info("Recomended prices")

				logrus.Info("====================================")

				queue = queue[1:]
			}
			if !candle.IsComplete {
				continue
			}

			logrus.
				WithField("volume", candle.GetVolume()).
				WithField("max_prices", max).
				WithField("min_prices", min).
				Info("Added candle")

			queue = append(queue, candle)
			params.ReportData.AnalyzedData = append(params.ReportData.AnalyzedData, repData)
			lastPrice = max
		}
	}

	logrus.Info("====================================")

	logrus.Infof("buy result: success %v/%v", buySuccess, len(orders))
	logrus.Infof("sell result: success %v/%v", sellSuccess, len(orders))

	profit := 0.0
	onMarket := 0.0
	notTaken := 0.0
	lastSale := 0.0
	onMarketQty := 0.0
	for _, order := range orders {
		if order.IsSold {
			profit = profit + order.Profit
			logrus.Infof("profitable: %v", order.Profit)
			continue
		}
		if order.IsPurchased {
			onMarket = onMarket + order.BuySum
			onMarketQty = onMarketQty + float64(order.Qty)
			logrus.Infof("unprofitable: %v", order.Profit)
			continue
		}
		notTaken = notTaken + order.BuySum
	}

	lastSale = onMarketQty * lastPrice

	logrus.Infof("qty per operation: %v", qty)
	logrus.Infof("profit: %v", profit)
	logrus.Infof("still on market: %v", onMarket)
	logrus.Infof("not taken: %v", notTaken)
	logrus.Infof("sale at last price: %v", lastSale)
	logrus.Infof("profit on sale at last price: %v", lastSale-onMarket)
	return nil
}

func (p priceBandImpl) buy(ctx context.Context, accountID string, share *investapi.Share, buyPrice float64, qty int64) (ok bool, err error) {
	order, err := p.client.GetActiveOrder(ctx, accountID, share.Figi)
	if err != nil {
		return false, err
	}

	orderID := ""
	if order != nil {
		orderID = order.OrderId
	} else {
		position, err := p.client.GetOpenPosition(ctx, accountID, share.Figi)
		if err != nil {
			return false, err
		}
		if position != nil && position.GetBalance() > 0 {
			logrus.WithFields(logrus.Fields{
				"account_id": accountID,
				"position":   position,
				"figi":       share.Figi,
			}).Warn("found open position")
			return true, nil
		}

		orderID, err = p.client.SandboxBuyOrder(ctx, accountID, share.Figi, buyPrice, qty)
		if err != nil {
			return false, err
		}
	}

	return p.waitOrder(ctx, accountID, orderID)
}

func (p priceBandImpl) sell(ctx context.Context, accountID string, share *investapi.Share, sellPrice float64) (ok bool, err error) {
	order, err := p.client.GetActiveOrder(ctx, accountID, share.Figi)
	if err != nil {
		return false, err
	}
	orderID := ""
	if order != nil {
		orderID = order.OrderId
	} else {
		position, err := p.client.GetOpenPosition(ctx, accountID, share.Figi)
		if err != nil {
			return false, err
		}
		if position != nil && position.GetBalance() > 0 {
			orderID, err = p.client.SandboxSellOrder(ctx, accountID, share.Figi, sellPrice, position.GetBalance())
			if err != nil {
				return false, err
			}
		}
	}

	return p.waitOrder(ctx, accountID, orderID)
}

func (p priceBandImpl) waitOrder(ctx context.Context, accountID, orderID string) (ok bool, err error) {
	log := logrus.WithFields(logrus.Fields{
		"strategy":   p.Name(),
		"account_id": accountID,
		"order_id":   orderID,
	})
	log.Info("begin waiting operation")
	orderDone := false
	select {
	case <-time.After(10 * time.Second):
		orderDone, err = p.client.CheckOrderStatus(ctx, accountID, orderID)
		if err != nil {
			return false, err
		}
	case <-ctx.Done():
		log.Info("Strategy canceled")
		return false, nil
	}
	if orderDone {
		log.Info("operation done")
		return true, nil
	}
	return p.waitOrder(ctx, accountID, orderID)
}

func (p priceBandImpl) validate(params strategy.TradeParams) error {
	if params.MaxDealSum > params.DealLimit {
		return errors.New("DealLimit should be bigger when MaxDealSum")
	}

	if params.SimulateDayTrade && params.SimulateLotQty <= 0 {
		return errors.New("SimulateLotQty param should be bigger when zero")
	}

	return nil
}
