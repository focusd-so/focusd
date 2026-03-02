package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"connectrpc.com/connect"
	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
	"github.com/polarsource/polar-go/models/operations"
	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
)

// PolarWebhookEvent represents a Polar.sh webhook event payload.
// Polar.sh follows the Standard Webhooks specification for signature verification.
//
// Webhook Event Types:
//   - checkout.created, checkout.updated
//   - subscription.created, subscription.active, subscription.canceled, subscription.updated, subscription.revoked
//   - order.created, order.paid, order.updated, order.refunded
//   - customer.created, customer.updated, customer.deleted, customer.state_changed
//   - benefit_grant.created, benefit_grant.updated, benefit_grant.revoked
//
// Example payload for checkout.created:
//
//	{
//	  "type": "checkout.created",
//	  "data": {
//	    "id": "checkout_123",
//	    "status": "open",
//	    "customer_email": "user@example.com",
//	    "product_id": "prod_123",
//	    "metadata": {
//	      "user_id": "42"
//	    }
//	  }
//	}
//
// Example payload for subscription.active:
//
//	{
//	  "type": "subscription.active",
//	  "data": {
//	    "id": "sub_123",
//	    "status": "active",
//	    "customer_id": "cust_123",
//	    "product_id": "prod_123",
//	    "current_period_start": "2025-01-01T00:00:00Z",
//	    "current_period_end": "2025-02-01T00:00:00Z"
//	  }
//	}
//
// Example payload for order.created:
//
//	{
//	  "type": "order.created",
//	  "data": {
//	    "id": "order_123",
//	    "customer_id": "cust_123",
//	    "product_id": "prod_123",
//	    "billing_reason": "subscription_create",
//	    "amount": 999,
//	    "currency": "usd",
//	    "metadata": {
//	      "user_id": "42"
//	    }
//	  }
//	}
type PolarWebhookEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// CheckoutWebhookData represents the data for checkout.created/checkout.updated events.
type CheckoutWebhookData struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	CustomerEmail string            `json:"customer_email,omitempty"`
	CustomerID    string            `json:"customer_id,omitempty"`
	ProductID     string            `json:"product_id,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// SubscriptionWebhookData represents the data for subscription events.
type SubscriptionWebhookData struct {
	ID                 string            `json:"id"`
	Status             string            `json:"status"`
	CustomerID         string            `json:"customer_id"`
	ProductID          string            `json:"product_id,omitempty"`
	CurrentPeriodStart string            `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   string            `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd  bool              `json:"cancel_at_period_end,omitempty"`
	TrialStart         *time.Time        `json:"trial_start,omitempty"`
	TrialEnd           *time.Time        `json:"trial_end,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// OrderWebhookData represents the data for order events.
type OrderWebhookData struct {
	ID            string            `json:"id"`
	CustomerID    string            `json:"customer_id"`
	ProductID     string            `json:"product_id,omitempty"`
	BillingReason string            `json:"billing_reason,omitempty"` // purchase, subscription_create, subscription_cycle, subscription_update
	Amount        int64             `json:"amount,omitempty"`
	Currency      string            `json:"currency,omitempty"`
	Status        string            `json:"status,omitempty"` // pending, paid
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func (s *ServiceImpl) CheckoutGetLink(ctx context.Context, req *connect.Request[apiv1.CheckoutGetLinkRequest]) (*connect.Response[apiv1.CheckoutGetLinkResponse], error) {
	if req.Msg.Product == apiv1.CheckoutProduct_CHECKOUT_PRODUCT_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("product is required"))
	}

	productID, ok := s.productIDs[req.Msg.Product]
	if !ok || productID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or unconfigured product: %v", req.Msg.Product))
	}

	accessToken := os.Getenv("POLAR_ACCESS_TOKEN")
	if accessToken == "" {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("POLAR_ACCESS_TOKEN environment variable not set"))
	}

	polarOpts := []polargo.SDKOption{
		polargo.WithSecurity(accessToken),
	}

	polarServer := os.Getenv("POLAR_SERVER")
	if polarServer != "" {
		slog.Info("creating checkout link", "polar_server", polarServer)

		polarOpts = append(polarOpts, polargo.WithServer(polarServer))
	}

	polarClient := polargo.New(polarOpts...)

	claims, err := GetClaims(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get claims: %w", err))
	}

	successURL := "https://focusd.so/checkout/success/$checkoutId"

	slog.Info("creating checkout link", "product_id", productID)
	res, err := polarClient.CheckoutLinks.Create(ctx, components.CreateCheckoutLinkCreateCheckoutLinkCreateProducts(
		components.CheckoutLinkCreateProducts{
			Products: []string{productID},
			Metadata: map[string]components.CheckoutLinkCreateProductsMetadata{
				"user_id": components.CreateCheckoutLinkCreateProductsMetadataStr(strconv.FormatInt(claims.UserID, 10)),
			},
			SuccessURL: &successURL,
		},
	))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create checkout link: %w", err))
	}

	return connect.NewResponse(&apiv1.CheckoutGetLinkResponse{
		Link: res.CheckoutLink.URL,
	}), nil
}

