package handler

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"checkoutservice/money"
	pb "checkoutservice/proto"
)

// Log
var (
	buf    bytes.Buffer
	logger = log.New(&buf, "logger: ", log.LstdFlags)
)

type CheckoutService struct {
	CartService           pb.CartServiceClient
	CurrencyService       pb.CurrencyServiceClient
	EmailService          pb.EmailServiceClient
	PaymentService        pb.PaymentServiceClient
	ProductCatalogService pb.ProductCatalogServiceClient
	ShippingService       pb.ShippingServiceClient
}

// Place Order
func (s *CheckoutService) PlaceOrder(ctx context.Context, in *pb.PlaceOrderRequest) (out *pb.PlaceOrderResponse, e error) {
	logger.Printf("[PlaceOrder] user_id=%q user_currency=%q", in.UserId, in.UserCurrency)

	out = new(pb.PlaceOrderResponse)
	orderID, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
		return nil, status.Errorf(codes.Internal, "Failed to generate Order Id")
	}

	prep, err := s.prepareOrderItemsAndShippingQuoteFromCart(ctx, in.UserId, in.UserCurrency, in.Address)
	if err != nil {
		log.Fatal(err)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	total := &pb.Money{CurrencyCode: in.UserCurrency, Units: 0, Nanos: 0}
	total = money.Must(money.Sum(total, prep.shippingCostLocalized))
	for _, it := range prep.orderItems {
		multPrice := money.MultiplySlow(it.Cost, uint32(it.GetItem().GetQuantity()))
		total = money.Must(money.Sum(total, multPrice))
	}

	txID, err := s.chargeCard(ctx, total, in.CreditCard)
	if err != nil {
		log.Fatal(err)
		return nil, status.Errorf(codes.Internal, "Failed to change card: %+v", err)
	}
	logger.Printf("Payment (transaction_id: %s)", txID)

	shippingTrackingID, err := s.shipOrder(ctx, in.Address, prep.cartItems)
	if err != nil {
		log.Fatal(err)
		return nil, status.Errorf(codes.Unavailable, "Shipping Error: %+v", err)
	}

	if err := s.emptyUserCart(ctx, in.UserId); err != nil {
		logger.Printf("Failed to empty user's cart: %s: %+v", in.UserId, err)
		log.Fatal(err)
	}

	orderResult := &pb.OrderResult{
		OrderId:            orderID.String(),
		ShippingTrackingId: shippingTrackingID,
		ShippingCost:       prep.shippingCostLocalized,
		ShippingAddress:    in.Address,
		Items:              prep.orderItems,
	}

	if err := s.sendOrderConfirmation(ctx, in.Email, orderResult); err != nil {
		logger.Printf("Failed to send order confirmation message: %q: %+v", in.Email, err)
		log.Fatal(err)
	} else {
		logger.Printf("Order Confirmation Email Sent Successfully: %q", in.Email)
		log.Printf(in.Email)
	}
	out.Order = orderResult
	return out, nil
}

// order preparation
type orderPrep struct {
	orderItems            []*pb.OrderItem
	cartItems             []*pb.CartItem
	shippingCostLocalized *pb.Money
}

// Preparation of orders and shipping
func (s *CheckoutService) prepareOrderItemsAndShippingQuoteFromCart(ctx context.Context, userID, userCurrency string, address *pb.Address) (orderPrep, error) {
	var out orderPrep

	cartItems, err := s.getUserCart(ctx, userID)
	if err != nil {
		return out, fmt.Errorf("get cart failed: %+v", err)
	}
	orderItems, err := s.prepOrderItems(ctx, cartItems, userCurrency)
	if err != nil {
		return out, fmt.Errorf("prepare order failed: %+v", err)
	}
	shippingUSD, err := s.quoteShipping(ctx, address, cartItems)
	if err != nil {
		return out, fmt.Errorf("quote shipping failed: %+v", err)
	}
	shippingPrice, err := s.convertCurrency(ctx, shippingUSD, userCurrency)
	if err != nil {
		return out, fmt.Errorf("failed currency conversion: %+v", err)
	}

	out.shippingCostLocalized = shippingPrice
	out.cartItems = cartItems
	out.orderItems = orderItems
	return out, nil
}

