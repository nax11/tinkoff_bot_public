package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	investapi "github.com/nax11/tinkoff_bot_public/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (c Client) SandboxBuyOrder(ctx context.Context, accountID, figi string, buyPrice float64, qty int64) (orderID string, err error) {
	req := investapi.PostOrderRequest{
		Figi:      figi,
		Quantity:  qty,
		Price:     BuildQuotationByPrice(buyPrice),
		Direction: investapi.OrderDirection_ORDER_DIRECTION_BUY,
		AccountId: accountID,
		OrderType: investapi.OrderType_ORDER_TYPE_LIMIT,
		OrderId:   uuid.New().String(),
	}
	log := logrus.WithFields(logrus.Fields{
		"account_id": accountID,
		"figi":       figi,
		"buy_price":  buyPrice,
		"request":    fmt.Sprintf("%v", &req),
	})

	log.WithField("request", fmt.Sprintf("%v", &req)).Info("PostSandboxOrder")
	resp, err := c.sandboxClient.PostSandboxOrder(ctx, &req)
	if err != nil {
		return "", errors.Wrap(err, "error on execute buy on PostSandboxOrder")
	}
	log.WithField("response", fmt.Sprintf("%v", resp)).Info("PostSandboxOrder sent")

	if resp.ExecutionReportStatus == investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED ||
		resp.ExecutionReportStatus == investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED {
		return "", errors.New("post order with unsuccessful status")
	}

	return resp.OrderId, nil
}

func (c Client) SandboxSellOrder(ctx context.Context, accountID, figi string, sellPrice float64, qty int64) (orderID string, err error) {
	req := investapi.PostOrderRequest{
		Figi:      figi,
		Quantity:  qty,
		Price:     BuildQuotationByPrice(sellPrice),
		Direction: investapi.OrderDirection_ORDER_DIRECTION_SELL,
		AccountId: accountID,
		OrderType: investapi.OrderType_ORDER_TYPE_LIMIT,
		OrderId:   uuid.New().String(),
	}
	log := logrus.WithFields(logrus.Fields{
		"account_id": accountID,
		"figi":       figi,
		"buy_price":  sellPrice,
		"request":    fmt.Sprintf("%v", &req),
	})

	log.WithField("request", fmt.Sprintf("%v", &req)).Info("PostSandboxOrder")
	resp, err := c.sandboxClient.PostSandboxOrder(ctx, &req)
	if err != nil {
		return "", errors.Wrap(err, "error on execute buy on PostSandboxOrder")
	}
	log.WithField("response", fmt.Sprintf("%v", resp)).Info("PostSandboxOrder sent")

	if resp.ExecutionReportStatus == investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED ||
		resp.ExecutionReportStatus == investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED {
		return "", errors.New("post order with unsuccessful status")
	}

	return resp.OrderId, nil
}

func (c Client) GetShare(ctx context.Context, figi string) (share *investapi.Share, err error) {
	req := investapi.InstrumentRequest{
		IdType: investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_FIGI,
		Id:     figi,
	}
	resp, err := c.InstrumentsServiceClient.ShareBy(ctx, &req)
	if err != nil {
		return nil, errors.Wrap(err, "fail get share by figi")
	}

	share = resp.GetInstrument()

	if share == nil {
		return nil, errors.New("the share are empty")
	}
	return share, nil
}

func (c Client) GetOpenPosition(ctx context.Context, accountID, figi string) (openPosition *investapi.PositionsSecurities, err error) {
	req := investapi.PositionsRequest{
		AccountId: accountID,
	}
	resp, err := c.sandboxClient.GetSandboxPositions(ctx, &req)
	if err != nil {
		return nil, errors.Wrap(err, "fail get open positions")
	}
	if resp == nil {
		return nil, nil
	}

	for _, position := range resp.GetSecurities() {
		if position.Figi != figi {
			continue
		}
		if position.ExchangeBlocked {
			return nil, errors.New("exchange blocked by stock")
		}

		if position.GetBalance() > 0 {
			return position, nil
		}
	}
	return nil, nil
}

