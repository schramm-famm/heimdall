package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/schramm-famm/heimdall/models"
)

// Env represents all application-level items that are needed by HTTP handlers.
type Env struct {
	RC         *http.Client
	PrivateKey []byte
	PublicKey  []byte
	Hosts      map[string]string
}

const (
	authRoute = "/karen/v1/users/auth"
)

var (
	whitelist = map[string]string{"/karen/v1/users": "POST"}
)

// createToken generates a token with claims based on a registered user.
func (e *Env) createToken(user models.User) (string, error) {
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(time.Hour * 24)

	claims := &models.TokenClaims{
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  issuedAt.Unix(),
			ExpiresAt: expiresAt.Unix(),
		},
	}

	claims.ID = user.ID
	claims.Name = user.Name
	claims.Email = user.Email

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(e.PrivateKey)
	if err != nil {
		log.Println(`Failed to parse RSA private key: `, err)
		return "", err
	}

	if token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privateKey); err != nil {
		log.Println(`Failed to sign token string: `, err)
		return "", err
	} else {
		return token, nil
	}
}

// PostTokenHandler verifies credentials passed in an HTTP request by
// communicating with the karen service. A token will be generated and put in
// the response if the credentials are valid.
func (e *Env) PostTokenHandler(w http.ResponseWriter, r *http.Request) {
	// /* Uncomment this for token generation to work w/o karen
	resp, err := e.RC.Post("http://"+e.Hosts["karen"]+authRoute, "application/json", r.Body)
	if err != nil {
		log.Printf(`Failed to send request to "%s": %s\n`, authRoute, err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf(`Failed to authorize user, response code of "%s" request: %d\n`, authRoute, resp.StatusCode)
		http.Error(w, `Failed to authorize user`, resp.StatusCode)
		return
	}
	// */ // Uncomment this for token generation to work w/o karen

	userBody := models.User{}
	// /* Uncomment this for token generation to work w/o karen
	if err = json.NewDecoder(resp.Body).Decode(&userBody); err != nil {
		log.Printf(`Failed to authorize user, unable to parse response body of "%s" request: %s\n`, authRoute, err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// */ // Uncomment this for token generation to work w/o karen

	token, err := e.createToken(userBody)
	if err != nil {
		log.Println(`Failed to create token: `, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// PostTokenAuthHandler checks whether a token sent in an HTTP request is valid
// and responds with the encoded user ID if it is.
func (e *Env) PostTokenAuthHandler(w http.ResponseWriter, r *http.Request) {
	tokenBody := models.AuthBody{}
	if err := json.NewDecoder(r.Body).Decode(&tokenBody); err != nil {
		errMsg := "Failed to read/parse request body"
		log.Println(errMsg, err.Error())
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	if tokenBody.Token == "" {
		errMsg := `Request body has missing/empty "token" field`
		log.Println(errMsg)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	claims, err := e.validateToken(tokenBody.Token)
	if err != nil {
		errMsg := "Token invalid"
		log.Println(errMsg + ": " + err.Error())
		http.Error(w, errMsg, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"user_id": claims.ID})
}

// forwardRequest forwards a client request to the service associated with the
// request path, if there is one, and returns the response to the client.
func (e *Env) forwardRequest(w http.ResponseWriter, r *http.Request) {
	r.RequestURI = ""
	urlString := r.URL.String()
	re := regexp.MustCompile("[^/]+")

	// Parse the URL for the service name
	appName := re.FindString(urlString)
	if appName == "" {
		log.Printf("Service name not provided")
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Check if app host is already cached
	if _, ok := e.Hosts[appName]; !ok {
		// Get the IP address from the environment variable
		re = regexp.MustCompile(`\W`)
		// The environment variables are the app names capitalized + _HOST
		envVar := strings.ToUpper(re.ReplaceAllString(appName, "") + "_HOST")
		appHost := os.Getenv(envVar)
		if appHost == "" {
			log.Printf(`Service "%s" could not be found`, appName)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		e.Hosts[appName] = appHost
	}

	// Build the new URL
	if newURL, err := url.Parse("http://" + e.Hosts[appName] + urlString); err != nil {
		log.Println("Failed to create new URL: ", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	} else {
		r.URL = newURL
	}

	resp, err := e.RC.Do(r)
	if err != nil {
		log.Println("Failed to forward user request:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	// Copy headers from response and add CORS header
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Copy status code and body
	w.WriteHeader(resp.StatusCode)
	if _, err = io.Copy(w, resp.Body); err != nil {
		log.Println("Failed to write response to user:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	return
}

// validateToken checks whether a given token is valid and returns the encoded
// claims if it is.
func (e *Env) validateToken(tokenString string) (*models.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&models.TokenClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			publicKey, err := jwt.ParseRSAPublicKeyFromPEM(e.PublicKey)

			return publicKey, err
		},
	)

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, errors.New("malformed token")
			} else if ve.Errors&(jwt.ValidationErrorExpired) != 0 {
				return nil, errors.New("expired token")
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, errors.New("token not activated yet")
			} else {
				log.Println("Couldn't handle this token: ", err)
				return nil, fmt.Errorf("failed to handle token: %s", err.Error())
			}
		} else {
			return nil, fmt.Errorf("failed to handle this token: %s", err)
		}
	}

	claims := token.Claims.(*models.TokenClaims)
	return claims, nil
}

// ReqHandler handles all HTTP requests to be forwarded.
func (e *Env) ReqHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the path of the request is in the whitelist
	for route, method := range whitelist {
		if r.URL.Path == route && r.Method == method {
			e.forwardRequest(w, r)
			return
		}
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		errMsg := `Request to protected route does not contain "Authorization" header`
		log.Println(errMsg)
		http.Error(w, errMsg, http.StatusUnauthorized)
		return
	}

	// "Authorization" header must have format of "Bearer <token>"
	authHeaderSlice := strings.Split(strings.Trim(authHeader, " "), " ")
	if len(authHeaderSlice) != 2 || authHeaderSlice[0] != "Bearer" {
		errMsg := `Request to protected route contains invalid "Authorization" header`
		log.Println(errMsg)
		http.Error(w, errMsg, http.StatusUnauthorized)
		return
	}

	claims, err := e.validateToken(authHeaderSlice[1])
	if err != nil {
		errMsg := "Token invalid"
		log.Println(errMsg + ": " + err.Error())
		http.Error(w, errMsg, http.StatusUnauthorized)
		return
	}

	r.Header.Add("User-ID", strconv.Itoa(claims.ID))
	e.forwardRequest(w, r)
}

// OptionsHandler handles pre-flight CORS HTTP requests and responds with the
// headers needed to allow CORS for certain methods and headers and all origins.
func (e *Env) OptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusNoContent)
}
