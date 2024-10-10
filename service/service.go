package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/wanmei002/testgrpc-proxy/generated/golang/everai/billings/v1"
	"github.com/wanmei002/tls/private_key"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
)

type Service struct {
	billings.UnimplementedBillingServiceServer
}

func (svc *Service) ListAppBillings(ctx context.Context, in *emptypb.Empty) (*billings.ListAppBillingsResponse, error) {
	return &billings.ListAppBillingsResponse{
		ListBillings: "list",
	}, nil
}

func (svc *Service) GetAppBilling(ctx context.Context, in *emptypb.Empty) (*billings.GetAppBillingResponse, error) {
	return &billings.GetAppBillingResponse{Resp: "i am billing"}, nil
}

func StartServer() {
	ca, err := private_key.NewCA("expvent.com")
	if err != nil {
		log.Fatal(err)
	}
	cliCert, err := tls.X509KeyPair(ca.CertPem(), ca.KeyPerm())
	if err != nil {
		log.Fatal(err)
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cliCert},
		ClientAuth:   tls.NoClientCert,
	}
	svc := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(cfg)),
	)
	billings.RegisterBillingServiceServer(svc, &Service{})
	reflection.Register(svc)
	ls, err := net.Listen("tcp", ":8089")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("start port 8089")
	err = svc.Serve(ls)
	if err != nil {
		fmt.Printf("failed to serve: %v", err)
	}
	return
}