func (s *ServiceImpl) CheckoutCustomerPortal(ctx context.Context, req *connect.Request[apiv1.CheckoutCustomerPortalRequest]) (*connect.Response[apiv1.CheckoutCustomerPortalResponse], error) {
	claims, err := GetClaims(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get claims: %w", err))
	}

	user := User{}
	if err := s.gormDB.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get user: %w", err))
	}

	if user.PolarCustomerID == "" {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user does not have a polar customer id"))
	}

	accessToken := os.Getenv("POLAR_ACCESS_TOKEN")
	if accessToken == "" {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("POLAR_ACCESS_TOKEN environment variable not set"))
	}

	polarOpts := []polargo.SDKOption{
		polargo.WithSecurity(accessToken),
	}

	polarServer := os.Getenv("POLAR_SERVER")
	if polarServer != "" {
		polarOpts = append(polarOpts, polargo.WithServer(polarServer))
	}

	polarClient := polargo.New(polarOpts...)

	slog.Info("creating customer portal session", "customer_id", user.PolarCustomerID)
	res, err := polarClient.CustomerSessions.Create(ctx, operations.CreateCustomerSessionsCreateCustomerSessionCreateCustomerSessionCustomerIDCreate(
		components.CustomerSessionCustomerIDCreate{
			CustomerID: user.PolarCustomerID,
		},
	))
	if err != nil {
		slog.Error("failed to create customer portal session", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create customer portal session: %w", err))
	}

	slog.Info("customer portal session created", "url", res.CustomerSession.CustomerPortalURL)
	return connect.NewResponse(&apiv1.CheckoutCustomerPortalResponse{
		Url: res.CustomerSession.CustomerPortalURL,
	}), nil
}

