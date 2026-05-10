import type { JSX, MutableRefObject } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { Activity, Info, MessageSquare, Mic, MicOff, Moon, Phone, PhoneOff, RadioTower, Send, Sun, Video, VideoOff, X } from "lucide-react";

import { Button } from "../../packages/components/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../packages/components/card";
import { cn } from "../../packages/components/utils";

interface RpcRequest {
  v: 1;
  id: string;
  method: string;
  params?: unknown;
}

interface RpcResponse {
  v?: 1;
  id: string;
  result?: unknown;
  error?: {
    code?: number;
    message: string;
  };
}

interface RpcLogEntry {
  id: string;
  event: string;
  detail: string;
}

interface SignalDescription {
  sdp: string;
  type: RTCSdpType;
}

type CallStatus = "Idle" | "Starting" | "Connected" | "RPC failed" | "Ended";
type Theme = "dark" | "light";

interface NerdStats {
  downlinkKbps: number;
  inboundBytes: number;
  outboundBytes: number;
  packetsLost: number;
  rttMs?: number;
  uplinkKbps: number;
}

interface RpcCommand {
  label: string;
  method: string;
  description: string;
  params: () => unknown;
}

const RPC_COMMANDS: RpcCommand[] = [
  {
    label: "Ping",
    method: "peer.ping",
    description: "Check the RPC stream and round-trip timing.",
    params: () => ({ client_send_time: Date.now() }),
  },
  {
    label: "Get Registration",
    method: "gear.registration.get",
    description: "Read this gear's registration record.",
    params: () => ({}),
  },
  {
    label: "Get Info",
    method: "gear.info.get",
    description: "Read device identity and hardware metadata.",
    params: () => ({}),
  },
  {
    label: "Get Config",
    method: "gear.config.get",
    description: "Read the assigned gear configuration.",
    params: () => ({}),
  },
  {
    label: "Get OTA",
    method: "gear.ota.get",
    description: "Read the current OTA recommendation.",
    params: () => ({}),
  },
  {
    label: "Get Runtime",
    method: "gear.runtime.get",
    description: "Read online state and transport counters.",
    params: () => ({}),
  },
];

