package main

import (
	"fmt"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "frontend/proto"
)

const (
	name    = "frontend"
	version = "1.0.0"

	defaultCurrency = "USD"
	cookieMaxAge    = 60 * 60 * 48

	cookiePrefix    = "shop_"
	cookieSessionID = cookiePrefix + "session-id"
	cookieCurrency  = cookiePrefix + "currency"
)

var (
	whitelistedCurrencies = map[string]bool{
		"USD": true,
		"EUR": true,
		"CAD": true,
		"JPY": true,
		"GBP": true,
		"TRY": true,
	}
)

type ctxKeySessionID struct{}

// 前端server
type FrontendServer struct {
	adService             pb.AdServiceClient
	cartService           pb.CartServiceClient
	checkoutService       pb.CheckoutServiceClient
	currencyService       pb.CurrencyServiceClient
	productCatalogService pb.ProductCatalogServiceClient
	recommendationService pb.RecommendationServiceClient
	shippingService       pb.ShippingServiceClient
}

func GetGrpcConn(consulClient *api.Client, serviceName string, serviceTag string) *grpc.ClientConn {
	service, _, err := consulClient.Health().Service(serviceName, serviceTag, true, nil)
	if err != nil {
		fmt.Println("Error retrieving healthy service:", err)
		return nil
	}
	if len(service) == 0 {
		fmt.Println("No healthy services found")
		return nil
	}
	s := service[0].Service
	address := s.Address + ":" + strconv.Itoa(s.Port)
	fmt.Printf("Service name: %v\n", serviceName)
	fmt.Printf("Address: %s\n", address)

	// Connect to the gRPC service with secure credentials
	creds := credentials.NewClientTLSFromCert(nil, "")
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}
	grpcConn, err := grpc.NewClient(address, opts...)
	if err != nil {
		fmt.Println("Error connecting to gRPC service:", err)
		return nil
	}

	return grpcConn
}

func main() {

	log := logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout

	// init consul config
	consulConfig := api.DefaultConfig()

	// init consul client
	consulClient, err_consul := api.NewClient(consulConfig)
	if err_consul != nil {
		fmt.Println("consul client init error:", err_consul)
		return
	}

	svc := &FrontendServer{
		adService:             pb.NewAdServiceClient(GetGrpcConn(consulClient, "adservice", "adservice")),
		cartService:           pb.NewCartServiceClient(GetGrpcConn(consulClient, "cartservice", "cartservice")),
		checkoutService:       pb.NewCheckoutServiceClient(GetGrpcConn(consulClient, "checkoutservice", "checkoutservice")),
		currencyService:       pb.NewCurrencyServiceClient(GetGrpcConn(consulClient, "currencyservice", "currencyservice")),
		productCatalogService: pb.NewProductCatalogServiceClient(GetGrpcConn(consulClient, "productcatalogservice", "productcatalogservice")),
		recommendationService: pb.NewRecommendationServiceClient(GetGrpcConn(consulClient, "recommendationservice", "recommendationservice")),
		shippingService:       pb.NewShippingServiceClient(GetGrpcConn(consulClient, "shippingservice", "shippingservice")),
	}

	r := gin.Default()

	r.FuncMap = template.FuncMap{
		"renderMoney":        renderMoney,
		"renderCurrencyLogo": renderCurrencyLogo,
	}

	r.LoadHTMLGlob("templates/*")

	r.Static("/static", "./static")

	// Home page
	r.GET("/", svc.HomeHandler)
	// Product page
	r.GET("/product/:id", svc.ProductHandler)
	// Get cart
	r.GET("/cart", svc.viewCartHandler)
	// Add to cart
	r.POST("/cart", svc.addToCartHandler)
	// empty cart
	r.POST("/cart/empty", svc.emptyCartHandler)
	// setting currency
	r.POST("/setCurrency", svc.setCurrencyHandler)
	// Log out
	r.GET("/logout", svc.logoutHandler)
	// Checkout
	r.POST("/cart/checkout", svc.placeOrderHandler)

	if err := r.Run(":8052"); err != nil {
		log.Fatalf("Failed to start gin: %v", err)
	}
}
