package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/wanmei002/testgrpc-proxy/generated/golang/everai/billings/v1"
	"github.com/wanmei002/testgrpc-proxy/service"
	"github.com/wanmei002/tls/private_key"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	log.SetPrefix("[proxy] ")
	log.SetOutput(os.Stdout)
	go service.StartServer()
	go clientReq()
	go middleProxy()

	svc := &http.Server{
		Addr:    ":9001",
		Handler: getHandler(),
	}
	ca, err := private_key.NewCA("expvent.com")
	if err != nil {
		log.Fatal(err)
	}
	cliCert, err := tls.X509KeyPair(ca.CertPem(), ca.KeyPerm())
	if err != nil {
		log.Fatal(err)
	}
	svc.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cliCert},
	}
	err = svc.ListenAndServeTLS("", "")
	//err = svc.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

// Upstream ...
type Upstream struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
}

func getHandler() http.Handler {
	proxy := &httputil.ReverseProxy{}
	proxy.Director = func(req *http.Request) {
		// 必需是 https
		req.URL.Scheme = "https"
		// 跳转到二级反向代理
		req.URL.Host = "127.0.0.1:9002"
		// 去除前缀
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/r/apps/test-grpc")
		req.RequestURI = strings.TrimPrefix(req.RequestURI, "/r/apps/test-grpc")
		req.Host = "127.0.0.1:9002"
	}
	proxy.Transport =
		&http2.Transport{
			AllowHTTP:       true,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	return proxy
}

func middleProxy() {
	u, err := url.Parse("https://127.0.0.1:8089")
	if err != nil {
		log.Fatal(err)
	}
	p := httputil.NewSingleHostReverseProxy(u)
	p.Transport = &http2.Transport{
		AllowHTTP:       true,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	svc := &http.Server{
		Addr:    ":9002",
		Handler: p,
	}
	ca, err := private_key.NewCA("expvent.com")
	if err != nil {
		log.Fatal(err)
	}
	cliCert, err := tls.X509KeyPair(ca.CertPem(), ca.KeyPerm())
	if err != nil {
		log.Fatal(err)
	}
	cfg := &tls.Config{
		Certificates:       []tls.Certificate{cliCert},
		InsecureSkipVerify: true,
	}

	svc.TLSConfig = cfg
	err = svc.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatal(err)
	}
}

func clientReq() {
	time.Sleep(4 * time.Second)
	opt := grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		fmt.Printf("method:%s; req:%v; reply:%v\n", method, req, reply)
		method = fmt.Sprintf("%s%s", "/r/apps/test-grpc", method)
		fmt.Println("method after: ", method)
		err := invoker(ctx, method, req, reply, cc, opts...)
		return err
	})
	ctx := context.Background()
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		"Authorization": "Bearer aaaa.bbb.ccc",
	}))
	conn, err := grpc.DialContext(context.Background(), "127.0.0.1:9001", grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})), opt)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := billings.NewBillingServiceClient(conn)

	resp, err := client.GetAppBilling(ctx, &emptypb.Empty{})
	if err != nil {
		fmt.Printf("could not greet: %v\n", err)
		return
	}
	log.Println(resp.Resp)
}