function App(): JSX.Element {
  const [theme, setTheme] = useState<Theme>("dark");
  const [status, setStatus] = useState<CallStatus>("Idle");
  const [logs, setLogs] = useState<RpcLogEntry[]>([]);
  const [rpcSent, setRpcSent] = useState(0);
  const [rpcReceived, setRpcReceived] = useState(0);
  const [statsOpen, setStatsOpen] = useState(false);
  const [logDrawerOpen, setLogDrawerOpen] = useState(false);
  const [expandedLogID, setExpandedLogID] = useState<string | null>(null);
  const [nerdStats, setNerdStats] = useState<NerdStats>({
    downlinkKbps: 0,
    inboundBytes: 0,
    outboundBytes: 0,
    packetsLost: 0,
    uplinkKbps: 0,
  });
  const peerRef = useRef<RTCPeerConnection | null>(null);
  const localVideoRef = useRef<HTMLVideoElement | null>(null);
  const remoteVideoRef = useRef<HTMLVideoElement | null>(null);
  const remoteAudioRef = useRef<HTMLAudioElement | null>(null);
  const localStreamRef = useRef<MediaStream | null>(null);
  const remoteStreamRef = useRef<MediaStream | null>(null);
  const lastStatsSampleRef = useRef<{ bytesReceived: number; bytesSent: number; timestamp: number } | null>(null);
  const pendingRPCRef = useRef<Map<string, string>>(new Map());
  const initialRPCSentRef = useRef(false);
  const [cameraEnabled, setCameraEnabled] = useState(true);
  const [micEnabled, setMicEnabled] = useState(true);

  const appendLog = useCallback((event: string, detail: unknown) => {
    setLogs((current) => [
      {
        id: `${Date.now()}-${current.length + 1}`,
        event,
        detail: formatLogDetail(detail),
      },
      ...current,
    ].slice(0, 100));
  }, []);

  const closeCall = useCallback(() => {
    peerRef.current?.close();
    localStreamRef.current?.getTracks().forEach((track) => track.stop());
    peerRef.current = null;
    localStreamRef.current = null;
    remoteStreamRef.current = null;
    if (localVideoRef.current) {
      localVideoRef.current.srcObject = null;
    }
    if (remoteVideoRef.current) {
      remoteVideoRef.current.srcObject = null;
    }
    if (remoteAudioRef.current) {
      remoteAudioRef.current.srcObject = null;
    }
    pendingRPCRef.current.clear();
    initialRPCSentRef.current = false;
    lastStatsSampleRef.current = null;
    setCameraEnabled(true);
    setMicEnabled(true);
    setStatus("Ended");
    appendLog("call.closed", "WebRTC call closed");
  }, [appendLog]);

  const playRemoteAudio = useCallback(() => {
    const audio = remoteAudioRef.current;
    if (!audio) {
      return;
    }
    void audio.play().catch((error: unknown) => {
      appendLog("audio.play.error", error instanceof Error ? error.message : String(error));
    });
  }, [appendLog]);

  useEffect(() => {
    const peer = peerRef.current;
    if (!peer || status === "Idle" || status === "Ended") {
      lastStatsSampleRef.current = null;
      return;
    }
    const interval = window.setInterval(() => {
      const current = peerRef.current;
      if (!current || current.connectionState === "closed") {
        return;
      }
      void samplePeerStats(current, lastStatsSampleRef, setNerdStats);
    }, 1000);
    return () => window.clearInterval(interval);
  }, [status]);

  const sendRPC = useCallback((command: RpcCommand) => {
    const peer = peerRef.current;
    if (!peer || (peer.connectionState !== "connected" && peer.connectionState !== "connecting")) {
      setStatus("RPC failed");
      appendLog("rpc.error", "WebRTC connection is not ready for RPC yet");
      return;
    }
    const request: RpcRequest = {
      v: 1,
      id: crypto.randomUUID(),
      method: command.method,
      params: command.params(),
    };
    pendingRPCRef.current.set(request.id, command.method);
    setRpcSent((current) => current + 1);
    appendLog("rpc.send", request.method);

    const channel = peer.createDataChannel(`rpc:${request.id}`, { ordered: true });
    channel.onopen = () => {
      channel.send(JSON.stringify(request));
    };
    channel.onmessage = (message) => {
      try {
        const response = JSON.parse(String(message.data)) as RpcResponse;
        const method = pendingRPCRef.current.get(response.id) ?? command.method;
        pendingRPCRef.current.delete(response.id);
        setRpcReceived((current) => current + 1);
        if (response.error) {
          setStatus("RPC failed");
          appendLog("rpc.error", `${method}: ${response.error.message}`);
          return;
        }
        appendLog("rpc.response", { method, result: response.result ?? {} });
      } catch (error) {
        setStatus("RPC failed");
        appendLog("rpc.error", error instanceof Error ? error.message : String(error));
      }
    };
    channel.onerror = () => {
      setStatus("RPC failed");
      appendLog("rpc.error", `${command.method}: data channel error`);
    };
    channel.onclose = () => appendLog("rpc.close", command.method);
  }, [appendLog]);

  const startCall = useCallback(async () => {
    if (status === "Starting") {
      return;
    }
    closeCall();
    setStatus("Starting");
    setLogs([]);
    setRpcSent(0);
    setRpcReceived(0);
    initialRPCSentRef.current = false;

    const uiPeer = new RTCPeerConnection({ iceServers: [] });
    peerRef.current = uiPeer;

    const remoteStream = new MediaStream();
    remoteStreamRef.current = remoteStream;
    if (remoteVideoRef.current) {
      remoteVideoRef.current.srcObject = remoteStream;
    }
    if (remoteAudioRef.current) {
      remoteAudioRef.current.srcObject = remoteStream;
    }

    uiPeer.onconnectionstatechange = () => {
      appendLog("webrtc.state", uiPeer.connectionState);
      if (peerRef.current !== uiPeer) {
        return;
      }
      if (uiPeer.connectionState === "connected") {
        setStatus("Connected");
        if (!initialRPCSentRef.current) {
          initialRPCSentRef.current = true;
          sendRPC(RPC_COMMANDS[0]);
        }
      }
      if (uiPeer.connectionState === "failed" || uiPeer.connectionState === "disconnected") {
        setStatus("RPC failed");
      }
      if (uiPeer.connectionState === "closed") {
        setStatus("Ended");
      }
    };
    uiPeer.ontrack = (event) => {
      if (peerRef.current !== uiPeer) {
        return;
      }
      for (const track of event.streams[0]?.getTracks() ?? [event.track]) {
        if (!remoteStream.getTracks().some((current) => current.id === track.id)) {
          remoteStream.addTrack(track);
          appendLog("webrtc.track", track.kind);
        }
      }
      playRemoteAudio();
    };

    const bootstrapChannel = uiPeer.createDataChannel("rpc-bootstrap");
    bootstrapChannel.onopen = () => bootstrapChannel.close();

    try {
      const localStream = await navigator.mediaDevices.getUserMedia({
        audio: {
          autoGainControl: false,
          echoCancellation: false,
          noiseSuppression: false,
        },
        video: true,
      });
      localStreamRef.current = localStream;
      const audioSettings = localStream.getAudioTracks()[0]?.getSettings();
      appendLog("media.audio", audioSettings ?? "no audio track");
      if (localVideoRef.current) {
        localVideoRef.current.srcObject = localStream;
      }
      for (const track of localStream.getTracks()) {
        uiPeer.addTrack(track, localStream);
      }
      appendLog("media.open", "Camera and microphone tracks attached");
    } catch (error) {
      appendLog("media.error", error instanceof Error ? error.message : String(error));
    }

    try {
      const offer = await uiPeer.createOffer();
      await uiPeer.setLocalDescription(offer);
      await waitForIceGathering(uiPeer);

      const localDescription = uiPeer.localDescription;
      if (!localDescription) {
        throw new Error("missing local WebRTC offer");
      }
      const response = await fetch("/webrtc/offer", {
        body: JSON.stringify({
          sdp: localDescription.sdp,
          type: localDescription.type,
        } satisfies SignalDescription),
        headers: { "Content-Type": "application/json" },
        method: "POST",
      });
      if (!response.ok) {
        const detail = await response.text();
        throw new Error(`signaling failed: ${response.status} ${response.statusText}${detail === "" ? "" : `: ${detail.trim()}`}`);
      }
      const answer = await response.json() as SignalDescription;
      await uiPeer.setRemoteDescription(answer);
    } catch (error) {
      closeCall();
      setStatus("RPC failed");
      appendLog("webrtc.error", error instanceof Error ? error.message : String(error));
    }
  }, [appendLog, closeCall, playRemoteAudio, sendRPC, status]);

  const toggleMic = useCallback(() => {
    const next = !micEnabled;
    localStreamRef.current?.getAudioTracks().forEach((track) => {
      track.enabled = next;
    });
    setMicEnabled(next);
  }, [micEnabled]);

  const toggleCamera = useCallback(() => {
    const next = !cameraEnabled;
    localStreamRef.current?.getVideoTracks().forEach((track) => {
      track.enabled = next;
    });
    setCameraEnabled(next);
  }, [cameraEnabled]);

  const light = theme === "light";
  const peerState = peerRef.current?.connectionState ?? "closed";
  const rpcReady = status === "Connected" || peerState === "connected" || peerState === "connecting";
  const panelClass = light ? "border-slate-200 bg-white text-slate-950 shadow-xl shadow-slate-300/30" : "border-white/10 bg-white/[0.06] text-slate-50";
  const actionButtonClass = (active: boolean): string => cn(
    "rounded-full border px-3 transition",
    light
      ? "border-slate-200 bg-white text-slate-700 shadow-sm hover:bg-slate-100"
      : "border-white/10 bg-black/30 text-slate-200 hover:bg-white/10",
    active && (light ? "bg-slate-950 text-white hover:bg-slate-800" : "bg-white text-slate-950 hover:bg-slate-200"),
  );

  return (
    <main className={cn("min-h-screen transition-colors", light ? "bg-slate-100 text-slate-950" : "bg-slate-950 text-slate-50")}>
      <div className="mx-auto flex min-h-screen w-full max-w-7xl flex-col gap-4 p-4 lg:p-6">
        <header
          className={cn(
            "flex flex-wrap items-center justify-between gap-4 rounded-3xl border px-4 py-3 shadow-2xl transition-colors",
            light ? "border-slate-200 bg-white shadow-slate-300/30" : "border-white/10 bg-white/[0.05] shadow-black/20",
          )}
        >
          <div>
            <div className="text-xl font-semibold tracking-tight">GizClaw Play</div>
            <div className={cn("text-sm", light ? "text-slate-500" : "text-slate-400")}>WebRTC Play surface with RPC data-channel controls</div>
          </div>
          <div className="flex flex-wrap items-center justify-end gap-2">
            <ThemeSelector onThemeChange={setTheme} theme={theme} />
            <Button aria-label="Toggle call stats" className={actionButtonClass(statsOpen)} onClick={() => setStatsOpen((open) => !open)} type="button" variant="ghost">
              <Info className="size-4" />
              Info
            </Button>
            <Button aria-label="Logs" className={actionButtonClass(logDrawerOpen)} onClick={() => setLogDrawerOpen((open) => !open)} type="button" variant="ghost">
              <MessageSquare className="size-4" />
              Logs
              <span className={cn("rounded-full px-2 py-0.5 text-xs", light ? "bg-slate-100" : "bg-white/10")}>{logs.length}</span>
            </Button>
          </div>
        </header>

        <div className="grid min-h-0 flex-1 gap-4 lg:grid-cols-[minmax(0,1fr)_23rem]">
          <section className="flex min-h-0 flex-col gap-4">
            <div
              className={cn(
                "relative aspect-video min-h-[22rem] overflow-hidden rounded-3xl border bg-black shadow-2xl",
                light ? "border-slate-200 shadow-slate-300/60" : "border-white/25 shadow-black/50 ring-1 ring-white/10",
              )}
            >
              <div className="absolute inset-0 bg-[radial-gradient(circle_at_30%_25%,rgba(34,211,238,0.26),transparent_34%),radial-gradient(circle_at_75%_70%,rgba(16,185,129,0.18),transparent_30%),linear-gradient(135deg,#020617,#0f172a_50%,#020617)]" />
              <video autoPlay className="absolute inset-0 h-full w-full bg-black object-cover" playsInline ref={remoteVideoRef} />
              <audio autoPlay ref={remoteAudioRef} />
              <div className="absolute right-4 top-16 w-28 overflow-hidden rounded-2xl border border-white/15 bg-black shadow-2xl shadow-black/50 sm:w-36">
                <div className="aspect-square">
                  <video autoPlay className={cn("h-full w-full bg-black object-cover", !cameraEnabled && "opacity-20")} muted playsInline ref={localVideoRef} />
                </div>
                <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/70 to-transparent px-3 py-2 text-xs text-slate-200">You</div>
              </div>
              <div className="absolute left-4 top-4 rounded-2xl border border-white/10 bg-black/55 px-4 py-3 text-sm text-slate-100 shadow-2xl backdrop-blur">
                <div className="font-semibold">WebRTC Play</div>
                <div>Status: {status}</div>
                <div>RPC up/down: {rpcSent}/{rpcReceived}</div>
              </div>
              {statsOpen ? (
                <div className="absolute right-4 top-4 w-72 rounded-2xl border border-white/15 bg-black/70 p-3 text-xs text-slate-200 shadow-2xl shadow-black/50 backdrop-blur">
                  <div className="mb-2 font-semibold text-slate-100">Stats for nerds</div>
                  <div className="space-y-1">
                    <StatusRow label="Call" value={peerState} />
                    <StatusRow label="Peer" value={peerState} />
                    <StatusRow label="Downlink" value={`${nerdStats.downlinkKbps.toFixed(1)} kbps`} />
                    <StatusRow label="Uplink" value={`${nerdStats.uplinkKbps.toFixed(1)} kbps`} />
                    <StatusRow label="RTT" value={nerdStats.rttMs === undefined ? "-" : `${nerdStats.rttMs.toFixed(0)} ms`} />
                    <StatusRow label="Packets lost" value={String(nerdStats.packetsLost)} />
                    <StatusRow label="RPC sent" value={String(rpcSent)} />
                    <StatusRow label="RPC received" value={String(rpcReceived)} />
                    <StatusRow label="RPC pending" value={String(pendingRPCRef.current.size)} />
                  </div>
                </div>
              ) : null}
              <div className={cn("pointer-events-none absolute inset-0 flex items-center justify-center", status === "Connected" && "hidden")}>
                <div className="relative flex size-44 items-center justify-center rounded-full border border-cyan-300/40 bg-cyan-300/10 shadow-2xl shadow-cyan-950/50">
                  <div className="absolute inset-4 rounded-full border border-cyan-200/20" />
                  <RadioTower className={cn("size-16", status === "Connected" ? "text-cyan-300" : "text-slate-400")} />
                </div>
              </div>
              <div className="absolute inset-x-0 bottom-0 flex flex-wrap items-center justify-center gap-3 bg-gradient-to-t from-black/85 to-transparent p-6">
                <Button aria-label={micEnabled ? "Mute microphone" : "Unmute microphone"} className="size-12 rounded-full border border-white/10 bg-white/15 text-white backdrop-blur hover:bg-white/25" onClick={toggleMic} type="button" variant="ghost">
                  {micEnabled ? <Mic className="size-5" /> : <MicOff className="size-5" />}
                </Button>
                <Button aria-label="Start call from video surface" className="rounded-full px-6" disabled={status === "Starting"} onClick={() => void startCall()} type="button">
                  <Phone className="size-4" />
                  Start Video Call
                </Button>
                <Button aria-label="End call from video surface" className="rounded-full border-white/10 bg-white/15 px-6 text-white hover:bg-white/25" onClick={closeCall} type="button" variant="ghost">
                  <PhoneOff className="size-4" />
                  End Call
                </Button>
                <Button aria-label={cameraEnabled ? "Turn camera off" : "Turn camera on"} className="size-12 rounded-full border border-white/10 bg-white/15 text-white backdrop-blur hover:bg-white/25" onClick={toggleCamera} type="button" variant="ghost">
                  {cameraEnabled ? <Video className="size-5" /> : <VideoOff className="size-5" />}
                </Button>
              </div>
            </div>

          </section>

          <aside className="flex min-h-0 flex-col gap-4">
            <Card className={panelClass}>
              <CardHeader>
                <CardTitle>Controls</CardTitle>
                <CardDescription className={cn(light ? "text-slate-500" : "text-slate-300")}>Start the local WebRTC call and tunnel GizClaw RPC over the `rpc` data channel.</CardDescription>
              </CardHeader>
              <CardContent className="grid gap-3">
                <Button className="w-full" disabled={status === "Starting"} onClick={() => void startCall()} type="button">
                  <Phone className="size-4" />
                  Start Video Call
                </Button>
                <Button
                  className={cn(
                    "w-full",
                    light ? "border-slate-200 bg-white text-slate-700 hover:bg-slate-100" : "border-white/10 bg-white/10 text-slate-100 hover:bg-white/20",
                  )}
                  onClick={closeCall}
                  type="button"
                  variant="outline"
                >
                  <PhoneOff className="size-4" />
                  End Call
                </Button>
              </CardContent>
            </Card>

            <Card className={panelClass}>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Send className="size-4" />
                  RPC Commands
                </CardTitle>
                <CardDescription className={cn(light ? "text-slate-500" : "text-slate-300")}>Send gear-service RPC methods through the active data channel.</CardDescription>
              </CardHeader>
              <CardContent className="grid gap-2">
                {RPC_COMMANDS.map((command) => (
                  <Button
                    aria-label={command.label}
                    className={cn(
                      "h-auto justify-start gap-3 px-3 py-2 text-left",
                      light ? "border-slate-200 bg-white hover:bg-slate-100" : "border-white/10 bg-black/20 text-slate-100 hover:bg-white/10",
                    )}
                    disabled={!rpcReady}
                    key={command.method}
                    onClick={() => sendRPC(command)}
                    type="button"
                    variant="outline"
                  >
                    <span className="flex flex-col items-start gap-0.5">
                      <span>{command.label}</span>
                      <span className="font-mono text-xs text-slate-500">{command.method}</span>
                      <span className={cn("text-xs font-normal", light ? "text-slate-500" : "text-slate-400")}>{command.description}</span>
                    </span>
                  </Button>
                ))}
              </CardContent>
            </Card>

            <Card className={panelClass}>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Activity className="size-4" />
                  Stats
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                <InfoRow label="WebRTC status" light={light} value={status} />
                <InfoRow label="Peer connection" light={light} value={peerState} />
                <InfoRow label="RPC sent" light={light} value={String(rpcSent)} />
                <InfoRow label="RPC received" light={light} value={String(rpcReceived)} />
              </CardContent>
            </Card>
          </aside>
        </div>

        <RPCLogDrawer
          expandedID={expandedLogID}
          logs={logs}
          onClose={() => {
            setLogDrawerOpen(false);
            setExpandedLogID(null);
          }}
          onToggleExpanded={setExpandedLogID}
          open={logDrawerOpen}
          theme={theme}
        />
      </div>
    </main>
  );
}

