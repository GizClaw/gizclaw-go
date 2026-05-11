package peer

import (
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

func TestHTTPErrorHelpersAndVisitors(t *testing.T) {
	if apitypes.NewErrorResponse("G", "boom").Error.Code != "G" {
		t.Fatal("NewErrorResponse code mismatch")
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
		return getGearConfig500JSONResponse(apitypes.NewErrorResponse("ERR", "boom")).VisitGetPeerConfigResponse(c)
	})
	checkStatus("put-gear-config", func(c *fiber.Ctx) error {
		return putGearConfig500JSONResponse(apitypes.NewErrorResponse("ERR", "boom")).VisitPutPeerConfigResponse(c)
	})
	checkStatus("get-gear-info", func(c *fiber.Ctx) error {
		return getGearInfo500JSONResponse(apitypes.NewErrorResponse("ERR", "boom")).VisitGetPeerInfoResponse(c)
	})
	checkStatus("refresh-gear", func(c *fiber.Ctx) error {
		return refreshGear500JSONResponse(apitypes.NewErrorResponse("ERR", "boom")).VisitRefreshPeerResponse(c)
	})
	checkStatus("get-config", func(c *fiber.Ctx) error {
		return getConfig500JSONResponse(apitypes.NewErrorResponse("ERR", "boom")).VisitGetConfigResponse(c)
	})
	checkStatus("get-info", func(c *fiber.Ctx) error {
		return getInfo500JSONResponse(apitypes.NewErrorResponse("ERR", "boom")).VisitGetInfoResponse(c)
	})
	checkStatus("put-info", func(c *fiber.Ctx) error {
		return putInfo500JSONResponse(apitypes.NewErrorResponse("ERR", "boom")).VisitPutInfoResponse(c)
	})
	checkStatus("register-gear", func(c *fiber.Ctx) error {
		return registerGear500JSONResponse(apitypes.NewErrorResponse("ERR", "boom")).VisitRegisterGearResponse(c)
	})

	var (
		_ apitypes.ErrorResponse
		_ apitypes.ErrorResponse
		_ apitypes.ErrorResponse
	)
}
