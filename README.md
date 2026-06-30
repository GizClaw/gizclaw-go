# gizclaw-go

[![CI](https://github.com/GizClaw/gizclaw-go/actions/workflows/ci.yml/badge.svg)](https://github.com/GizClaw/gizclaw-go/actions/workflows/ci.yml)
[![CodeQL](https://github.com/GizClaw/gizclaw-go/actions/workflows/codeql.yml/badge.svg)](https://github.com/GizClaw/gizclaw-go/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/GizClaw/gizclaw-go)](https://goreportcard.com/report/github.com/GizClaw/gizclaw-go)

`gizclaw-go` is the Go implementation of the GizClaw server, CLI, store layer, and agent/runtime packages.

## Layout

- `cmd/`: CLI entrypoint and command implementations
- `pkgs/store/`: storage primitives such as KV, graph, filesystem, and vector stores
- `pkgs/agent/`: agent-side runtime packages such as `embed`, `memory`, `ncnn`, and `recall`
- `pkgs/genx/`: model/generation abstractions and integrations
- `examples/`: runnable examples; each `main.go` example directory is its own Go module
- `tests/`: end-to-end, benchmark, and scenario-driven tests

## Development

```bash
go test ./...
```

The GitHub Actions workflow in `.github/workflows/ci.yml` currently runs `go test -count=1 ./...` on pushes to `main`, pull requests, and manual dispatch across Linux, macOS, and Windows.

## Benchmarks

There are two main benchmark entry points:

- `tests/giznet-e2e/benchmark`: public-facing `giznet` API benchmarks such as datagram write/read, stream echo, and HTTP round-trip
- `pkgs/giznet/internal/benchmark`: internal transport-stack benchmarks for `KCP`, `Noise`, and `KCP over Noise`

Run the public `giznet` benchmarks with:

```bash
go test ./tests/giznet-e2e/benchmark -run '^$' -bench . -benchmem
```

Run the internal `giznet` benchmark suite directly with:

```bash
go test ./pkgs/giznet/internal/benchmark -run '^$' -bench . -benchmem
```

Useful scoped runs for the internal suite:

```bash
# One benchmark family
go test ./pkgs/giznet/internal/benchmark -run '^$' -bench 'BenchmarkNet_KCP_' -benchmem

# In-process packet loss simulation
BENCH_NET_LOSS_RATES=0,0.01,0.05 go test ./pkgs/giznet/internal/benchmark -run '^$' -bench 'BenchmarkNet_KCPOverNoise_Throughput' -benchmem

# Fixed smoke profile via the helper runner
./pkgs/giznet/internal/benchmark/run.sh --profile clean --scale smoke --no-system-netem
```

For the full transport benchmark guide, matrix options, and `run.sh` profiles, see `pkgs/giznet/internal/benchmark/README.md`.

## Docs And Skills

- GizClaw CLI skill: `skills/gizclaw-cli/SKILL.md`
- GenX example: `examples/genx/README.md`