function InfoRow({ label, light, value }: { label: string; light?: boolean; value: string }): JSX.Element {
  return (
    <div className="flex items-start justify-between gap-4">
      <span className={light ? "text-slate-500" : "text-slate-300"}>{label}</span>
      <span className={cn("text-right font-medium", light ? "text-slate-950" : "text-slate-50")}>{value}</span>
    </div>
  );
}

function ThemeSelector({ onThemeChange, theme }: { onThemeChange: (theme: Theme) => void; theme: Theme }): JSX.Element {
  const light = theme === "light";
  return (
    <div aria-label="Theme selector" className={cn("flex rounded-full border p-1 text-sm", light ? "border-slate-200 bg-white shadow-sm" : "border-white/10 bg-black/30")}>
      <button
        className={cn(
          "flex items-center gap-1 rounded-full px-3 py-1.5 transition",
          theme === "dark" ? "bg-slate-950 text-white" : "text-slate-600 hover:bg-slate-100",
        )}
        onClick={() => onThemeChange("dark")}
        type="button"
      >
        <Moon className="size-4" />
        Dark
      </button>
      <button
        className={cn(
          "flex items-center gap-1 rounded-full px-3 py-1.5 transition",
          theme === "light" ? "bg-slate-950 text-white" : "text-slate-300 hover:bg-white/10",
        )}
        onClick={() => onThemeChange("light")}
        type="button"
      >
        <Sun className="size-4" />
        Light
      </button>
    </div>
  );
}

