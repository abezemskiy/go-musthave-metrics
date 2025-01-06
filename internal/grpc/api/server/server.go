package server

import (
	"log"
	"net"

	server "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/impl"
	pb "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

func Serve(netAddr string, stor repositories.IStorage) error {
	lis, err := net.Listen("tcp", netAddr)
	if err != nil {
		log.Fatalf("Error starting gRPC server: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterServiceServer(grpcServer, server.NewServer(stor))

	reflection.Register(grpcServer)

	return grpcServer.Serve(lis)
}
