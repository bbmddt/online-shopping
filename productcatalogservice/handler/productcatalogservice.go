package handler

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	pb "productcatalogservice/proto"
)

var reloadCatalog bool

// log
var (
	buf    bytes.Buffer
	logger = log.New(&buf, "logger: ", log.Lshortfile)
)

// struct
type ProductCatalogService struct {
	sync.Mutex
	products []*pb.Product
}

// product list
func (s *ProductCatalogService) ListProducts(ctx context.Context, in *pb.Empty) (out *pb.ListProductsResponse, e error) {
	out = new(pb.ListProductsResponse)
	out.Products = s.parseCatalog()
	return out, nil
}

// get product by id
func (s *ProductCatalogService) GetProduct(ctx context.Context, in *pb.GetProductRequest) (out *pb.Product, e error) {
	var found *pb.Product
	out = new(pb.Product)
	products := s.parseCatalog()
	for _, p := range products {
		if in.Id == p.Id {
			found = p
		}
	}
	if found == nil {
		return out, status.Errorf(codes.NotFound, "no product with ID %s", in.Id)
	}
	out.Id = found.Id
	out.Name = found.Name
	out.Categories = found.Categories
	out.Description = found.Description
	out.Picture = found.Picture
	out.PriceUsd = found.PriceUsd
	return out, nil
}

// search product by name or description
func (s *ProductCatalogService) SearchProducts(ctx context.Context, in *pb.SearchProductsRequest) (out *pb.SearchProductsResponse, e error) {
	var ps []*pb.Product
	out = new(pb.SearchProductsResponse)
	products := s.parseCatalog()
	for _, p := range products {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(in.Query)) ||
			strings.Contains(strings.ToLower(p.Description), strings.ToLower(in.Query)) {
			ps = append(ps, p)
		}
	}
	out.Results = ps
	return out, nil
}

// load product json file
func (s *ProductCatalogService) readCatalogFile() (*pb.ListProductsResponse, error) {
	s.Lock()
	defer s.Unlock()
	catalogJSON, err := os.ReadFile("data/products.json")
	if err != nil {
		logger.Printf("Failed to open product json file: %v", err)
		return nil, err
	}
	catalog := &pb.ListProductsResponse{}
	if err := protojson.Unmarshal(catalogJSON, catalog); err != nil {
		logger.Printf("Failed to parse product json file: %v", err)
		return nil, err
	}
	logger.Printf("Parsing Product json file Succeeded")
	return catalog, nil
}

// parse product json file
func (s *ProductCatalogService) parseCatalog() []*pb.Product {
	if reloadCatalog || len(s.products) == 0 {
		catalog, err := s.readCatalogFile()
		if err != nil {
			return []*pb.Product{}
		}
		s.products = catalog.Products
	}
	return s.products
}

// initialize
func init() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for {
			sig := <-sigs
			logger.Printf("Receive Signal: %s", sig)
			if sig == syscall.SIGUSR1 {
				reloadCatalog = true
				logger.Printf("Product info can be loaded")
			} else {
				reloadCatalog = false
				logger.Printf("Can't load product info")
			}
		}
	}()
}
