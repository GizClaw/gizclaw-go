package gizclaw

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestFiberHTTPHandlerHidesPanicDetail(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/panic", func(*fiber.Ctx) error {
		panic("secret-panic-detail")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	fiberHTTPHandler(app).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("body = %q", rec.Body.String())
	}
}
