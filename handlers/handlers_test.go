package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func resetVariables() {
	karenAuth = "/karen/api/auth"
	privateKeyPath = "id_rsa"
}

func TestPostTokenHandler(t *testing.T) {
	resetVariables()

	mockAuthHandler := func(w http.ResponseWriter, r *http.Request) {
		userBody := User{}
		if err := json.NewDecoder(r.Body).Decode(&userBody); err != nil {
			log.Println("Failed to parse request body: ", err)
			http.Error(w, "Failed to parse", http.StatusBadRequest)
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
	privateKeyPath = "../tmp/id_rsa"

	r := httptest.NewRequest("POST", "/api/token", bytes.NewReader([]byte(`{"email":"fake@gmail.com","password":"fakepassword"}`)))
	w := httptest.NewRecorder()

	PostTokenHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("User validation failed, expected status code %d, got %d", http.StatusOK, w.Code)
	}

	tokenBody := struct {
		Token string `json:"token"`
	}{}

	_ = json.NewDecoder(w.Body).Decode(&tokenBody)

	if tokenBody.Token == "" {
		t.Error("Token creation failed, expected token to not be empty")
	}
}
