package gear

import (
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/gofiber/fiber/v2"
)

type getGearConfig500JSONResponse apitypes.ErrorResponse

func (response getGearConfig500JSONResponse) VisitGetGearConfigResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type putGearConfig500JSONResponse apitypes.ErrorResponse

func (response putGearConfig500JSONResponse) VisitPutGearConfigResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type getGearInfo500JSONResponse apitypes.ErrorResponse

func (response getGearInfo500JSONResponse) VisitGetGearInfoResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type refreshGear500JSONResponse apitypes.ErrorResponse

func (response refreshGear500JSONResponse) VisitRefreshGearResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type getConfig500JSONResponse apitypes.ErrorResponse

func (response getConfig500JSONResponse) VisitGetConfigResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type getInfo500JSONResponse apitypes.ErrorResponse

func (response getInfo500JSONResponse) VisitGetInfoResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type putInfo500JSONResponse apitypes.ErrorResponse

func (response putInfo500JSONResponse) VisitPutInfoResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type registerGear500JSONResponse apitypes.ErrorResponse

func (response registerGear500JSONResponse) VisitRegisterGearResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}
