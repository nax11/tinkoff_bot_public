package main

import (
	"context"
	"time"

	"github.com/nax11/tinkoff_bot_public/api"
	"github.com/nax11/tinkoff_bot_public/profile"
	investapi "github.com/nax11/tinkoff_bot_public/proto"
	"github.com/nax11/tinkoff_bot_public/strategy"
	priceband "github.com/nax11/tinkoff_bot_public/strategy/price-band"
	uirender "github.com/nax11/tinkoff_bot_public/ui-render"
	"github.com/sirupsen/logrus"
)

var FigiMap = map[string]string{
	"SBER":  "BBG004730N88",
	"SBERP": "BBG0047315Y7",
	"M":     "BBG000C46HM9",
	"SAVE":  "BBG000BF6RQ9",
	"MGNT":  "BBG004RVFCY3",
	"DSKY":  "BBG000BN56Q9",
	"MAGN":  "BBG004S68507", // Магнитогорский металлургический комбинат
	"NLMK":  "BBG004S681B4",
	"GMKN":  "BBG004731489",
}

var AvailableStartegy strategy.StartegyMap = strategy.StartegyMap{
	"band": priceband.NewStrategy,
}

func main() {
	client, err := api.NewClient("Put token here")
	if err != nil {
		panic("")
	}

	ctx := context.Background()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	clientProfile := profile.Instance(client)
	accountID, err := clientProfile.GetOrCreateOpenedAccount(ctx, 3000)
	if err != nil {
		logrus.WithError(err).Error("can't get account")
		return
	}

	strategyProvider, ok := AvailableStartegy["band"]
	if !ok {
		logrus.Error("can't find selected startegy")
		return
	}

	figi, ok := FigiMap["SBER"]
	if !ok {
		logrus.Error("can't find selected figi from map")
		return
	}

	strategyOperation := strategyProvider(client)
	params := strategy.TradeParams{
		AccountID:        accountID,
		Figi:             figi,
		OperationLots:    10,
		MaxDealSum:       2000,
		DealLimit:        3000,
		Interval:         investapi.CandleInterval_CANDLE_INTERVAL_5_MIN,
		DealPeriod:       time.Minute * 30,
		AnalyzePeriod:    time.Minute * 20,
		SimulateDayTrade: true,
		SimulateLotQty:   10,
		ReportData:       &strategy.ReportParams{},
	}

	err = clientProfile.CheckFigiOperations(params.AccountID, params.Figi)
	if err != nil {
		logrus.WithError(err).Error("fail check figi operations")
		return
	}

	err = strategyOperation.Run(ctx, params)
	if err != nil {
		logrus.WithError(err).Error("strategyOperation complete with error")
	}

	if params.SimulateDayTrade {
		//run UI with market charh on http://localhost:8080/
		uirender.RunUI(*params.ReportData)
	}
}
