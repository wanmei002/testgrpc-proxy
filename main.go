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

	proxy := &Upstream{target: nil, proxy: &httputil.ReverseProxy{}}
	svc := &http.Server{
		Addr:    ":9001",
		Handler: proxy,
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

func (p *Upstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Forwarded-For", r.Host)
	p.proxy.Director = func(req *http.Request) {
		log.Printf("req: %#v\n", req.Header)
		req.URL.Scheme = "https"
		req.URL.Host = "127.0.0.1:9002"
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/r/apps/test-grpc")
		req.RequestURI = strings.TrimPrefix(req.RequestURI, "/r/apps/test-grpc")
		req.Host = "127.0.0.1:9002"
		log.Printf("req2: %#v\n", req)
		log.Printf("req2: %#v\n", req.URL)
	}
	p.proxy.Transport =
		&http2.Transport{
			AllowHTTP:       true,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	p.proxy.ServeHTTP(w, r)
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
