package gear

import (
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/serverpublic"
	"github.com/gofiber/fiber/v2"
)

func adminError(code, message string) adminservice.ErrorResponse {
	return adminservice.ErrorResponse{
		Error: adminservice.ErrorPayload{
			Code:    code,
			Message: message,
		},
	}
}

func gearError(code, message string) gearservice.ErrorResponse {
	return gearservice.ErrorResponse{
		Error: gearservice.ErrorPayload{
			Code:    code,
			Message: message,
		},
	}
}

func publicError(code, message string) serverpublic.ErrorResponse {
	return serverpublic.ErrorResponse{
		Error: serverpublic.ErrorPayload{
			Code:    code,
			Message: message,
		},
	}
}

type getGearConfig500JSONResponse adminservice.ErrorResponse

func (response getGearConfig500JSONResponse) VisitGetGearConfigResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type putGearConfig500JSONResponse adminservice.ErrorResponse

func (response putGearConfig500JSONResponse) VisitPutGearConfigResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type getGearInfo500JSONResponse adminservice.ErrorResponse

func (response getGearInfo500JSONResponse) VisitGetGearInfoResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type refreshGear500JSONResponse adminservice.ErrorResponse

func (response refreshGear500JSONResponse) VisitRefreshGearResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type getConfig500JSONResponse gearservice.ErrorResponse

func (response getConfig500JSONResponse) VisitGetConfigResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type getInfo500JSONResponse gearservice.ErrorResponse

func (response getInfo500JSONResponse) VisitGetInfoResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type putInfo500JSONResponse gearservice.ErrorResponse

func (response putInfo500JSONResponse) VisitPutInfoResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}

type registerGear500JSONResponse serverpublic.ErrorResponse

func (response registerGear500JSONResponse) VisitRegisterGearResponse(ctx *fiber.Ctx) error {
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Status(500)
	return ctx.JSON(&response)
}
