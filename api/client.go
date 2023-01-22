package api

import (
	"context"
	"crypto/tls"
	"fmt"

	investapi "github.com/nax11/tinkoff_bot_public/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	url = "invest-public-api.tinkoff.ru"
)

type Config struct {
	Token     string   `required:"true"`
	AccountID []string `split_words:"true"` // required in non-sandbox mode
}

func CreateStreamContext(cfg Config) context.Context {
	//investapi.Share{}
	ctx := context.TODO()

	//authHeader := fmt.Sprintf("Bearer %s", cfg.Token)
	// ctx = grpcMetadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
	// ctx = grpcMetadata.AppendToOutgoingContext(ctx, "x-tracking-id", uuid.New().String())
	//ctx = grpcMetadata.AppendToOutgoingContext(ctx, "x-app-name", AppName)

	return ctx
}

type Client struct {
	connection               *grpc.ClientConn
	InstrumentsServiceClient investapi.InstrumentsServiceClient
	//UsersServiceClient       investapi.UsersServiceClient
	MarketDataServiceClient investapi.MarketDataServiceClient
	//OperationsServiceClient  investapi.OperationsServiceClient
	//OrdersServiceClient      investapi.OrdersServiceClient
	//StopOrdersServiceClient  investapi.StopOrdersServiceClient
	sandboxClient investapi.SandboxServiceClient
}
type tokenAuth struct {
	// Token from // https://tinkoff.github.io/investAPI/grpc/#tinkoff-invest-api_1
	Token string
}

func (t tokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": "Bearer " + t.Token,
	}, nil
}

func (tokenAuth) RequireTransportSecurity() bool {
	return true
}

func NewClient(token string) (client *Client, err error) {
	return NewWithOpts(token, url, make([]grpc.DialOption, 0)...)
}

func NewWithOpts(token, endpoint string, opts ...grpc.DialOption) (client *Client, err error) {
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		ServerName: endpoint,
	})))
	opts = append(opts, grpc.WithPerRPCCredentials(tokenAuth{
		Token: token,
	}))
	conn, err := grpc.Dial(fmt.Sprintf("%s:443", endpoint), opts...)
	if err != nil {
		return
	}
	client = new(Client)
	client.connection = conn
	client.InstrumentsServiceClient = investapi.NewInstrumentsServiceClient(conn)
	//client.UsersServiceClient = investapi.NewUsersServiceClient(conn)
	client.MarketDataServiceClient = investapi.NewMarketDataServiceClient(conn)
	//client.OperationsServiceClient = investapi.NewOperationsServiceClient(conn)
	//client.OrdersServiceClient = investapi.NewOrdersServiceClient(conn)
	//client.StopOrdersServiceClient = investapi.NewStopOrdersServiceClient(conn)
	client.sandboxClient = investapi.NewSandboxServiceClient(conn)
	return
}
