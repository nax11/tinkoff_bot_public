package profile

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/nax11/tinkoff_bot_public/api"
	investapi "github.com/nax11/tinkoff_bot_public/proto"
)

type Provider interface {
	GetOrCreateOpenedAccount(ctx context.Context, amount int) (accountID string, err error)
	CheckFigiOperations(accountID, figi string) error
}

type impl struct {
	client *api.Client
}

func Instance(client *api.Client) Provider {
	return &impl{client: client}
}

func (i *impl) GetOrCreateOpenedAccount(ctx context.Context, amount int) (accountID string, err error) {
	accounts, err := i.client.SandboxGetAccounts(ctx)
	if err != nil {
		return "", errors.Wrap(err, "fail get sandbox accounts")
	}

	for _, acc := range accounts {
		if acc.GetStatus() == investapi.AccountStatus_ACCOUNT_STATUS_OPEN {
			if acc.Id != "" {
				return acc.Id, nil
			}
		}
	}

	accountID, err = i.client.SandboxOpenAccount(ctx, 3000)
	if err != nil {
		return "", errors.Wrap(err, "fail create sandbox account")
	}
	return accountID, nil
}

func (i *impl) CheckFigiOperations(accountID, figi string) error {
	pos, err := i.client.GetOpenPosition(context.TODO(), accountID, figi)
	if err != nil {
		return errors.Wrap(err, "fail GetOpenPosition")
	}
	logrus.WithField("open_position", pos).Info("GetOpenPosition sent")

	ops, err := i.client.SandboxGetOperations(context.TODO(), accountID, figi, investapi.OperationState_OPERATION_STATE_EXECUTED)
	if err != nil {
		return errors.Wrap(err, "fail GetOperations")
	}
	logrus.WithField("operations", ops).Info("GetOperations sent")

	activeOrder, err := i.client.GetActiveOrder(context.TODO(), accountID, figi)
	if err != nil {
		return errors.Wrap(err, "fail GetActiveOrder")
	}
	logrus.WithField("active_order", activeOrder).Info("GetActiveOrder sent")
	return nil
}
