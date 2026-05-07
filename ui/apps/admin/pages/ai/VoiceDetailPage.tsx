import { ChevronLeft, Copy, RefreshCw } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { getResource, getVoice, type Resource, type Voice } from "../../../../packages/adminservice";
import { expectData, toMessage } from "../../../../packages/components/api";
import { Badge } from "../../../../packages/components/badge";
import { Button } from "../../../../packages/components/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../../../packages/components/card";
import { DetailBlock } from "../../../../packages/components/detail-block";
import { EmptyState } from "../../../../packages/components/empty-state";
import { ErrorBanner } from "../../../../packages/components/banners";
import { PageBreadcrumb } from "../../../../packages/components/page-breadcrumb";
import { Skeleton } from "../../../../packages/components/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../../../packages/components/tabs";
import { formatDate, formatValue } from "../../lib/format";

export function VoiceDetailPage(): JSX.Element {
  const params = useParams();
  const voiceID = useMemo(() => decodeRouteParam(params.id ?? ""), [params.id]);
  const [voice, setVoice] = useState<Voice | null>(null);
  const [voiceResource, setVoiceResource] = useState<Resource | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);

  const load = async (): Promise<void> => {
    if (voiceID === "") {
      setLoading(false);
      setError("Missing voice ID in the URL.");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const [nextVoice, nextResource] = await Promise.all([
        expectData(getVoice({ path: { id: voiceID } })),
        expectData(getResource({ path: { kind: "Voice", name: voiceID } })),
      ]);
      setVoice(nextVoice);
      setVoiceResource(nextResource);
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [voiceID]);

  const copyJSON = async (): Promise<void> => {
    if (voiceResource === null) {
      return;
    }
    await navigator.clipboard.writeText(JSON.stringify(voiceResource, null, 2));
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  };

  if (voiceID === "") {
    return <EmptyState description="Missing voice ID in the URL." title="Invalid route" />;
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb
        items={[
          { href: "/overview", label: "Overview" },
          { href: "/ai/voices", label: "Voices" },
          { label: compactVoiceID(voiceID) },
        ]}
      />

      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-2">
          <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">AI</div>
          <h1 className="text-3xl font-semibold tracking-tight">{voice?.name?.trim() || compactVoiceID(voiceID)}</h1>
          <p className="max-w-3xl break-all font-mono text-xs leading-6 text-muted-foreground">{voiceID}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button asChild size="sm" variant="outline">
            <Link to="/ai/voices">
              <ChevronLeft className="size-4" />
              Back to list
            </Link>
          </Button>
          <Button className="min-w-fit shrink-0 whitespace-nowrap" onClick={() => void load()} size="sm" variant="outline">
            <span className="inline-flex items-center gap-2 whitespace-nowrap">
              <RefreshCw className="size-4" />
              Reload
            </span>
          </Button>
          {voice ? <Badge variant={voice.source === "sync" ? "secondary" : "outline"}>{voice.source}</Badge> : null}
        </div>
      </div>

      {loading ? (
        <div className="space-y-4">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-80 w-full" />
        </div>
      ) : error !== "" ? (
        <ErrorBanner message={error} />
      ) : voice === null ? (
        <EmptyState description="This voice could not be loaded." title="Voice not found" />
      ) : (
        <Tabs defaultValue="summary">
          <TabsList>
            <TabsTrigger value="summary">Summary</TabsTrigger>
            <TabsTrigger value="cli">CLI</TabsTrigger>
          </TabsList>

          <TabsContent className="space-y-4" value="summary">
            <div className="grid gap-4 xl:grid-cols-2">
              <DetailBlock
                items={[
                  ["Internal ID", voice.id],
                  ["Name", voice.name],
                  ["Description", voice.description],
                  ["Source", voice.source],
                  ["Provider", providerDisplayText(voice)],
                  ["Provider kind", voice.provider.kind],
                ]}
                title="Voice"
              />
              <DetailBlock
                items={[
                  ["Provider voice ID", providerDataString(voice, "voice_id")],
                  ["Resource ID", providerDataString(voice, "resource_id")],
                  ["State", providerDataString(voice, "state")],
                  ["Status", providerDataString(voice, "status")],
                  ["Synced at", voice.synced_at],
                  ["Created", voice.created_at],
                  ["Updated", voice.updated_at],
                ]}
                title="Provider Data"
              />
            </div>
          </TabsContent>

          <TabsContent className="space-y-4" value="cli">
            <Card>
              <CardHeader>
                <CardTitle>CLI Commands</CardTitle>
                <CardDescription>Declarative admin resources use JSON. Use a separate CLI context instead of reusing the UI context.</CardDescription>
              </CardHeader>
              <CardContent>
                <pre className="overflow-auto rounded-md bg-muted p-4 text-xs leading-5">
                  {cliCommands(voice)}
                </pre>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
                <div className="space-y-1">
                  <CardTitle>Voice Resource Spec</CardTitle>
                  <CardDescription>{voiceResourceDescription(voice)}</CardDescription>
                </div>
                <Button className="min-w-fit shrink-0 whitespace-nowrap" onClick={() => void copyJSON()} size="sm" variant="outline">
                  <Copy className="size-4" />
                  {copied ? "Copied" : "Copy Spec"}
                </Button>
              </CardHeader>
              <CardContent>
                <pre className="max-h-[36rem] overflow-auto rounded-md bg-muted p-4 text-xs leading-5">
                  {JSON.stringify(voiceResource, null, 2)}
                </pre>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      )}
    </div>
  );
}

