import { useCallback, useEffect, useMemo, useState } from "react";
import { Ban, Check, ChevronLeft, RefreshCw, Save, Trash2 } from "lucide-react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { expectData, toMessage } from "../../components/api";
import { Badge } from "../../components/badge";
import { Button } from "../../components/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/select";
import { Skeleton } from "../../components/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../components/tabs";

import {
  approvePeer,
  blockPeer,
  deletePeer,
  getResource,
  putPeerConfig,
  refreshPeer,
  type Configuration,
  type GearRole,
  type Resource,
} from "@gizclaw/adminservice";

import { ResourceCliPanel } from "../../components/ResourceCliPanel";
import { DetailBlock } from "../../components/detail-block";
import { ErrorBanner, NoticeBanner } from "../../components/banners";
import { EmptyState } from "../../components/empty-state";
import { FormField } from "../../components/form-field";
import { PageBreadcrumb } from "../../components/page-breadcrumb";
import { StatusBadge } from "../../components/status-badge";
import { usePeerDetail } from "../../hooks/usePeerDetail";
import { formatDate, formatShortKey, peerTitle } from "../../lib/format";

const FIRMWARE_CHANNEL_OPTIONS = ["stable", "beta", "testing"] as const;

export function PeerDetailPage(): JSX.Element {
  const params = useParams();
  const navigate = useNavigate();
  const rawKey = params.publicKey ?? "";
  const publicKey = useMemo(() => {
    try {
      return decodeURIComponent(rawKey);
    } catch {
      return rawKey;
    }
  }, [rawKey]);

  const detail = usePeerDetail(publicKey === "" ? undefined : publicKey);
  const [peerNotice, setPeerNotice] = useState<{ message: string; tone: "error" | "success" } | null>(null);
  const [peerActionBusy, setPeerActionBusy] = useState<string | null>(null);
  const [approveRole, setApproveRole] = useState<GearRole>("gear");
  const [configChannel, setConfigChannel] = useState("stable");
  const [peerConfigResource, setPeerConfigResource] = useState<Resource | null>(null);

  const registration = detail.data?.registration ?? null;
  const isBlocked = registration?.status === "blocked";
  const isActive = registration?.status === "active";
  const isApproved = isActive && registration?.role !== "unspecified";

  useEffect(() => {
    const nextChannel = detail.data?.config?.firmware?.channel ?? "stable";
    setConfigChannel(FIRMWARE_CHANNEL_OPTIONS.includes(nextChannel as (typeof FIRMWARE_CHANNEL_OPTIONS)[number]) ? nextChannel : "stable");
    if (detail.data?.registration?.role && detail.data.registration.role !== "unspecified") {
      setApproveRole(detail.data.registration.role);
    }
  }, [detail.data?.config?.firmware?.channel, detail.data?.registration?.role]);

  const loadPeerConfigResource = useCallback(async () => {
    if (publicKey === "") {
      setPeerConfigResource(null);
      return;
    }
    try {
      const resource = await expectData(getResource({ path: { kind: "PeerConfig", name: publicKey } }));
      setPeerConfigResource(resource);
    } catch {
      setPeerConfigResource(null);
    }
  }, [publicKey]);

  useEffect(() => {
    void loadPeerConfigResource();
  }, [loadPeerConfigResource]);

  const runPeerAction = useCallback(async (name: string, action: () => Promise<void>, successMessage: string) => {
    setPeerActionBusy(name);
    setPeerNotice(null);
    try {
      await action();
      setPeerNotice({ message: successMessage, tone: "success" });
    } catch (error) {
      setPeerNotice({ message: toMessage(error), tone: "error" });
    } finally {
      setPeerActionBusy(null);
    }
  }, []);

  const handleApprove = useCallback(async () => {
    if (publicKey === "") {
      return;
    }
    const nextRole = approveRole;
    await runPeerAction(
      "approve",
      async () => {
        await expectData(
          approvePeer({
            body: { role: nextRole },
            path: { publicKey },
          }),
        );
        await detail.reload();
      },
      isApproved ? `Peer role saved as ${nextRole}.` : `Peer approved as ${nextRole}.`,
    );
  }, [approveRole, detail, isApproved, publicKey, runPeerAction]);

  const handleUnblock = useCallback(async () => {
    if (publicKey === "") {
      return;
    }
    const nextRole = approveRole;
    await runPeerAction(
      "unblock",
      async () => {
        await expectData(
          approvePeer({
            body: { role: nextRole },
            path: { publicKey },
          }),
        );
        await detail.reload();
      },
      `Peer restored as ${nextRole}.`,
    );
  }, [approveRole, detail, publicKey, runPeerAction]);

  const handleBlock = useCallback(async () => {
    if (publicKey === "") {
      return;
    }
    await runPeerAction(
      "block",
      async () => {
        await expectData(blockPeer({ path: { publicKey } }));
        await detail.reload();
      },
      "Peer blocked.",
    );
  }, [detail, publicKey, runPeerAction]);

  const handleRefreshPeer = useCallback(async () => {
    if (publicKey === "") {
      return;
    }
    await runPeerAction(
      "refresh",
      async () => {
        await expectData(refreshPeer({ path: { publicKey } }));
        await detail.reload();
      },
      "Peer refreshed.",
    );
  }, [detail, publicKey, runPeerAction]);

  const handleDeletePeer = useCallback(async () => {
    if (publicKey === "") {
      return;
    }
    await runPeerAction(
      "delete",
      async () => {
        await expectData(deletePeer({ path: { publicKey } }));
        navigate("/peers");
      },
      "Peer deleted.",
    );
  }, [navigate, publicKey, runPeerAction]);

  const handleSaveChannel = useCallback(async () => {
    if (publicKey === "") {
      return;
    }
    await runPeerAction(
      "config",
      async () => {
        const nextConfig: Configuration = {
          ...(detail.data?.config ?? {}),
          firmware: {
            ...(detail.data?.config?.firmware ?? {}),
            channel: configChannel,
          },
        };
        await expectData(
          putPeerConfig({
            body: nextConfig,
            path: { publicKey },
          }),
        );
        await detail.reload();
        await loadPeerConfigResource();
      },
      `Desired channel updated to ${configChannel}.`,
    );
  }, [configChannel, detail.data?.config, detail, loadPeerConfigResource, publicKey, runPeerAction]);

  if (publicKey === "") {
    return <EmptyState description="Missing peer public key in the URL." title="Invalid route" />;
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb
        items={[
          { href: "/overview", label: "Overview" },
          { href: "/peers", label: "Peers" },
          { label: formatShortKey(publicKey) },
        ]}
      />

      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-2">
          <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Peers</div>
          <h1 className="text-3xl font-semibold tracking-tight">{registration ? peerTitle(detail.data?.info, registration.public_key) : "Peer"}</h1>
          <p className="max-w-3xl text-sm leading-6 text-muted-foreground lg:text-base break-all">{publicKey}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button asChild size="sm" variant="outline">
            <Link to="/peers">
              <ChevronLeft className="size-4" />
              Back to list
            </Link>
          </Button>
          <Button className="min-w-fit shrink-0 whitespace-nowrap" onClick={() => void detail.reload()} size="sm" variant="outline">
            <span className="inline-flex items-center gap-2 whitespace-nowrap">
              <RefreshCw className="size-4" />
              Reload
            </span>
          </Button>
          {registration ? <StatusBadge status={registration.status} /> : null}
        </div>
      </div>

      {detail.loading ? (
        <div className="space-y-4">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      ) : detail.error !== "" ? (
        <ErrorBanner message={detail.error} />
      ) : registration === null ? (
        <EmptyState description="This peer could not be loaded." title="Not found" />
      ) : (
        <div className="space-y-4">
          {peerNotice !== null ? <NoticeBanner message={peerNotice.message} tone={peerNotice.tone} /> : null}

          <div className="flex min-w-0 flex-col gap-3 rounded-xl border bg-muted/30 p-4 lg:flex-row lg:items-center lg:justify-between">
            <div className="min-w-0 space-y-1">
              <div className="text-base font-semibold">{peerTitle(detail.data?.info, registration.public_key)}</div>
              <div className="text-sm text-muted-foreground break-all">{registration.public_key}</div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button disabled={peerActionBusy !== null} onClick={() => void handleRefreshPeer()} size="sm" type="button" variant="outline">
                <RefreshCw className="size-4" />
                Refresh Peer
              </Button>
              <Badge variant="outline">{registration.role}</Badge>
              {registration.auto_registered ? <Badge variant="secondary">Auto Registered</Badge> : null}
              {detail.data?.runtime?.online ? <Badge variant="success">Online</Badge> : <Badge variant="outline">Offline</Badge>}
            </div>
          </div>

          <Tabs className="space-y-4" defaultValue="info">
            <TabsList className="grid h-auto w-full grid-cols-3 lg:w-[26rem]">
              <TabsTrigger value="info">Info</TabsTrigger>
              <TabsTrigger value="edit">Edit</TabsTrigger>
              <TabsTrigger value="cli">CLI</TabsTrigger>
            </TabsList>

            <TabsContent className="space-y-4" value="info">
              <div className="grid gap-4 xl:grid-cols-2">
                <DetailBlock
                  items={[
                    ["Name", detail.data?.info?.name],
                    ["Serial", detail.data?.info?.sn],
                    ["Manufacturer", detail.data?.info?.hardware?.manufacturer],
                    ["Model", detail.data?.info?.hardware?.model],
                    ["Revision", detail.data?.info?.hardware?.hardware_revision],
                    ["Depot", detail.data?.info?.hardware?.depot],
                    ["Firmware", detail.data?.info?.hardware?.firmware_semver],
                  ]}
                  title="Peer Info"
                />
                <DetailBlock
                  items={[
                    ["Public Key", registration.public_key],
                    ["Role", registration.role],
                    ["Status", registration.status],
                    ["Auto registered", registration.auto_registered ? "Yes" : "No"],
                    ["Created", registration.created_at],
                    ["Approved", registration.approved_at],
                    ["Updated", registration.updated_at],
                  ]}
                  title="Registration"
                />
                <DetailBlock
                  items={[
                    ["Channel", detail.data?.config?.firmware?.channel],
                    ["Resource kind", "PeerConfig"],
                    ["Resource name", registration.public_key],
                    ["Certifications", String(detail.data?.config?.certifications?.length ?? 0)],
                  ]}
                  title="Configuration"
                />
                <DetailBlock
                  items={[
                    ["Online", detail.data?.runtime?.online ? "Yes" : "No"],
                    ["Last Seen", formatDate(detail.data?.runtime?.last_seen_at)],
                    ["Last Address", detail.data?.runtime?.last_addr],
                    ["RX Bytes", formatBytes(detail.data?.runtime?.rx_bytes)],
                    ["TX Bytes", formatBytes(detail.data?.runtime?.tx_bytes)],
                  ]}
                  title="Runtime"
                />
              </div>

              <div className="grid gap-4 xl:grid-cols-2">
                <Card className="min-w-0">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base">Certifications</CardTitle>
                    <CardDescription>Attached compliance metadata.</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    {detail.data?.config?.certifications?.length ? (
                      detail.data.config.certifications.map((certification, index) => (
                        <div className="rounded-lg border bg-background px-3 py-2 text-sm" key={`${certification.id ?? "cert"}-${index}`}>
                          <div className="font-medium">{certification.id ?? "Unknown ID"}</div>
                          <div className="text-muted-foreground">
                            {certification.type ?? "type"} • {certification.authority_name ?? certification.authority ?? "authority"}
                          </div>
                        </div>
                      ))
                    ) : (
                      <EmptyState description="No certifications are attached to this peer yet." title="No certifications" />
                    )}
                  </CardContent>
                </Card>

                <Card className="min-w-0">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base">OTA Summary</CardTitle>
                    <CardDescription>Latest OTA state reported for this peer.</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <pre className="max-h-80 min-w-0 overflow-x-auto rounded-lg border bg-muted/50 p-4 text-xs leading-6 text-foreground">
                      {JSON.stringify(detail.data?.ota ?? null, null, 2)}
                    </pre>
                  </CardContent>
                </Card>
              </div>

              <Card className="min-w-0">
                <CardHeader className="pb-3">
                  <CardTitle className="text-base">Raw Detail</CardTitle>
                  <CardDescription>Combined registration, info, config, runtime, and OTA payloads.</CardDescription>
                </CardHeader>
                <CardContent className="pt-6">
                  <pre className="max-h-[32rem] min-w-0 overflow-x-auto rounded-lg border bg-muted/50 p-4 text-xs leading-6 text-foreground">
                    {JSON.stringify(detail.data, null, 2)}
                  </pre>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent className="space-y-4" value="edit">
              <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base">Peer Actions</CardTitle>
                    <CardDescription>Approve, restore, block, refresh, or reset this peer registration.</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <FormField
                      description={
                        isBlocked
                          ? "Choose the role to assign when this peer is restored."
                          : "Choose the role to assign when this peer moves into service, or block it from this same flow."
                      }
                      label={isBlocked ? "Restore role" : "Approval role"}
                    >
                      <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto] md:items-end">
                        <Select onValueChange={(value) => setApproveRole(value as GearRole)} value={approveRole}>
                          <SelectTrigger id="approve-role">
                            <SelectValue placeholder="Select role" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="gear">gear</SelectItem>
                            <SelectItem value="server">server</SelectItem>
                            <SelectItem value="admin">admin</SelectItem>
                          </SelectContent>
                        </Select>
                        <div className="flex flex-wrap gap-2">
                          {isBlocked ? (
                            <Button className="w-full md:w-auto" disabled={peerActionBusy !== null} onClick={() => void handleUnblock()} type="button">
                              <Check className="size-4" />
                              Unblock
                            </Button>
                          ) : (
                            <>
                              <Button className="w-full md:w-auto" disabled={peerActionBusy !== null} onClick={() => void handleApprove()} type="button">
                                <Check className="size-4" />
                                {isApproved ? "Save Role" : "Approve"}
                              </Button>
                              <Button
                                className="w-full md:w-auto"
                                disabled={peerActionBusy !== null}
                                onClick={() => void handleBlock()}
                                type="button"
                                variant="outline"
                              >
                                <Ban className="size-4" />
                                Block
                              </Button>
                            </>
                          )}
                        </div>
                      </div>
                    </FormField>

                    <div className="space-y-3 rounded-lg border bg-muted/20 p-4">
                      <div className="space-y-1">
                        <div className="text-sm font-medium">Registration reset</div>
                        <p className="text-sm leading-6 text-muted-foreground">Reset the peer registration back to the unapproved state.</p>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        <Button disabled={peerActionBusy !== null} onClick={() => void handleDeletePeer()} type="button" variant="outline">
                          <Trash2 className="size-4" />
                          Reset
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base">Firmware Policy</CardTitle>
                    <CardDescription>Set the desired firmware channel for this peer.</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <FormField description="This controls which release stream the peer should follow." label="Desired channel">
                      <Select onValueChange={setConfigChannel} value={configChannel}>
                        <SelectTrigger id="peer-channel">
                          <SelectValue placeholder="Select channel" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="stable">stable</SelectItem>
                          <SelectItem value="beta">beta</SelectItem>
                          <SelectItem value="testing">testing</SelectItem>
                        </SelectContent>
                      </Select>
                    </FormField>
                    <div className="flex justify-end border-t pt-4">
                      <Button disabled={peerActionBusy !== null} onClick={() => void handleSaveChannel()} type="button">
                        <Save className="size-4" />
                        Save Channel
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </TabsContent>

            <TabsContent className="space-y-4" value="cli">
              <ResourceCliPanel
                commands={peerCliCommands(registration.public_key, registration.role, configChannel)}
                resource={peerConfigResource}
                resourceDescription="JSON returned by the resource API and accepted by admin apply. PeerConfig manages desired peer configuration."
                resourceTitle="PeerConfig Resource Spec"
              />
            </TabsContent>
          </Tabs>
        </div>
      )}
    </div>
  );
}

