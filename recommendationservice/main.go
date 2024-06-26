package main

import (
	"fmt"
	"net"
	handler "recommendationservice/handler"
	pb "recommendationservice/proto"
	"strconv"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
)

const PORT = 50016
const ADDRESS = "127.0.0.1"

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
	// creds := credentials.NewClientTLSFromCert(nil, "")
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
	}
	grpcConn, err := grpc.NewClient(address, opts...)
	if err != nil {
		fmt.Println("Error connecting to gRPC service:", err)
		return nil
	}

	return grpcConn
}

func main() {
	ipport := ADDRESS + ":" + strconv.Itoa(PORT)
	// -------------Register on consul---------------
	// init consul config
	consulConfig := api.DefaultConfig()

	// init consul client
	consulClient, err_consul := api.NewClient(consulConfig)
	if err_consul != nil {
		fmt.Println("consul client init error:", err_consul)
		return
	}

	// init consul service registration
	reg := api.AgentServiceRegistration{
		Tags:    []string{"recommendationservice"},
		Name:    "recommendationservice",
		Address: ADDRESS,
		Port:    PORT,
	}

	// register grpc service on consul
	err_agent := consulClient.Agent().ServiceRegister(&reg)
	if err_agent != nil {
		fmt.Println("grpc service register error:", err_agent)
		return
	}

	//-----------------------grpc code--------------------------------
	// init grpc server
	grpcServer := grpc.NewServer()
	recommendationservice := &handler.RecommendationService{
		ProductCatalogService: pb.NewProductCatalogServiceClient(GetGrpcConn(consulClient, "productcatalogservice", "productcatalogservice")),
	}

	// register grpc service
	pb.RegisterRecommendationServiceServer(grpcServer, recommendationservice)

	// start grpc listen
	listen, err := net.Listen("tcp", ipport)
	if err != nil {
		fmt.Println("listen error:", err)
		return
	}
	defer listen.Close()

	// start grpc server
	fmt.Println("Server started successfully.")

	err_grpc := grpcServer.Serve(listen)
	if err_grpc != nil {
		fmt.Println("grpc server error:", err_grpc)
		return
	}
}
