package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func resetVariables() {
	karenAuth = "/karen/api/auth"
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
			resetVariables()

			mockAuthHandler := func(w http.ResponseWriter, r *http.Request) {
				userBody := User{}
				if err := json.NewDecoder(r.Body).Decode(&userBody); err != nil {
					http.Error(w, "Failed to parse", http.StatusBadRequest)
					return
				}

				if test.AuthStatusCode != http.StatusOK {
					http.Error(w, "Authentication failed", test.AuthStatusCode)
					return
				}

				userBody.ID = "fake-id"
				userBody.Name = "Fake Name"

				json.NewEncoder(w).Encode(userBody)
			}

			server := httptest.NewServer(http.HandlerFunc(mockAuthHandler))
			defer server.Close()

			rc = server.Client()
			karenAuth = server.URL

			rBody, _ := json.Marshal(test.ReqBody)
			r := httptest.NewRequest("POST", "/api/token", bytes.NewReader([]byte(rBody)))
			w := httptest.NewRecorder()

			PostTokenHandler(w, r)

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

func TestReqHandler(t *testing.T) {
	validToken, err := createToken(User{})
	if err != nil {
		t.Error("Failed to generate valid token: " + err.Error())
	}

	tests := []struct {
		Name       string
		StatusCode int
		Header     map[string]string
		Path       string
	}{
		{
			Name:       "Successful route validation",
			StatusCode: http.StatusOK,
			Header:     map[string]string{"Authorization": "Bearer " + validToken},
			Path:       "/karen",
		},
		{
			Name:       "Successful access to whitelisted route",
			StatusCode: http.StatusOK,
			Path:       "/login",
		},
		{
			Name:       "Unsuccessful route validation without Authorization header",
			StatusCode: http.StatusUnauthorized,
			Path:       "/karen",
		},
		{
			Name:       "Unsuccessful route validation with invalid token",
			StatusCode: http.StatusUnauthorized,
			Header:     map[string]string{"Authorization": "Bearer invalid"},
			Path:       "/karen",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}

			server := httptest.NewServer(http.HandlerFunc(mockHandler))
			defer server.Close()

			rc = server.Client()

			r := httptest.NewRequest("GET", server.URL+test.Path, bytes.NewReader([]byte{}))
			r.Header = http.Header{}
			for key, val := range test.Header {
				r.Header.Set(key, val)
			}
			w := httptest.NewRecorder()

			ReqHandler(w, r)

			if w.Code != test.StatusCode {
				t.Errorf("Response has incorrect status code, expected status code %d, got %d", test.StatusCode, w.Code)
			}
		})
	}
}
