package handlers

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var (
	rc             *http.Client
	karenAuth      = "/karen/api/auth"
	privateKeyPath = "id_rsa"
)

func init() {
	rc = &http.Client{
		Timeout: time.Second * 10,
	}
}

func createToken(user User) (string, error) {
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(time.Hour * 24)

	claims := &TokenClaims{
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  issuedAt.Unix(),
			ExpiresAt: expiresAt.Unix(),
		},
	}

	claims.ID = user.ID
	claims.Name = user.Name
	claims.Email = user.Email

	keyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		log.Println(`Failed to read private key file: `, err)
		return "", err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
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
	/*
		resp, err := rc.Post(karenAuth, "application/json", r.Body)
		if err != nil {
			log.Printf(`Failed to send request to "%s": %s\n`, karenAuth, err.Error())
			http.Error(w, `Failed to authorize user`, http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf(`Failed to authorize user, response code of "%s" request: %d\n`, karenAuth, resp.StatusCode)
			http.Error(w, `Failed to authorize user`, resp.StatusCode)
			return
		}
	*/

	userBody := User{}
	/*
		if err = json.NewDecoder(resp.Body).Decode(&userBody); err != nil {
			log.Printf(`Failed to authorize user, unable to parse response body of "%s" request: %s\n`, karenAuth, err.Error())
			http.Error(w, `Failed to authorize user`, http.StatusInternalServerError)
			return
		}
	*/

	token, err := createToken(userBody)
	if err != nil {
		log.Println(`Failed to create token: `, err)
		http.Error(w, `Failed to create token for authorized user`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func ReqHandler(w http.ResponseWriter, r *http.Request) {}
