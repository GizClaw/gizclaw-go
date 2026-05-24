package gizclaw

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
	"github.com/GizClaw/gizclaw-go/pkg/giznet/gizhttp"
)

func (s *PeerService) serveGear(conn *giznet.Conn) error {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(ctx *fiber.Ctx) error {
		base := ctx.UserContext()
		if base == nil {
			base = context.Background()
		}
		ctx.SetUserContext(gearservice.WithCallerPublicKey(base, conn.PublicKey()))
		return ctx.Next()
	})
	gearservice.RegisterHandlers(app, gearservice.NewStrictHandler(s.gear, nil))

	server := gizhttp.NewServer(conn, ServiceGear, fiberHTTPHandler(app))
	defer func() {
		_ = server.Shutdown(context.Background())
	}()
	defer func() {
		_ = conn.Close()
	}()
	return server.Serve()
}
