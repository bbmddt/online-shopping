package handler

import (
	"bytes"
	"context"
	"fmt"
	"log"
	pb "shippingservice/proto"
)

type ShippingService struct{}

// Log
var (
	buf    bytes.Buffer
	logger = log.New(&buf, "logger: ", log.Lshortfile)
)

func (s *ShippingService) GetQuote(ctx context.Context, in *pb.GetQuoteRequest) (out *pb.GetQuoteResponse, e error) {
	logger.Print("[GetQuote] Request received")
	defer logger.Print("[GetQuote] Request completed")

	// 1. Generate quote based on the count of items
	out = new(pb.GetQuoteResponse)
	quote := CreateQuoteFromCount(0)

	// 2. Generate response
	out.CostUsd = &pb.Money{
		CurrencyCode: "USD",
		Units:        int64(quote.Dollars),
		Nanos:        int32(quote.Cents * 10000000),
	}
	return out, nil
}

func (s *ShippingService) ShipOrder(ctx context.Context, in *pb.ShipOrderRequest) (out *pb.ShipOrderResponse, e error) {
	logger.Print("[ShipOrder] Request received")
	defer logger.Print("[ShipOrder] Request completed")
	// 1. Create tracking ID
	out = new(pb.ShipOrderResponse)
	baseAddress := fmt.Sprintf("%s, %s, %s", in.Address.StreetAddress, in.Address.City, in.Address.State)
	id := CreateTrackingId(baseAddress)

	// 2. Generate response
	out.TrackingId = id
	return out, nil
}
