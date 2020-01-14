package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/schramm-famm/heimdall/models"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	privateKeyBytes []byte
	publicKeyBytes  []byte
	rc              *http.Client
	authRoute       = "http://karen/api/auth"
	whitelist       = []string{"/", "/login", "/register"}
)

func init() {
	rc = &http.Client{
		Timeout: time.Second * 10,
	}

	privateKeyPath := os.Getenv("PRIVATE_KEY")
	if privateKeyPath == "" {
		privateKeyPath = "id_rsa"
	}

	if keyBytes, err := ioutil.ReadFile(privateKeyPath); err != nil {
		log.Println(`Failed to read private key file: `, err)
	} else {
		privateKeyBytes = keyBytes
	}

	if keyBytes, err := ioutil.ReadFile(privateKeyPath + ".pub"); err != nil {
		log.Println(`Failed to read public key file: `, err)
	} else {
		publicKeyBytes = keyBytes
	}
}

func createToken(user models.User) (string, error) {
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

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
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

func PostTokenHandler(w http.ResponseWriter, r *http.Request) {
	// /* Uncomment this for token generation to work w/o karen
	resp, err := rc.Post(authRoute, "application/json", r.Body)
	if err != nil {
		log.Printf(`Failed to send request to "%s": %s\n`, authRoute, err.Error())
		http.Error(w, `Failed to authorize user`, http.StatusInternalServerError)
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
		http.Error(w, `Failed to authorize user`, http.StatusInternalServerError)
		return
	}
	// */ // Uncomment this for token generation to work w/o karen

	token, err := createToken(userBody)
	if err != nil {
		log.Println(`Failed to create token: `, err)
		http.Error(w, `Failed to create token for authorized user`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func forwardRequest(w http.ResponseWriter, r *http.Request) {
	r.RequestURI = ""
	if resp, err := rc.Do(r); err != nil {
		log.Println("Failed to forward user request:", err)
		http.Error(w, "Failed to forward request", http.StatusInternalServerError)
	} else if err = resp.Write(w); err != nil {
		log.Println("Failed to write response to user:", err)
		http.Error(w, "Failed to write response to user", http.StatusInternalServerError)
	}

	return
}

func validateToken(tokenString string) (bool, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyBytes)

		return publicKey, err
	})

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return false, errors.New("malformed token")
			} else if ve.Errors&(jwt.ValidationErrorExpired) != 0 {
				return false, errors.New("expired token")
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return false, errors.New("token not activated yet")
			} else {
				fmt.Println("Couldn't handle this token:", err)
				return false, fmt.Errorf("failed to handle token: %s", err.Error())
			}
		} else {
			return false, fmt.Errorf("failed to handle this token: %s", err)
		}
	}

	return token.Valid, nil
}

func ReqHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the path of the request is in the whitelist
	for _, route := range whitelist {
		if r.URL.Path == route {
			forwardRequest(w, r)
			return
		}
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Println(`Request to protected route does not contain "Authorization" header`)
		http.Error(w, `Request to protected route does not contain "Authorization" header`, http.StatusUnauthorized)
		return
	}

	// "Authorization" header must have format of "Bearer <token>"
	authHeaderSlice := strings.Split(strings.Trim(authHeader, " "), " ")
	if len(authHeaderSlice) != 2 || authHeaderSlice[0] != "Bearer" {
		log.Println(`Request to protected route contains invalid "Authorization" header`)
		http.Error(w, `Request to protected route contains invalid "Authorization" header`, http.StatusUnauthorized)
		return
	}

	if valid, err := validateToken(authHeaderSlice[1]); !valid {
		log.Println("Provided token is invalid:", err)
		http.Error(w, "Provided token is invalid: "+err.Error(), http.StatusUnauthorized)
		return
	}

	log.Println("Token validated!")

	forwardRequest(w, r)
}
