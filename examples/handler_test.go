package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/jsonapi"
)

func TestExampleHandler_post(t *testing.T) {
	blog := fixtureBlogCreate(1)
	requestBody := bytes.NewBuffer(nil)
	jsonapi.MarshalOnePayloadEmbedded(requestBody, blog)

	r, err := http.NewRequest(http.MethodPost, "/blogs?id=1", requestBody)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set(headerAccept, jsonapi.MediaType)

	rr := httptest.NewRecorder()
	handler := &ExampleHandler{}
	handler.ServeHTTP(rr, r)

	if e, a := http.StatusCreated, rr.Code; e != a {
		t.Fatalf("Expected a status of %d, got %d", e, a)
	}
}

func TestExampleHandler_put(t *testing.T) {
	blogs := []interface{}{
		fixtureBlogCreate(1),
		fixtureBlogCreate(2),
		fixtureBlogCreate(3),
	}
	requestBody := bytes.NewBuffer(nil)
	jsonapi.MarshalPayload(requestBody, blogs)

	r, err := http.NewRequest(http.MethodPut, "/blogs", requestBody)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set(headerAccept, jsonapi.MediaType)

	rr := httptest.NewRecorder()
	handler := &ExampleHandler{}
	handler.ServeHTTP(rr, r)

	if e, a := http.StatusOK, rr.Code; e != a {
		t.Fatalf("Expected a status of %d, got %d", e, a)
	}
}

func TestExampleHandler_get_show(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/blogs?id=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set(headerAccept, jsonapi.MediaType)

	rr := httptest.NewRecorder()
	handler := &ExampleHandler{}
	handler.ServeHTTP(rr, r)

	if e, a := http.StatusOK, rr.Code; e != a {
		t.Fatalf("Expected a status of %d, got %d", e, a)
	}
}

func TestExampleHandler_get_list(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/blogs", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set(headerAccept, jsonapi.MediaType)

	rr := httptest.NewRecorder()
	handler := &ExampleHandler{}
	handler.ServeHTTP(rr, r)

	if e, a := http.StatusOK, rr.Code; e != a {
		t.Fatalf("Expected a status of %d, got %d", e, a)
	}
}

func TestHttpErrorWhenHeaderDoesNotMatch(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/blogs", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set(headerAccept, "application/xml")

	rr := httptest.NewRecorder()
	handler := &ExampleHandler{}
	handler.ServeHTTP(rr, r)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatal("expected Unsupported Media Type staus error")
	}
}

func TestHttpErrorWhenMethodDoesNotMatch(t *testing.T) {
	r, err := http.NewRequest(http.MethodPatch, "/blogs", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set(headerAccept, jsonapi.MediaType)

	rr := httptest.NewRecorder()
	handler := &ExampleHandler{}
	handler.ServeHTTP(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Fatal("expected HTTP Status Not Found status error")
	}
}
