package handler

import (
	"context"
	pb "currencyservice/proto"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CurrencyService struct{}

// Get currency
func (s *CurrencyService) GetSupportedCurrencies(ctx context.Context, in *pb.Empty) (out *pb.GetSupportedCurrenciesResponse, e error) {
	data, err := os.ReadFile("data/currency_conversion.json")

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to load currency data: %+v", err)
	}

	currencies := make(map[string]float32)
	if err := json.Unmarshal(data, &currencies); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to parse currency data: %+v", err)
	}

	fmt.Printf("currency: %v\n", currencies)

	out = new(pb.GetSupportedCurrenciesResponse)

	out.CurrencyCodes = make([]string, 0, len(currencies))

	for k := range currencies {
		out.CurrencyCodes = append(out.CurrencyCodes, k)
	}
	return out, nil
}

// convert
func (s *CurrencyService) Convert(ctx context.Context, in *pb.CurrencyConversionRequest) (out *pb.Money, e error) {
	data, err := os.ReadFile("data/currency_conversion.json")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to load currency data: %+v", err)
	}
	currencies := make(map[string]float64)
	if err := json.Unmarshal(data, &currencies); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to parse currency data: %+v", err)
	}
	fromCurrency, found := currencies[in.From.CurrencyCode]
	if !found {
		return nil, status.Errorf(codes.InvalidArgument, "Unsupported currency: %s", in.From.CurrencyCode)
	}
	toCurrency, found := currencies[in.ToCode]
	if !found {
		return nil, status.Errorf(codes.InvalidArgument, "Unsupported currency: %s", in.ToCode)
	}

	out = new(pb.Money)
	out.CurrencyCode = in.ToCode
	total := int64(math.Floor(float64(in.From.Units*10^9+int64(in.From.Nanos)) / fromCurrency * toCurrency))
	out.Units = total / 1e9
	out.Nanos = int32(total % 1e9)
	return out, nil
}
