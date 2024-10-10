# grpc、http2 代理

1. 在这里我们采用包 `golang.org/x/net/http2`

采用golang 的自带的代理 `httputil.ReverseProxy`

transport 采用 golang.org/x/net/http2 的http2.Transport

grpc 使用的是 http2, 如果要代理 grpc 请求，就需要开启 tls, tls 的生成请参考我以前的博客[地址](https://blog.csdn.net/wanmei002/article/details/139602762)

2. 代理代码
```go
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
	ca, err := cax.NewCA("expvent.com")
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
```

## 问题: error reading server preface: http2: frame too large
看看使用的是不是 http2.Transport, 还有是不是启用的是 https 服务

## http2 只支持 https scheme

