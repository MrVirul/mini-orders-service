package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type fakePinger struct{ err error }

func (f fakePinger) PingContext(context.Context) error { return f.err }

func getHealth(t *testing.T, p pinger) (int, map[string]string) {
	t.Helper()

	app := newApp(p)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	return resp.StatusCode, body
}

func TestHealth_MasterUp(t *testing.T) {
	status, body := getHealth(t, fakePinger{})

	if status != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", status, fiber.StatusOK)
	}
	if body["status"] != "ok" || body["master"] != "up" {
		t.Fatalf("body = %v, want status ok / master up", body)
	}
}

func TestHealth_MasterDown(t *testing.T) {
	status, body := getHealth(t, fakePinger{err: errors.New("connection refused")})

	if status != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", status, fiber.StatusServiceUnavailable)
	}
	if body["status"] != "unavailable" || body["master"] != "down" {
		t.Fatalf("body = %v, want status unavailable / master down", body)
	}
}