function decodeRouteParam(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function compactVoiceID(id: string): string {
  const trimmed = id.trim();
  if (trimmed === "") {
    return "Voice";
  }
  const parts = trimmed.split(":");
  return parts[parts.length - 1] ?? trimmed;
}

function providerDataString(voice: Voice, key: string): string | undefined {
  const providerData = voice.provider_data?.[voice.provider.kind];
  if (typeof providerData !== "object" || providerData === null || Array.isArray(providerData)) {
    return undefined;
  }
  const value = (providerData as Record<string, unknown>)[key];
  return typeof value === "string" && value.trim() !== "" ? value : undefined;
}

function providerDisplayText(voice: Voice): string {
  return `${providerPrefix(voice.provider.kind)}/${voice.provider.name}`;
}

function providerPrefix(kind: string): string {
  switch (kind) {
    case "minimax-tenant":
      return "minimax";
    case "volc-tenant":
      return "volc";
    default:
      return kind;
  }
}

function cliCommands(voice: Voice): string {
  const id = shellQuote(voice.id);
  const commands = [
    `# Show this voice resource`,
    `gizclaw admin --context <admin-cli-context> show Voice ${id}`,
  ];
  if (voice.source === "sync") {
    commands.push(
      ``,
      `# Synced voices are read-only; update them by syncing the provider tenant.`,
    );
  } else {
    commands.push(
      ``,
      `# Apply/update from a JSON file`,
      `gizclaw admin --context <admin-cli-context> apply -f voice.json`,
    );
  }
  commands.push(
    ``,
    `# Delete this voice resource`,
    `gizclaw admin --context <admin-cli-context> delete Voice ${id}`,
  );
  if (voice.provider.kind === "volc-tenant") {
    const tenantName = shellQuote(voice.provider.name);
    commands.push(
      ``,
      `# Show the Volcengine tenant that owns this voice`,
      `gizclaw admin --context <admin-cli-context> show VolcTenant ${tenantName}`,
      ``,
      `# Re-sync voices from this Volcengine tenant`,
      `gizclaw admin volc-tenants --context <admin-cli-context> sync-voices ${tenantName}`,
    );
  }
  return commands.join("\n");
}

function voiceResourceDescription(voice: Voice): string {
  if (voice.source === "sync") {
    return "JSON returned by the resource API. Synced voice resources are read-only; update them by syncing the provider tenant.";
  }
  return "JSON returned by the resource API and accepted by admin apply. The resource metadata name is the voice ID.";
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`;
}
