package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/schramm-famm/heimdall/models"
)

var (
	privateKey []byte
	publicKey  []byte
)

func init() {
	var err error
	privateKeyPath := os.Getenv("PRIVATE_KEY")

	if privateKey, err = ioutil.ReadFile(privateKeyPath); err != nil {
		log.Fatal(`Failed to read private key file: `, err)
	}

	if publicKey, err = ioutil.ReadFile(privateKeyPath + ".pub"); err != nil {
		log.Fatal(`Failed to read public key file: `, err)
	}
}

func TestPostTokenHandler(t *testing.T) {
	tests := []struct {
		Name           string
		AuthStatusCode int
		ReqBody        map[string]string
	}{
		{
			Name:           "Successful token generation",
			AuthStatusCode: http.StatusOK,
			ReqBody: map[string]string{
				"email":    "fake@gmail.com",
				"password": "password",
			},
		},
		{
			Name:           "Unsuccessful token generation with bad request",
			AuthStatusCode: http.StatusBadRequest,
			ReqBody: map[string]string{
				"blah":     "fake@gmail.com",
				"password": "password",
			},
		},
		{
			Name:           "Unsuccessful token generation with invalid user",
			AuthStatusCode: http.StatusForbidden,
			ReqBody: map[string]string{
				"email":    "fake@gmail.com",
				"password": "password",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mockAuthHandler := func(w http.ResponseWriter, r *http.Request) {
				userBody := models.User{}
				if err := json.NewDecoder(r.Body).Decode(&userBody); err != nil {
					http.Error(w, "Failed to parse", http.StatusBadRequest)
					return
				}

				if test.AuthStatusCode != http.StatusOK {
					http.Error(w, "Authentication failed", test.AuthStatusCode)
					return
				}

				userBody.ID = 1
				userBody.Name = "Fake Name"

				json.NewEncoder(w).Encode(userBody)
			}

			server := httptest.NewServer(http.HandlerFunc(mockAuthHandler))
			defer server.Close()

			e := &Env{
				PrivateKey: privateKey,
				PublicKey:  publicKey,
				RC:         server.Client(),
				Hosts:      map[string]string{"karen": strings.TrimPrefix(server.URL, "http://")},
			}

			rBody, _ := json.Marshal(test.ReqBody)
			r := httptest.NewRequest("POST", "/heimdall/v1/token", bytes.NewReader([]byte(rBody)))
			w := httptest.NewRecorder()

			e.PostTokenHandler(w, r)

			if w.Code != test.AuthStatusCode {
				t.Errorf("Response has incorrect status code, expected status code %d, got %d", test.AuthStatusCode, w.Code)
			}

			if w.Code == http.StatusOK {
				tokenBody := struct {
					Token string `json:"token"`
				}{}

				_ = json.NewDecoder(w.Body).Decode(&tokenBody)

				if tokenBody.Token == "" {
					t.Error("Token creation failed, expected token to not be empty")
				}
			}
		})
	}
}

func TestTokenAuthHandler(t *testing.T) {
	type responseBody struct {
		UserID int `json:"user_id"`
	}

	e := &Env{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	userID := 1337
	validToken, err := e.createToken(models.User{
		ID: userID,
	})
	if err != nil {
		t.Error("Failed to generate valid token: " + err.Error())
	}

	tests := []struct {
		Name       string
		StatusCode int
		ReqBody    interface{}
	}{
		{
			Name:       "Successful token validation",
			StatusCode: http.StatusOK,
			ReqBody: map[string]interface{}{
				"token": validToken,
			},
		},
		{
			Name:       "Failed token validation (malformed token)",
			StatusCode: http.StatusNotFound,
			ReqBody: map[string]interface{}{
				"token": "foobar",
			},
		},
		{
			Name:       "Failed token validation (empty json)",
			StatusCode: http.StatusBadRequest,
			ReqBody:    map[string]interface{}{},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			rBody, _ := json.Marshal(test.ReqBody)
			r := httptest.NewRequest("POST", "/heimdall/v1/token/auth", bytes.NewReader([]byte(rBody)))
			w := httptest.NewRecorder()

			e.PostTokenAuthHandler(w, r)

			if w.Code != test.StatusCode {
				t.Errorf("Response has incorrect status code, expected status code %d, got %d", test.StatusCode, w.Code)
			}

			if w.Code == http.StatusOK {
				// Validate HTTP response content
				resBody := responseBody{}
				_ = json.NewDecoder(w.Body).Decode(&resBody)
				if userID != resBody.UserID {
					t.Errorf("Response body has incorrect user ID, expected %d, got %d", userID, resBody.UserID)
				}
			}
		})
	}
}

func TestReqHandler(t *testing.T) {
	e := &Env{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	validToken, err := e.createToken(models.User{})
	if err != nil {
		t.Error("Failed to generate valid token: " + err.Error())
	}

	tests := []struct {
		Name       string
		StatusCode int
		Header     map[string]string
		Path       string
		Method     string
	}{
		{
			Name:       "Successful route validation",
			StatusCode: http.StatusOK,
			Header:     map[string]string{"Authorization": "Bearer " + validToken},
			Path:       "/karen",
			Method:     http.MethodGet,
		},
		{
			Name:       "Successful access to whitelisted route",
			StatusCode: http.StatusCreated,
			Path:       "/karen/v1/users",
			Method:     http.MethodPost,
		},
		{
			Name:       "Unsuccessful route validation without Authorization header",
			StatusCode: http.StatusUnauthorized,
			Path:       "/karen",
			Method:     http.MethodPost,
		},
		{
			Name:       "Unsuccessful route validation with invalid token",
			StatusCode: http.StatusUnauthorized,
			Header:     map[string]string{"Authorization": "Bearer invalid"},
			Path:       "/karen",
			Method:     http.MethodPost,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.StatusCode)
			}

			server := httptest.NewServer(http.HandlerFunc(mockHandler))
			defer server.Close()

			e.RC = server.Client()

			re := regexp.MustCompile("[^/]+")
			appName := re.FindString(test.Path)
			e.Hosts = map[string]string{appName: strings.TrimPrefix(server.URL, "http://")}

			r := httptest.NewRequest(test.Method, test.Path, bytes.NewReader([]byte{}))
			r.Header = http.Header{}
			for key, val := range test.Header {
				r.Header.Set(key, val)
			}
			w := httptest.NewRecorder()

			e.ReqHandler(w, r)

			if w.Code != test.StatusCode {
				t.Errorf("Response has incorrect status code, expected status code %d, got %d", test.StatusCode, w.Code)
			}
		})
	}
}
