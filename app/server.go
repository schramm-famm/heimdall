package main

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/schramm-famm/heimdall/handlers"
)

func logging(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("path: %s, method: %s", r.URL.Path, r.Method)
		f.ServeHTTP(w, r)
	})
}

func strictTransportSecurity(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Force all future requests to use HTTPS
		w.Header().Set(
			"Strict-Transport-Security",
			"max-age=63072000; includeSubDomains; preload",
		)
		f.ServeHTTP(w, r)
	})
}

func reverseProxyHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		p.ServeHTTP(w, r)
	}
}

func makeServerFromMux(r *mux.Router) *http.Server {
	return &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      r,
	}
}

func main() {
	var err error

	e := &handlers.Env{
		RC: &http.Client{
			Timeout: time.Second * 10,
		},
		Hosts: make(map[string]string),
	}

	privateKeyPath := os.Getenv("PRIVATE_KEY")
	if privateKeyPath == "" {
		privateKeyPath = "id_rsa"
	}

	certPath := os.Getenv("SERVER_CERT")
	if certPath == "" {
		certPath = "server.cert"
	}

	if e.PrivateKey, err = ioutil.ReadFile(privateKeyPath); err != nil {
		log.Fatal(`Failed to read private key file: `, err)
	}

	if e.PublicKey, err = ioutil.ReadFile(privateKeyPath + ".pub"); err != nil {
		log.Fatal(`Failed to read public key file: `, err)
	}

	// /* Uncomment this to work w/o karen
	e.Hosts["karen"] = os.Getenv("KAREN_HOST")
	if e.Hosts["karen"] == "" {
		log.Fatal(`required "KAREN_HOST" environment variable not set`)
	}
	// */ // Uncomment this to work w/o karen

	httpsMux := mux.NewRouter()
	httpsMux.HandleFunc("/heimdall/v1/token", e.PostTokenHandler).Methods("POST")

	e.Hosts["patches"] = os.Getenv("PATCHES_HOST")
	if e.Hosts["patches"] != "" {
		target := "http://" + e.Hosts["patches"]
		remote, err := url.Parse(target)
		if err != nil {
			log.Fatal(`Not able to parse "PATCHES_HOST" environment variable as URL: `, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		httpsMux.HandleFunc("/patches/v1/connect/{conversation_id:[0-9]+}", reverseProxyHandler(proxy))
	}

	httpsMux.PathPrefix("/").HandlerFunc(e.ReqHandler)
	httpsMux.Use(logging, strictTransportSecurity)

	httpsSrv := &http.Server{
		Addr:         ":443",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      httpsMux,
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		},
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}

	e.RC.Transport = &http.Transport{TLSClientConfig: httpsSrv.TLSConfig}

	// Start HTTPS server
	go func() {
		log.Fatal(httpsSrv.ListenAndServeTLS(certPath, privateKeyPath))
	}()

	// Create and start internal HTTP server
	httpMux := mux.NewRouter()
	httpMux.HandleFunc("/heimdall/v1/token/auth", e.PostTokenAuthHandler).Methods("POST")
	httpSrv := makeServerFromMux(httpMux)
	httpSrv.Addr = ":80"
	log.Fatal(httpSrv.ListenAndServe())
}
