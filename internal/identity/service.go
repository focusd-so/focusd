package identity

import (
	"context"

	"connectrpc.com/connect"
	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/gen/api/v1/apiv1connect"
)

type Service struct {
	apiClient apiv1connect.ApiServiceClient
}

func NewService(apiClient apiv1connect.ApiServiceClient) *Service {
	return &Service{
		apiClient: apiClient,
	}
}

func (s *Service) GetToken(ctx context.Context) (string, error) {
	return GetToken(ctx)
}

func (s *Service) GetAccountTier(ctx context.Context) (apiv1.DeviceHandshakeResponse_AccountTier, error) {
	return accountTier, nil
}

func (s *Service) CheckoutLink(ctx context.Context) (string, error) {
	product := apiv1.CheckoutProduct_CHECKOUT_PRODUCT_BASIC

	res, err := s.apiClient.CheckoutGetLink(ctx, &connect.Request[apiv1.CheckoutGetLinkRequest]{Msg: &apiv1.CheckoutGetLinkRequest{
		Product: product,
	}})
	if err != nil {
		return "", err
	}

	return res.Msg.Link, nil
}