// quote shipping
func (s *CheckoutService) quoteShipping(ctx context.Context, address *pb.Address, items []*pb.CartItem) (*pb.Money, error) {
	shippingQuote, err := s.ShippingService.GetQuote(ctx, &pb.GetQuoteRequest{
		Address: address,
		Items:   items,
	})
	if err != nil {
		return nil, fmt.Errorf("shipping quote failed: %+v", err)
	}
	return shippingQuote.GetCostUsd(), nil
}

// get user cart
func (s *CheckoutService) getUserCart(ctx context.Context, userID string) ([]*pb.CartItem, error) {
	cart, err := s.CartService.GetCart(ctx, &pb.GetCartRequest{UserId: userID})
	if err != nil {
		return nil, fmt.Errorf("get user cart failed: %+v", err)
	}
	return cart.GetItems(), nil
}

// empty user cart
func (s *CheckoutService) emptyUserCart(ctx context.Context, userID string) error {
	if _, err := s.CartService.EmptyCart(ctx, &pb.EmptyCartRequest{UserId: userID}); err != nil {
		return fmt.Errorf("empty user cart failed: %+v", err)
	}
	return nil
}

// prep order items
func (s *CheckoutService) prepOrderItems(ctx context.Context, items []*pb.CartItem, userCurrency string) ([]*pb.OrderItem, error) {
	out := make([]*pb.OrderItem, len(items))
	for i, item := range items {
		product, err := s.ProductCatalogService.GetProduct(ctx, &pb.GetProductRequest{Id: item.GetProductId()})
		if err != nil {
			return nil, fmt.Errorf("get prouduct failed #%q", item.GetProductId())
		}
		price, err := s.convertCurrency(ctx, product.GetPriceUsd(), userCurrency)
		if err != nil {
			return nil, fmt.Errorf("currency conversion failed %q to %s", item.GetProductId(), userCurrency)
		}
		out[i] = &pb.OrderItem{Item: item, Cost: price}
	}
	return out, nil
}

// convert currency
func (s *CheckoutService) convertCurrency(ctx context.Context, from *pb.Money, toCurrency string) (*pb.Money, error) {
	result, err := s.CurrencyService.Convert(context.TODO(), &pb.CurrencyConversionRequest{
		From:   from,
		ToCode: toCurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("currency conversion failed: %+v", err)
	}
	return result, err
}

// charge card
func (s *CheckoutService) chargeCard(ctx context.Context, amount *pb.Money, paymentInfo *pb.CreditCardInfo) (string, error) {
	paymentResp, err := s.PaymentService.Charge(ctx, &pb.ChargeRequest{
		Amount:     amount,
		CreditCard: paymentInfo,
	})
	if err != nil {
		return "", fmt.Errorf("cannot replace card: %+v", err)
	}
	return paymentResp.GetTransactionId(), nil
}

// send order confirmation
func (s *CheckoutService) sendOrderConfirmation(ctx context.Context, email string, order *pb.OrderResult) error {
	_, err := s.EmailService.SendOrderConfirmation(ctx, &pb.SendOrderConfirmationRequest{
		Email: email,
		Order: order,
	})
	return err
}

// ship order
func (s *CheckoutService) shipOrder(ctx context.Context, address *pb.Address, items []*pb.CartItem) (string, error) {
	resp, err := s.ShippingService.ShipOrder(ctx, &pb.ShipOrderRequest{
		Address: address,
		Items:   items,
	})
	if err != nil {
		return "", fmt.Errorf("shipping failed: %+v", err)
	}
	return resp.GetTrackingId(), nil
}
