package cmdhandler

import "context"

type Handler interface {
	Execute(context.Context, CommandLine) error
}

type HandleFunc func(context.Context, CommandLine) error

func (fn HandleFunc) Execute(ctx context.Context, commandLine CommandLine) error {
	return fn(ctx, commandLine)
}

type CommandLine struct {
	Args  []string
	Flags []string
}

func CommandLineFromArgv(argv []string) CommandLine {
	var args []string
	var flags []string
	for i, arg := range argv {
		if arg == "--" {
			args = append(args, argv[i+1:]...)
			break
		}
		if len(arg) > 1 && arg[0] == '-' {
			flags = append(flags, arg)
			continue
		}
		args = append(args, arg)
	}
	return CommandLine{
		Args:  append([]string(nil), args...),
		Flags: append([]string(nil), flags...),
	}
}
