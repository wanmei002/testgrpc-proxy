# grpc、http2 反向代理的实现
> 代码地址 [https://github.com/wanmei002/testgrpc-proxy](https://github.com/wanmei002/testgrpc-proxy)

采用golang 的自带的代理 `httputil.ReverseProxy`

transport 采用 `golang.org/x/net/http2 的http2.Transport`

grpc 使用的是 http2, 如果要代理 grpc 请求，就需要开启 tls, tls 的生成请参考我以前的博客[地址](https://blog.csdn.net/wanmei002/article/details/139602762)

### 服务端代码
```go
func StartServer() {
	ca, err := private_key.NewCA("expvent.com")
	if err != nil {
		log.Fatal(err)
	}
	// 生成证书
	cliCert, err := tls.X509KeyPair(ca.CertPem(), ca.KeyPerm())
	if err != nil {
		log.Fatal(err)
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cliCert},
		ClientAuth:   tls.NoClientCert,
	}
	svc := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(cfg)),// 使用证书
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
```

### 客户端请求代码
```go
func clientReq() {
	time.Sleep(4 * time.Second)
	opt := grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		fmt.Printf("method:%s; req:%v; reply:%v\n", method, req, reply)
		method = fmt.Sprintf("%s%s", "/r/apps/test-grpc", method) // 模拟修改路由
		fmt.Println("method after: ", method)
		err := invoker(ctx, method, req, reply, cc, opts...)
		return err
	})
	ctx := context.Background()
	// 模拟添加 token
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		"Authorization": "Bearer aaaa.bbb.ccc",
	}))
	// 连接代理服务
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
```

2. 反向代理代码
```go
func main() {
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
    proxy.Transport = &http2.Transport{
        AllowHTTP:       true,
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    return proxy
}
```

## 问题: error reading server preface: http2: frame too large
看看使用的是不是 http2.Transport, 还有是不是启用的是 https 服务

## http2 只支持 https scheme

