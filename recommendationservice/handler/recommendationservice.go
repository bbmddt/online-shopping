package handler

import (
	"bytes"
	"context"
	"log"
	"math/rand"

	pb "recommendationservice/proto"
)

// Log
var (
	buf    bytes.Buffer
	logger = log.New(&buf, "logger: ", log.Lshortfile)
)

type RecommendationService struct {
	ProductCatalogService pb.ProductCatalogServiceClient
}

// List recommendations
func (s *RecommendationService) ListRecommendations(ctx context.Context, in *pb.ListRecommendationsRequest) (out *pb.ListRecommendationsResponse, e error) {
	maxResponsesCount := 5
	out = new(pb.ListRecommendationsResponse)
	// search product catalog
	catalog, err := s.ProductCatalogService.ListProducts(ctx, &pb.Empty{})
	if err != nil {
		return out, err
	}
	filteredProductsIDs := make([]string, 0, len(catalog.Products))
	for _, p := range catalog.Products {
		if contains(p.Id, in.ProductIds) {
			continue
		}
		filteredProductsIDs = append(filteredProductsIDs, p.Id)
	}
	productIDs := sample(filteredProductsIDs, maxResponsesCount)
	logger.Printf("[Recv ListRecommendations] product_ids=%v", productIDs)
	out.ProductIds = productIDs
	return out, nil
}

// determine whether or not it contains
func contains(target string, source []string) bool {
	for _, s := range source {
		if target == s {
			return true
		}
	}
	return false
}

func sample(source []string, c int) []string {
	n := len(source)
	if n <= c {
		return source
	}
	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		indices[i], indices[j] = indices[j], indices[i]
	}
	result := make([]string, 0, c)
	for i := 0; i < c; i++ {
		result = append(result, source[indices[i]])
	}
	return result
}
