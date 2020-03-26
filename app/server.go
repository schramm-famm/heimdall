package main

import (
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

	externalMux := mux.NewRouter()
	externalMux.HandleFunc("/heimdall/v1/token", e.PostTokenHandler).Methods("POST")

	e.Hosts["patches"] = os.Getenv("PATCHES_HOST")
	if e.Hosts["patches"] != "" {
		target := "http://" + e.Hosts["patches"]
		remote, err := url.Parse(target)
		if err != nil {
			log.Fatal(`Not able to parse "PATCHES_HOST" environment variable as URL: `, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		externalMux.HandleFunc(
			"/patches/v1/connect/{conversation_id:[0-9]+}",
			reverseProxyHandler(proxy),
		)
	}

	externalMux.PathPrefix("/").HandlerFunc(e.ReqHandler)
	externalMux.Use(logging)

	externalSrv := makeServerFromMux(externalMux)
	externalSrv.Addr = ":80"

	go func() {
		log.Fatal(externalSrv.ListenAndServe())
	}()

	// Create server for handling internal requests
	internalMux := mux.NewRouter()
	internalMux.HandleFunc(
		"/heimdall/v1/token/auth",
		e.PostTokenAuthHandler,
	).Methods("POST")

	internalSrv := makeServerFromMux(internalMux)
	internalSrv.Addr = ":88"
	log.Fatal(internalSrv.ListenAndServe())
}