// NewPolarWebhookHandler creates an HTTP handler for Polar.sh webhooks.
// It verifies the webhook signature using the Standard Webhooks specification
// and dispatches events to the appropriate handler methods.
//
// Required environment variable:
//   - POLAR_WEBHOOK_SECRET: The webhook signing secret configured in Polar.sh dashboard
//
// Example curl request for testing (replace with actual values):
//
//	curl -X POST http://localhost:8089/api/v1/webhooks/polar \
//	  -H "Content-Type: application/json" \
//	  -H "webhook-id: msg_123" \
//	  -H "webhook-timestamp: 1706200000" \
//	  -H "webhook-signature: v1,base64signature..." \
//	  -d '{"type":"checkout.created","data":{"id":"checkout_123","status":"open"}}'
func NewPolarWebhookHandler(s *ServiceImpl) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("[POLAR_WEBHOOK] webhook request received")

		if r.Method != http.MethodPost {
			slog.Warn("[POLAR_WEBHOOK] invalid method", "method", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read the webhook secret from environment
		// Polar.sh secrets typically have a "whsec_" prefix that must be stripped
		// before passing to the standard-webhooks library which expects base64-encoded data
		webhookSecret := os.Getenv("POLAR_WEBHOOK_SECRET")
		if webhookSecret == "" {
			slog.Error("[POLAR_WEBHOOK] POLAR_WEBHOOK_SECRET environment variable not set")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("[POLAR_WEBHOOK] failed to read webhook body", "error", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// to base64
		base64Secret := base64.StdEncoding.EncodeToString([]byte(webhookSecret))

		// Verify the webhook signature using Standard Webhooks
		// Headers used: webhook-id, webhook-signature, webhook-timestamp
		wh, err := standardwebhooks.NewWebhook(base64Secret)
		if err != nil {
			slog.Error("[POLAR_WEBHOOK] failed to create webhook verifier", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err := wh.Verify(body, r.Header); err != nil {
			slog.Warn("[POLAR_WEBHOOK] webhook signature verification failed", "error", err)
			http.Error(w, "Invalid signature", http.StatusForbidden)
			return
		}

		slog.Info("[POLAR_WEBHOOK] webhook received", "event", string(body))

		var event PolarWebhookEvent
		if err := json.Unmarshal(body, &event); err != nil {
			slog.Error("[POLAR_WEBHOOK] failed to unmarshal webhook event", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		slog.Info("[POLAR_WEBHOOK] handling webhook event", "type", event.Type)

		if err := s.handlePolarWebhookEvent(r.Context(), event); err != nil {
			slog.Error("[POLAR_WEBHOOK] failed to handle webhook event", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		slog.Info("[POLAR_WEBHOOK] webhook processed successfully")

		// Return 202 Accepted as per Polar.sh recommendation
		w.WriteHeader(http.StatusAccepted)
	}
}

// handlePolarWebhookEvent dispatches the webhook event to the appropriate handler.
func (s *ServiceImpl) handlePolarWebhookEvent(ctx context.Context, event PolarWebhookEvent) error {
	switch event.Type {
	// Subscription events
	case "subscription.created":
		return s.handleSubscriptionCreated(ctx, event.Data)
	case "subscription.updated":
		return s.handleSubscriptionUpdated(ctx, event.Data)
	default:
		slog.Info("unhandled webhook event type", "type", event.Type)
		return nil // Don't error on unknown events
	}
}

func (s *ServiceImpl) handleSubscriptionCreated(ctx context.Context, data json.RawMessage) error {
	var subscription SubscriptionWebhookData
	if err := json.Unmarshal(data, &subscription); err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}

	slog.Info("Subscription created", "subscription", subscription)

	user, err := s.userFromSubscription(subscription)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.PolarCustomerID = subscription.CustomerID
	user.PolarSubscriptionID = subscription.ID

	switch subscription.Status {
	case "trialing":
		user.Tier = string(TierTrial)
	case "active":
		user.Tier = string(TierBasic)
	case "past_due":
		user.Tier = string(TierFree)
	}

	user.TierChangedAt = time.Now().Unix()

	if err := s.gormDB.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	slog.Info("subscription created", "subscription_id", subscription.ID, "customer_id", subscription.CustomerID)
	return nil
}

func (s *ServiceImpl) handleSubscriptionUpdated(ctx context.Context, data json.RawMessage) error {
	var subscription SubscriptionWebhookData
	if err := json.Unmarshal(data, &subscription); err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}

	user, err := s.userFromSubscription(subscription)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.PolarPastDue = subscription.Status == "past_due"

	if subscription.Status == "unpaid" {
		userID, err := strconv.ParseInt(subscription.Metadata["user_id"], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse user id: %w", err)
		}

		user := User{}
		if err := s.gormDB.Where("id = ?", userID).First(&user).Error; err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		user.Tier = string(TierFree)
		user.TierChangedAt = time.Now().Unix()
		if err := s.gormDB.Save(&user).Error; err != nil {
			return fmt.Errorf("failed to save user: %w", err)
		}
	}

	return nil
}

func (s *ServiceImpl) userFromSubscription(subscription SubscriptionWebhookData) (User, error) {
	userID, err := strconv.ParseInt(subscription.Metadata["user_id"], 10, 64)
	if err != nil {
		return User{}, fmt.Errorf("failed to parse user id: %w", err)
	}

	user := User{}
	if err := s.gormDB.Where("id = ?", userID).First(&user).Error; err != nil {
		return User{}, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
