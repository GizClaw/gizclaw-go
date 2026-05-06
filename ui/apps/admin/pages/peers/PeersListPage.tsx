import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import type { KeyboardEvent, MouseEvent } from "react";
import { Check, Copy, RefreshCw, Search } from "lucide-react";
import { useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";

import { Badge } from "../../../../packages/components/badge";
import { Button } from "../../../../packages/components/button";
import { Card, CardContent, CardDescription, CardTitle } from "../../../../packages/components/card";
import { Input } from "../../../../packages/components/input";
import { Skeleton } from "../../../../packages/components/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../../packages/components/table";

import type { Runtime } from "../../../../packages/adminservice";
import { EmptyState } from "../../../../packages/components/empty-state";
import { PageBreadcrumb } from "../../../../packages/components/page-breadcrumb";
import { StatusBadge } from "../../../../packages/components/status-badge";
import { usePeersPage } from "../../hooks/usePeersPage";
import { formatDate } from "../../lib/format";

dayjs.extend(relativeTime);

interface NamedRegistration {
  name?: string | null;
  device?: {
    name?: string | null;
  } | null;
}

export function PeersListPage(): JSX.Element {
  const navigate = useNavigate();
  const [copiedPublicKey, setCopiedPublicKey] = useState<string | null>(null);
  const {
    dashboard,
    peerList,
    peerPageNumber,
    filter,
    filteredGears,
    nextPage,
    prevPage,
    refreshDashboard,
    setFilter,
  } = usePeersPage();

  const handleCopyPublicKey = useCallback(async (event: MouseEvent<HTMLButtonElement>, publicKey: string) => {
    event.stopPropagation();
    try {
      await navigator.clipboard.writeText(publicKey);
      setCopiedPublicKey(publicKey);
      window.setTimeout(() => setCopiedPublicKey((current) => (current === publicKey ? null : current)), 1200);
    } catch {
      setCopiedPublicKey(null);
    }
  }, []);

  const openPeer = useCallback((publicKey: string) => {
    navigate(`/peers/${encodeURIComponent(publicKey)}`);
  }, [navigate]);

  const handlePeerRowKeyDown = useCallback((event: KeyboardEvent<HTMLTableRowElement>, publicKey: string) => {
    if (isInteractiveTarget(event.target)) {
      return;
    }
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    event.preventDefault();
    openPeer(publicKey);
  }, [openPeer]);

  return (
    <div className="space-y-6">
      <PageBreadcrumb items={[{ href: "/overview", label: "Overview" }, { label: "Peers" }]} />

      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-2">
          <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Inventory</div>
          <h1 className="text-3xl font-semibold tracking-tight">Peers</h1>
          <p className="max-w-3xl text-sm leading-6 text-muted-foreground lg:text-base">
            Browse paged inventory, filter the current page, and open a peer into its own detail route.
          </p>
        </div>
        <Button className="h-8 min-w-fit shrink-0 whitespace-nowrap px-3 text-sm" onClick={() => void refreshDashboard()} variant="outline">
          <span className="inline-flex items-center gap-2 whitespace-nowrap">
            <RefreshCw className="size-4" />
            Refresh
          </span>
        </Button>
      </div>

      {dashboard.error !== "" ? (
        <div className="rounded-lg border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive">{dashboard.error}</div>
      ) : null}

      <Card>
        <CardContent className="p-6">
          <div className="rounded-md border">
            <div className="flex flex-col gap-3 border-b px-4 py-4 lg:flex-row lg:items-start lg:justify-between">
              <div className="space-y-1">
                <CardTitle>Peer Inventory</CardTitle>
                <CardDescription>Browse paged peer results and open a row to inspect details.</CardDescription>
              </div>
              <div className="flex flex-wrap gap-2">
                <Badge variant="outline">Page {peerPageNumber}</Badge>
                <Badge variant="secondary">{dashboard.gears.length} loaded</Badge>
                {peerList.hasNext ? <Badge variant="outline">More Available</Badge> : null}
              </div>
            </div>

            <div className="flex flex-col gap-3 border-b px-4 py-4 lg:flex-row lg:items-center lg:justify-between">
              <div className="relative lg:max-w-sm lg:flex-1">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  className="h-8 pl-9"
                  onChange={(event) => setFilter(event.target.value)}
                  placeholder="Filter current page by key, role, or status"
                  value={filter}
                />
              </div>
              <div className="flex gap-2">
                <Button
                  className="h-8 min-w-fit shrink-0 whitespace-nowrap px-3 text-sm disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100 disabled:shadow-none"
                  disabled={dashboard.loading || peerList.history.length === 0}
                  onClick={prevPage}
                  type="button"
                  variant="outline"
                >
                  Previous
                </Button>
                <Button
                  className="h-8 min-w-fit shrink-0 whitespace-nowrap px-3 text-sm disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100 disabled:shadow-none"
                  disabled={dashboard.loading || !peerList.hasNext || peerList.nextCursor === null}
                  onClick={nextPage}
                  type="button"
                  variant="outline"
                >
                  Next
                </Button>
              </div>
            </div>

            {dashboard.loading ? (
              <div className="space-y-3 p-4">
                {Array.from({ length: 6 }).map((_, index) => (
                  <Skeleton className="h-16 w-full" key={index} />
                ))}
              </div>
            ) : filteredGears.length === 0 ? (
              <div className="p-4">
                <EmptyState
                  description={filter.trim() === "" ? "Peers will appear here as soon as they are registered." : "No peers on this page match the current filter."}
                  title="No matching peers"
                />
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-64">Peer</TableHead>
                    <TableHead className="w-56">Registration</TableHead>
                    <TableHead className="w-56">Runtime</TableHead>
                    <TableHead className="w-48">Endpoint</TableHead>
                    <TableHead className="text-right">Gear Updated</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredGears.map((gear) => {
                    const name = peerDisplayName(gear);
                    const copied = copiedPublicKey === gear.public_key;
                    const runtime = dashboard.runtimes[gear.public_key];

                    return (
                      <TableRow
                        className="cursor-pointer hover:bg-muted/40"
                        key={gear.public_key}
                        onClick={() => openPeer(gear.public_key)}
                        onKeyDown={(event) => handlePeerRowKeyDown(event, gear.public_key)}
                        role="link"
                        tabIndex={0}
                      >
                        <TableCell className="w-64 max-w-64">
                          <div className={name === "" ? "" : "space-y-1"}>
                            {name !== "" ? (
                              <div className="block truncate font-medium" title={name}>
                                {name}
                              </div>
                            ) : null}
                            <button
                              className="flex max-w-full items-center gap-1.5 font-mono text-xs text-muted-foreground hover:text-foreground"
                              onClick={(event) => void handleCopyPublicKey(event, gear.public_key)}
                              title="Copy public key"
                              type="button"
                            >
                              <span className="block truncate">{gear.public_key}</span>
                              {copied ? <Check className="size-3 shrink-0 text-emerald-600" /> : <Copy className="size-3 shrink-0" />}
                            </button>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex justify-start gap-1.5">
                            <Badge title={gear.auto_registered ? "Auto registered" : "Manual registration"} variant="secondary">
                              {gear.auto_registered ? "A" : "M"}
                            </Badge>
                            <StatusBadge status={gear.status} />
                            <Badge title={`Role: ${gear.role}`} variant="outline">
                              {gear.role}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell>
                          <PeerRuntime runtime={runtime} />
                        </TableCell>
                        <TableCell>
                          <PeerEndpoint runtime={runtime} />
                        </TableCell>
                        <TableCell className="text-right text-sm text-muted-foreground">{formatDate(gear.updated_at)}</TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}

            <div className="flex items-center justify-between border-t px-4 py-4 text-sm text-muted-foreground">
              <span>
                Showing {filteredGears.length} of {dashboard.gears.length} peers on page {peerPageNumber}
              </span>
              <span>{peerList.hasNext ? "Next page available" : "End of results"}</span>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function peerDisplayName(gear: NamedRegistration): string {
  return gear.name?.trim() || gear.device?.name?.trim() || "";
}

function PeerRuntime({ runtime }: { runtime: Runtime | null | undefined }): JSX.Element {
  if (runtime === undefined || runtime === null) {
    return <span className="text-sm text-muted-foreground">Unavailable</span>;
  }
  const lastSeen = formatRuntimeLastSeen(runtime.last_seen_at);

  return (
    <div className="max-w-56 space-y-1">
      <div className="flex items-center gap-2">
        <Badge className={runtime.online ? undefined : "border-transparent bg-muted text-muted-foreground hover:bg-muted"} variant={runtime.online ? "success" : "outline"}>
          {runtime.online ? "Online" : "Offline"}
        </Badge>
        <span className="whitespace-nowrap text-xs text-muted-foreground" title={lastSeen.title}>
          {lastSeen.label}
        </span>
      </div>
    </div>
  );
}

function PeerEndpoint({ runtime }: { runtime: Runtime | null | undefined }): JSX.Element {
  if (runtime === undefined || runtime === null) {
    return <span className="text-sm text-muted-foreground">Unavailable</span>;
  }
  const endpoint = runtime.last_addr?.trim() ?? "";
  if (endpoint === "") {
    return <span className="text-sm text-muted-foreground">No endpoint</span>;
  }
  return (
    <div className="max-w-48 truncate font-mono text-xs text-muted-foreground" title={endpoint}>
      {endpoint}
    </div>
  );
}

function formatRuntimeLastSeen(value: string | undefined): { label: string; title: string } {
  if (value === undefined || value === "" || value.startsWith("0001-01-01")) {
    return { label: "Never seen", title: "Never seen" };
  }
  const seenAt = dayjs(value);
  if (!seenAt.isValid()) {
    return { label: value, title: value };
  }
  return { label: `Seen ${seenAt.fromNow()}`, title: seenAt.toDate().toLocaleString() };
}

function isInteractiveTarget(target: EventTarget): boolean {
  return target instanceof Element && target.closest("a,button,input,select,textarea") !== null;
}