func (c Client) GetActiveOrder(ctx context.Context, accountID, figi string) (order *investapi.OrderState, err error) {
	req := investapi.GetOrdersRequest{
		AccountId: accountID,
	}
	resp, err := c.sandboxClient.GetSandboxOrders(ctx, &req)
	if err != nil {
		return nil, errors.Wrap(err, "fail get orders")
	}
	if resp == nil {
		return nil, nil
	}

	for _, order := range resp.GetOrders() {
		if order.Figi != figi {
			continue
		}
		return order, nil
	}
	return nil, nil
}

func (c Client) CheckOrderStatus(ctx context.Context, accountID, orderID string) (ok bool, err error) {
	stateReq := investapi.GetOrderStateRequest{
		AccountId: accountID,
		OrderId:   orderID,
	}
	stateResp, err := c.sandboxClient.GetSandboxOrderState(ctx, &stateReq)
	if err != nil {
		return false, errors.Wrapf(err, "error on execute GetSandboxOrderState for order: %v", orderID)
	}
	if stateResp.ExecutionReportStatus == investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED ||
		stateResp.ExecutionReportStatus == investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED {
		return false, errors.New("order not is success status")
	}

	logrus.WithFields(logrus.Fields{
		"account_id": accountID,
		"order_id":   orderID,
	}).Info("CheckOrderStatus")
	if stateResp.ExecutionReportStatus == investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL {
		return true, nil
	}
	return false, nil
}

func (c Client) SandboxOpenAccount(ctx context.Context, amount int) (accountID string, err error) {
	req := investapi.OpenSandboxAccountRequest{}
	resp, err := c.sandboxClient.OpenSandboxAccount(ctx, &req)
	if err != nil {
		return "", errors.Wrap(err, "fail open account")
	}
	if resp == nil {
		return "", errors.New("empty response received during open account")
	}

	return resp.GetAccountId(), nil
}

func (c Client) SandboxGetAccounts(ctx context.Context) (accounts []*investapi.Account, err error) {
	req := investapi.GetAccountsRequest{}
	resp, err := c.sandboxClient.GetSandboxAccounts(ctx, &req)
	if err != nil {
		return nil, errors.Wrap(err, "fail get accounts")
	}
	if resp == nil {
		return nil, errors.New("empty response received during get accounts")
	}
	return resp.GetAccounts(), nil
}

func (c Client) SandboxPayInAccount(ctx context.Context, accountID string, amount int64) (*investapi.MoneyValue, error) {
	req := investapi.SandboxPayInRequest{
		AccountId: accountID,
		Amount: &investapi.MoneyValue{
			Currency: "RUB",
			Units:    amount,
			Nano:     0,
		},
	}
	resp, err := c.sandboxClient.SandboxPayIn(ctx, &req)
	if err != nil {
		return nil, errors.Wrap(err, "fail pay in account")
	}
	if resp == nil {
		return nil, errors.New("empty response received during ay in account")
	}
	return resp.GetBalance(), nil
}

func (c Client) SandboxGetOperations(ctx context.Context, accountID, figi string, state investapi.OperationState) ([]*investapi.Operation, error) {
	req := investapi.OperationsRequest{
		AccountId: accountID,
		From:      timestamppb.New(time.Now().Add(time.Hour * (-24))),
		To:        timestamppb.New(time.Now()),
		State:     state,
		Figi:      figi,
	}
	resp, err := c.sandboxClient.GetSandboxOperations(ctx, &req)
	if err != nil {
		return nil, errors.Wrap(err, "fail get operations")
	}
	if resp == nil {
		return nil, nil
	}

	for _, operation := range resp.GetOperations() {
		logrus.
			WithField("operation", operation).
			WithField("account_id", accountID).
			Info("Account operation")
	}
	return resp.GetOperations(), nil
}
func BuildQuotationByPrice(price float64) *investapi.Quotation {
	priceUnits := int64(price)
	priceNano := int32((price - float64(priceUnits)) * 100)
	return &investapi.Quotation{
		Units: priceUnits,
		Nano:  priceNano,
	}
}

func GetPrice(q *investapi.Quotation) (float64, error) {
	return strconv.ParseFloat(fmt.Sprintf("%v.%v", q.GetUnits(), q.GetNano()), 64)
}

func CalcLotCount(maxDealSum, price float64, lot int32, operationLots int64) int64 {
	if maxDealSum > price*float64(operationLots*int64(lot)) {
		return operationLots
	}

	return int64(maxDealSum / (float64(lot) * price))
}
