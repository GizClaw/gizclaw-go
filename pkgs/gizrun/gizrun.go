package gizrun

import (
	"context"
	"errors"
	"flag"
	"os"
	"strings"

	"github.com/GizClaw/gizclaw-go/pkgs/gizrun/internal/cmdhandler"
	"github.com/GizClaw/gizclaw-go/pkgs/gizrun/internal/configfile"
	"github.com/gofiber/fiber/v2"
)

var runCtx = newRunContext()

type (
	CmdHandler    = cmdhandler.Handler
	CmdHandleFunc = cmdhandler.HandleFunc
	CommandLine   = cmdhandler.CommandLine
	ConfigParser  = configfile.Parser
)

func HandleCmd(path string, handler CmdHandler) error {
	return runCtx.cmdMux.Handle(path, handler)
}

func RegisterConfigParser(name string, parser ConfigParser) {
	runCtx.configParser.Register(name, parser)
}

func Debug() *fiber.App {
	return runCtx.debugApp
}

func HTTP() *fiber.App {
	return runCtx.publicApp
}

func Config(name string) (any, bool) {
	return runCtx.configFile.Config(name)
}

func Serve() error {
	runInitHooks(initHooks.hooks)
	argv, err := stripRegisteredFlags(flag.CommandLine, os.Args[1:])
	if err != nil {
		return err
	}
	if err := runCtx.loadConfig(flag.CommandLine); err != nil {
		return err
	}
	runPostInitHooks(runCtx, postInitHooks.hooks)
	defer runExitHooks(runCtx, exitHooks.hooks)
	commandLine := cmdhandler.CommandLineFromArgv(argv)
	if err := runCtx.cmdMux.Execute(context.Background(), commandLine); err != nil {
		if errors.Is(err, cmdhandler.ErrHandlerNotFound) {
			return errors.New("gizrun: command handler not found")
		}
		return err
	}
	return nil
}

func stripRegisteredFlags(fs *flag.FlagSet, argv []string) ([]string, error) {
	if fs == nil {
		return append([]string(nil), argv...), nil
	}
	var out []string
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--" {
			out = append(out, argv[i:]...)
			break
		}
		if !isArgFlag(arg) {
			out = append(out, arg)
			continue
		}
		name, value, hasValue := splitArgFlag(arg)
		flagValue := fs.Lookup(name)
		if flagValue == nil {
			out = append(out, arg)
			continue
		}
		if !hasValue {
			if boolValue, ok := flagValue.Value.(interface{ IsBoolFlag() bool }); ok && boolValue.IsBoolFlag() {
				value = "true"
			} else {
				i++
				if i >= len(argv) {
					return nil, errors.New("flag needs an argument: -" + name)
				}
				value = argv[i]
			}
		}
		if err := fs.Set(name, value); err != nil {
			return nil, err
		}
	}
	return append([]string(nil), out...), nil
}

func isArgFlag(arg string) bool {
	return len(arg) > 1 && strings.HasPrefix(arg, "-") && arg != "--"
}

func splitArgFlag(arg string) (name, value string, hasValue bool) {
	name = strings.TrimLeft(arg, "-")
	if name, value, ok := strings.Cut(name, "="); ok {
		return name, value, true
	}
	return name, "", false
}
