# AgentHost GenX Wiring

This document describes the connection-level AgentHost structure.

## Core Idea

```text
PeerConn / GearConn
└── wraps the live peer connection as GenX input and output streams
    └── AgentHost
        ├── chooses the current agent for this peer connection
        ├── starts the agent in the background
        ├── reloads the agent when workspace/config changes
        └── stops the agent when the peer connection closes
```

This is not a request-scoped RPC stream. The agent is a background runtime
attached to the live peer connection.

## GenX Interfaces

```go
type Stream interface {
	Next() (*MessageChunk, error)
	Close() error
	CloseWithError(error) error
}

type Transformer interface {
	Transform(ctx context.Context, pattern string, input Stream) (Stream, error)
}
```

## Peer Connection Stream

```go
type PeerConnStream interface {
	genx.Stream
}
```

`PeerConnStream` is the adapter that turns a live `GearConn` into a GenX input
stream.

```go
type GearConnGenXStream struct {
	Conn *GearConn
}

func (s *GearConnGenXStream) Next() (*genx.MessageChunk, error)
func (s *GearConnGenXStream) Close() error
func (s *GearConnGenXStream) CloseWithError(error) error
```

Input examples:

```text
incoming text event       -> genx.Text
incoming audio frame      -> genx.Blob
incoming control event    -> genx.MessageChunk{Ctrl: ...}
peer connection closed    -> io.EOF
```

## Peer Connection Output

```go
type PeerConnOutput interface {
	ConsumeAgentOutput(context.Context, genx.Stream) error
}
```

```go
type GearConnOutput struct {
	Conn *GearConn
}

func (o *GearConnOutput) ConsumeAgentOutput(ctx context.Context, output genx.Stream) error
```

Output examples:

```text
genx.Text                 -> RPC/text event to peer
genx.Blob audio           -> GearConn audio mixer
genx.MessageChunk tool    -> tool call handling
genx.MessageChunk Ctrl    -> stream routing/control
```

## Audio Mux

Each live `GearConn` has one peer-level mixer.

```go
type PeerAudioMixer interface {
	CreateAudioTrack(...pcm.TrackOption) (pcm.Track, *pcm.TrackCtrl, error)
}
```

Each GenX audio stream should get its own mixer track.

```go
type GenXAudioMux struct {
	Mixer PeerAudioMixer
}

func (m *GenXAudioMux) AddStream(ctx context.Context, label string, stream genx.Stream) error
```

Mux shape:

```text
genx audio stream A -> mixer track A
genx audio stream B -> mixer track B
genx audio stream C -> mixer track C
                         └── GearConn mixer
                             └── stamped opus output to peer
```

`TrackCtrl` controls per-stream gain, fade, and close. The peer mixer produces a
single mixed PCM stream, and `GearConn` encodes that mixed stream to stamped
opus packets.

## Agent Runtime

The connection runtime owns the active AgentHost run for a peer connection.

```go
type AgentService struct {
	Host     *agenthost.Host
	Pattern  PatternSource
	Source   StreamSource
	Consumer StreamConsumer
}
```

```go
type PatternSource interface {
	AgentPattern(context.Context) (string, error)
}

type StreamSource interface {
	OpenAgentInput(context.Context) (genx.Stream, error)
}

type StreamConsumer interface {
	ConsumeAgentOutput(context.Context, genx.Stream) error
}
```

Runtime operations:

```go
func (s *AgentService) Reload(ctx context.Context) error
func (s *AgentService) Stop(ctx context.Context) error
```

`Reload` chooses the current workspace pattern, opens the peer connection stream,
calls `agenthost.Host.Transform`, starts output consumption in the background,
then stops the previous runtime.

## Agent Resolution

```go
type Host struct {
	Resolver    Resolver
	Registry    *Registry
	Coordinator Coordinator
}

type Resolver interface {
	Resolve(context.Context, string) (Spec, error)
}

type Factory interface {
	NewAgent(context.Context, Spec) (genx.Transformer, error)
}
```

```go
type Spec struct {
	Workspace apitypes.Workspace
	Workflow  apitypes.WorkflowDocument
	AgentType string
}
```

Pattern resolution:

```text
peer connection
└── selected workspace name
    └── workspace.workflow_name
        └── workflow
            └── agent type
```

Agent type resolution:

```text
workspace.parameters["agent_type"]
└── else workflow apiVersion group
```

## GearConn Wiring

```go
type GearConn struct {
	Conn         *giznet.Conn
	Service      *PeerService
	AgentRuntime *AgentService
}
```

Connection lifecycle:

```text
GearConn.serve()
├── serve peer service
├── serve RPC
├── serve mixed audio packets
├── start/reload AgentRuntime
└── stop AgentRuntime on close
```

## Control RPC

Peer run RPC methods are control methods for the connection-level background
runtime.

```text
peer.run.agent.get
peer.run.agent.set
peer.run.reload
peer.run.status
peer.run.stop
```

They do not carry the agent input/output stream. The peer connection itself is
the stream.

```go
type PeerRunAgent struct {
	Active  *AgentSelection `json:"active,omitempty"`
	Pending *AgentSelection `json:"pending,omitempty"`
}

type AgentSelection struct {
	WorkspaceName string `json:"workspace_name"`
}
```

`peer.run.agent.get` reads the peer's active and pending agent selection.

`peer.run.agent.set` stores the peer's pending agent selection, such as the next
`workspace_name`. It does not switch the running agent by itself.

`peer.run.reload` applies the pending agent selection and switches the
connection-level background agent runtime.
