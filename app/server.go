package main

import (
	"crypto/tls"
	"github.com/gorilla/mux"
	"heimdall/handlers"
	"log"
	"net/http"
	"os"
	"time"
)

func makeServerFromMux(r *mux.Router) *http.Server {
	return &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      r,
	}
}

func main() {
	privateKeyPath := os.Getenv("PRIVATE_KEY")
	if privateKeyPath == "" {
		privateKeyPath = "id_rsa"
	}

	certPath := os.Getenv("SERVER_CERT")
	if certPath == "" {
		certPath = "server.cert"
	}

	httpsMux := mux.NewRouter()
	httpsMux.HandleFunc("/api/token", handlers.PostTokenHandler).Methods("POST")
	httpsMux.PathPrefix("/").Handler(http.HandlerFunc(handlers.ReqHandler))

	httpsSrv := makeServerFromMux(httpsMux)
	httpsSrv.Addr = ":443"
	httpsSrv.TLSConfig = &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}
	httpsSrv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0)

	// Start HTTPS server
	go func() {
		log.Fatal(httpsSrv.ListenAndServeTLS(certPath, privateKeyPath))
	}()

	// Start HTTP server that wil redirect to the HTTPS server
	httpMux := mux.NewRouter()
	httpMux.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newURI := "https://" + r.Host + r.URL.String()
		http.Redirect(w, r, newURI, http.StatusFound)
	}))

	httpSrv := makeServerFromMux(httpMux)
	httpSrv.Addr = ":80"

	log.Fatal(httpSrv.ListenAndServe())
}
