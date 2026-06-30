package cmdhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/GizClaw/gizclaw-go/pkgs/trie"
)

var _ Handler = (*CmdMux)(nil)

var ErrHandlerNotFound = errors.New("cmdhandler: handler not found")

type CmdMux struct {
	trie *trie.Trie[Handler]
}

func NewMux() *CmdMux {
	return &CmdMux{trie: trie.New[Handler]()}
}

func (m *CmdMux) Handle(path string, handler Handler) error {
	if handler == nil {
		return errors.New("cmdhandler: nil handler")
	}
	path = strings.Trim(path, "/")
	return m.trie.Set(path, func(ptr *Handler, existed bool) error {
		if existed {
			return fmt.Errorf("cmdhandler: handler already registered for %s", path)
		}
		*ptr = handler
		return nil
	})
}

func (m *CmdMux) Execute(ctx context.Context, commandLine CommandLine) error {
	handler, ok := m.handler(commandLine.Args)
	if !ok {
		path := strings.Trim(strings.Join(commandLine.Args, "/"), "/")
		return fmt.Errorf("%w: %s", ErrHandlerNotFound, path)
	}
	return handler.Execute(ctx, commandLine)
}

func (m *CmdMux) handler(args []string) (Handler, bool) {
	path := strings.Trim(strings.Join(args, "/"), "/")
	ptr, ok := m.trie.Get(path)
	if !ok || ptr == nil || *ptr == nil {
		return nil, false
	}
	return *ptr, true
}
