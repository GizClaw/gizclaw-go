package adminui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/GizClaw/gizclaw-go/cmd/ui/internal/static"
)

//go:generate npm exec --prefix .. --package @tailwindcss/cli -- tailwindcss -i ./styles.css -o ./dist/app.css --minify
//go:generate npm exec --prefix .. --package esbuild -- esbuild index.html app.tsx --bundle --format=esm --platform=browser --target=esnext --jsx=automatic --loader:.html=copy --outdir=dist --entry-names=[name] --minify

// distFS holds generated admin UI assets.
//
//go:embed dist/*
var distFS embed.FS

func FS() fs.FS {
	return subFS(distFS, "dist")
}

func Handler() http.Handler {
	return static.Handler(FS())
}

func subFS(root fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(root, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