function RPCLogTable({
  expandedID,
  logs,
  onToggleExpanded,
  theme,
}: {
  expandedID: string | null;
  logs: RpcLogEntry[];
  onToggleExpanded: (id: string | null) => void;
  theme: Theme;
}): JSX.Element {
  const light = theme === "light";
  return (
    <div className={cn("h-[min(72vh,42rem)] overflow-auto rounded-xl border text-sm", light ? "border-slate-200 bg-white" : "border-white/10 bg-slate-950/95")}>
      <div className={cn("sticky top-0 z-10 grid grid-cols-[11rem_minmax(0,1fr)] gap-x-6 border-b px-4 py-2 text-xs font-semibold uppercase tracking-wide", light ? "border-slate-200 bg-white text-slate-500" : "border-white/10 bg-slate-950 text-slate-400")}>
        <div>Event</div>
        <div>Summary</div>
      </div>
      {logs.length === 0 ? (
        <div className="px-4 py-4 text-slate-500">No RPC events yet.</div>
      ) : (
        <div className={cn("divide-y", light ? "divide-slate-100" : "divide-white/10")}>
          {logs.map((entry) => {
            const expanded = expandedID === entry.id;
            return (
              <div className={cn("transition", light ? "hover:bg-slate-50" : "hover:bg-white/[0.04]")} key={entry.id}>
                <button className="block w-full px-4 py-3 text-left" onClick={() => onToggleExpanded(expanded ? null : entry.id)} type="button">
                  <div className="grid grid-cols-[11rem_minmax(0,1fr)] gap-x-6">
                    <div className={cn("break-words font-semibold", light ? "text-slate-800" : "text-slate-100")}>{entry.event}</div>
                    <div className={cn("min-w-0 truncate font-mono text-xs", light ? "text-slate-700" : "text-slate-200")} title={entry.detail}>
                      {entry.detail}
                    </div>
                  </div>
                </button>
                {expanded ? (
                  <div className="px-4 pb-4">
                    <div className={cn("ml-[calc(11rem+1.5rem)] rounded-lg border p-3", light ? "border-slate-200 bg-slate-50" : "border-white/10 bg-black/35")}>
                      <div className={cn("mb-2 text-xs font-semibold uppercase tracking-wide", light ? "text-slate-500" : "text-slate-400")}>Detail</div>
                      <pre className={cn("whitespace-pre-wrap break-words font-mono text-xs leading-5", light ? "text-slate-800" : "text-slate-100")}>
                        {formatExpandedLogDetail(entry.detail)}
                      </pre>
                    </div>
                  </div>
                ) : null}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function RPCLogDrawer({
  expandedID,
  logs,
  onClose,
  onToggleExpanded,
  open,
  theme,
}: {
  expandedID: string | null;
  logs: RpcLogEntry[];
  onClose: () => void;
  onToggleExpanded: (id: string | null) => void;
  open: boolean;
  theme: Theme;
}): JSX.Element | null {
  if (!open) {
    return null;
  }
  const light = theme === "light";
  return (
    <div className="fixed bottom-4 left-1/2 z-50 w-[min(88rem,calc(100vw-2rem))] -translate-x-1/2">
      <div className={cn("rounded-2xl border p-3 shadow-2xl", light ? "border-slate-200 bg-white shadow-slate-400/30" : "border-white/10 bg-slate-950 shadow-black/70")}>
        <div className="mb-3 flex items-center justify-between gap-3">
          <div>
            <div className="font-semibold">RPC Log</div>
            <div className="text-xs text-slate-500">Click a row to expand full WebRTC data-channel details</div>
          </div>
          <button
            aria-label="Close RPC logs"
            className={cn(
              "flex size-8 items-center justify-center rounded-full border transition",
              light ? "border-slate-200 bg-white text-slate-700 hover:bg-slate-100" : "border-white/10 bg-black/30 text-slate-200 hover:bg-white/10",
            )}
            onClick={onClose}
            type="button"
          >
            <X className="size-4" />
          </button>
        </div>
        <RPCLogTable expandedID={expandedID} logs={logs} onToggleExpanded={onToggleExpanded} theme={theme} />
      </div>
    </div>
  );
}

function StatusRow({ label, value }: { label: string; value: string }): JSX.Element {
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border border-white/10 bg-black/25 px-3 py-2">
      <span className="text-slate-400">{label}</span>
      <span className="font-medium text-slate-100">{value}</span>
    </div>
  );
}

function formatLogDetail(detail: unknown): string {
  if (typeof detail === "string") {
    return detail;
  }
  const encoded = JSON.stringify(detail);
  return encoded === undefined ? String(detail) : encoded;
}

function formatExpandedLogDetail(detail: string): string {
  try {
    return JSON.stringify(JSON.parse(detail), null, 2);
  } catch {
    return detail;
  }
}

async function samplePeerStats(
  peer: RTCPeerConnection,
  lastSampleRef: MutableRefObject<{ bytesReceived: number; bytesSent: number; timestamp: number } | null>,
  setStats: (stats: NerdStats) => void,
): Promise<void> {
  const report = await peer.getStats();
  let bytesReceived = 0;
  let bytesSent = 0;
  let packetsLost = 0;
  let rttMs: number | undefined;

  report.forEach((raw) => {
    const item = raw as RTCStats & {
      bytesReceived?: number;
      bytesSent?: number;
      currentRoundTripTime?: number;
      packetsLost?: number;
      selected?: boolean;
      state?: string;
    };
    if (item.type === "inbound-rtp") {
      bytesReceived += item.bytesReceived ?? 0;
      packetsLost += item.packetsLost ?? 0;
    }
    if (item.type === "outbound-rtp") {
      bytesSent += item.bytesSent ?? 0;
    }
    if (item.type === "candidate-pair" && (item.selected === true || item.state === "succeeded")) {
      if (typeof item.currentRoundTripTime === "number") {
        rttMs = item.currentRoundTripTime * 1000;
      }
    }
  });

  const now = performance.now();
  const previous = lastSampleRef.current;
  let downlinkKbps = 0;
  let uplinkKbps = 0;
  if (previous && now > previous.timestamp) {
    const seconds = (now - previous.timestamp) / 1000;
    downlinkKbps = ((bytesReceived - previous.bytesReceived) * 8) / seconds / 1000;
    uplinkKbps = ((bytesSent - previous.bytesSent) * 8) / seconds / 1000;
  }
  lastSampleRef.current = { bytesReceived, bytesSent, timestamp: now };
  setStats({
    downlinkKbps: Math.max(0, downlinkKbps),
    inboundBytes: bytesReceived,
    outboundBytes: bytesSent,
    packetsLost,
    rttMs,
    uplinkKbps: Math.max(0, uplinkKbps),
  });
}

async function waitForIceGathering(peer: RTCPeerConnection): Promise<void> {
  if (peer.iceGatheringState === "complete") {
    return;
  }
  await new Promise<void>((resolve) => {
    const handleStateChange = (): void => {
      if (peer.iceGatheringState === "complete") {
        peer.removeEventListener("icegatheringstatechange", handleStateChange);
        resolve();
      }
    };
    peer.addEventListener("icegatheringstatechange", handleStateChange);
  });
}

const root = document.querySelector<HTMLElement>("#app");

if (root === null) {
  throw new Error("missing #app root");
}

createRoot(root).render(<App />);
