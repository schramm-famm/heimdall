package main

import (
	"crypto/tls"
	"github.com/gorilla/mux"
	"github.com/schramm-famm/heimdall/handlers"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
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
	log.Fatal(httpsSrv.ListenAndServeTLS(certPath, privateKeyPath))
}