function formatBytes(value: number | undefined): string {
  if (value === undefined) {
    return "—";
  }
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  const precision = size >= 10 || unitIndex === 0 ? 0 : 1;
  return `${size.toFixed(precision)} ${units[unitIndex]}`;
}

function peerCliCommands(publicKey: string, role: GearRole, channel: string): string {
  const key = shellQuote(publicKey);
  const nextRole = shellQuote(role === "unspecified" ? "gear" : role);
  const nextChannel = shellQuote(channel);
  return [
    `# Read this peer registration`,
    `gizclaw admin peers --context <admin-cli-context> get ${key}`,
    ``,
    `# Read peer snapshots`,
    `gizclaw admin peers --context <admin-cli-context> info ${key}`,
    `gizclaw admin peers --context <admin-cli-context> config ${key}`,
    `gizclaw admin peers --context <admin-cli-context> runtime ${key}`,
    `gizclaw admin peers --context <admin-cli-context> ota ${key}`,
    ``,
    `# Refresh state from the device-side API`,
    `gizclaw admin peers --context <admin-cli-context> refresh ${key}`,
    ``,
    `# Approve or block this peer`,
    `gizclaw admin peers --context <admin-cli-context> approve ${key} ${nextRole}`,
    `gizclaw admin peers --context <admin-cli-context> block ${key}`,
    ``,
    `# Update desired configuration`,
    `gizclaw admin peers --context <admin-cli-context> set-firmware-channel ${key} ${nextChannel}`,
    `gizclaw admin peers --context <admin-cli-context> put-config ${key} --file config.json`,
    ``,
    `# Show/apply the declarative PeerConfig resource`,
    `gizclaw admin --context <admin-cli-context> show PeerConfig ${key}`,
    `gizclaw admin --context <admin-cli-context> apply -f gear-config.json`,
    ``,
    `# Reset this peer registration`,
    `gizclaw admin peers --context <admin-cli-context> delete ${key}`,
  ].join("\n");
}

function shellQuote(value: string): string {
  return `'${value.replaceAll("'", "'\\''")}'`;
}
