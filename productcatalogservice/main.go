package main

import (
	"fmt"
	"net"
	handler "productcatalogservice/handler"
	pb "productcatalogservice/proto"
	"strconv"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
)

const PORT = 50015
const ADDRESS = "127.0.0.1"

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
		Tags:    []string{"productcatalogservice"},
		Name:    "productcatalogservice",
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

	// register grpc service
	pb.RegisterProductCatalogServiceServer(grpcServer, new(handler.ProductCatalogService))

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
