package gear

import (
	"testing"

	apitypes "github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

func TestHTTPErrorHelpersAndVisitors(t *testing.T) {
	if gearError("G", "boom").Error.Code != "G" {
		t.Fatal("gearError code mismatch")
	}
	if publicError("P", "boom").Error.Code != "P" {
		t.Fatal("publicError code mismatch")
	}

	app := fiber.New()
	t.Cleanup(func() {
		_ = app.Shutdown()
	})

	checkStatus := func(name string, visit func(*fiber.Ctx) error) {
		t.Helper()
		var reqCtx fasthttp.RequestCtx
		ctx := app.AcquireCtx(&reqCtx)
		defer app.ReleaseCtx(ctx)
		if err := visit(ctx); err != nil {
			t.Fatalf("%s error: %v", name, err)
		}
		if ctx.Response().StatusCode() != 500 {
			t.Fatalf("%s status = %d", name, ctx.Response().StatusCode())
		}
	}

	checkStatus("get-gear-config", func(c *fiber.Ctx) error {
		return getGearConfig500JSONResponse(adminError("ERR", "boom")).VisitGetGearConfigResponse(c)
	})
	checkStatus("put-gear-config", func(c *fiber.Ctx) error {
		return putGearConfig500JSONResponse(adminError("ERR", "boom")).VisitPutGearConfigResponse(c)
	})
	checkStatus("get-gear-info", func(c *fiber.Ctx) error {
		return getGearInfo500JSONResponse(adminError("ERR", "boom")).VisitGetGearInfoResponse(c)
	})
	checkStatus("refresh-gear", func(c *fiber.Ctx) error {
		return refreshGear500JSONResponse(adminError("ERR", "boom")).VisitRefreshGearResponse(c)
	})
	checkStatus("get-config", func(c *fiber.Ctx) error {
		return getConfig500JSONResponse(gearError("ERR", "boom")).VisitGetConfigResponse(c)
	})
	checkStatus("get-info", func(c *fiber.Ctx) error {
		return getInfo500JSONResponse(gearError("ERR", "boom")).VisitGetInfoResponse(c)
	})
	checkStatus("put-info", func(c *fiber.Ctx) error {
		return putInfo500JSONResponse(gearError("ERR", "boom")).VisitPutInfoResponse(c)
	})
	checkStatus("register-gear", func(c *fiber.Ctx) error {
		return registerGear500JSONResponse(publicError("ERR", "boom")).VisitRegisterGearResponse(c)
	})

	var (
		_ apitypes.ErrorResponse
		_ apitypes.ErrorResponse
		_ apitypes.ErrorResponse
	)
}
