import { ChevronLeft, Copy, RefreshCw, Save } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { getVolcTenant, listCredentials, putVolcTenant, syncVolcTenantVoices, type Credential, type VolcTenant } from "../../../../packages/adminservice";
import { expectData, toMessage } from "../../../../packages/components/api";
import { Badge } from "../../../../packages/components/badge";
import { Button } from "../../../../packages/components/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../../../packages/components/card";
import { DetailBlock } from "../../../../packages/components/detail-block";
import { EmptyState } from "../../../../packages/components/empty-state";
import { ErrorBanner, NoticeBanner } from "../../../../packages/components/banners";
import { FormField } from "../../../../packages/components/form-field";
import { Input } from "../../../../packages/components/input";
import { PageBreadcrumb } from "../../../../packages/components/page-breadcrumb";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../../../packages/components/select";
import { Skeleton } from "../../../../packages/components/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../../../packages/components/tabs";

type VolcTenantForm = {
  appID: string;
  credentialName: string;
  description: string;
  endpoint: string;
  region: string;
  resourceIDs: string;
};

export function VolcTenantDetailPage(): JSX.Element {
  const params = useParams();
  const tenantName = useMemo(() => decodeRouteParam(params.name ?? ""), [params.name]);
  const [tenant, setTenant] = useState<VolcTenant | null>(null);
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [form, setForm] = useState<VolcTenantForm>(() => emptyForm());
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [copied, setCopied] = useState(false);

  const load = async (): Promise<void> => {
    if (tenantName === "") {
      setLoading(false);
      setError("Missing Volcengine tenant name in the URL.");
      return;
    }
    setLoading(true);
    setError("");
    setNotice("");
    try {
      const [nextTenant, credentialList] = await Promise.all([
        expectData(getVolcTenant({ path: { name: tenantName } })),
        expectData(listCredentials({ query: { limit: 200 } })),
      ]);
      setTenant(nextTenant);
      setForm(formFromTenant(nextTenant));
      setCredentials(credentialList.items ?? []);
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [tenantName]);

  const save = async (): Promise<void> => {
    if (tenant === null) {
      return;
    }
    setSaving(true);
    setError("");
    setNotice("");
    try {
      const updated = await expectData(
        putVolcTenant({
          body: {
            name: tenant.name,
            app_id: form.appID.trim(),
            credential_name: form.credentialName.trim(),
            region: optionalString(form.region),
            endpoint: optionalString(form.endpoint),
            resource_ids: optionalStringList(form.resourceIDs),
            description: optionalString(form.description),
          },
          path: { name: tenant.name },
        }),
      );
      setTenant(updated);
      setForm(formFromTenant(updated));
      setNotice("Volcengine tenant saved.");
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setSaving(false);
    }
  };

  const syncVoices = async (): Promise<void> => {
    if (tenant === null) {
      return;
    }
    setSyncing(true);
    setError("");
    setNotice("");
    try {
      const result = await expectData(syncVolcTenantVoices({ path: { name: tenant.name } }));
      await load();
      setNotice(`Synced voices: ${result.created_count} created, ${result.updated_count} updated, ${result.deleted_count} deleted.`);
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setSyncing(false);
    }
  };

  const copyJSON = async (): Promise<void> => {
    if (tenant === null) {
      return;
    }
    await navigator.clipboard.writeText(JSON.stringify(tenant, null, 2));
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  };

  if (tenantName === "") {
    return <EmptyState description="Missing Volcengine tenant name in the URL." title="Invalid route" />;
  }

  const credentialOptions = mergeCredentialOptions(credentials, form.credentialName);

  return (
    <div className="space-y-6">
      <PageBreadcrumb
        items={[
          { href: "/overview", label: "Overview" },
          { href: "/providers/volc-tenants", label: "Volcengine Tenants" },
          { label: tenantName },
        ]}
      />

      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-2">
          <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Providers</div>
          <h1 className="text-3xl font-semibold tracking-tight">{tenant?.name ?? tenantName}</h1>
          <p className="max-w-3xl text-sm leading-6 text-muted-foreground lg:text-base">Volcengine speech tenant configuration and voice sync controls.</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button asChild size="sm" variant="outline">
            <Link to="/providers/volc-tenants">
              <ChevronLeft className="size-4" />
              Back to list
            </Link>
          </Button>
          <Button className="min-w-fit shrink-0 whitespace-nowrap" onClick={() => void load()} size="sm" variant="outline">
            <RefreshCw className="size-4" />
            Reload
          </Button>
          <Button className="min-w-fit shrink-0 whitespace-nowrap" disabled={tenant === null || syncing} onClick={() => void syncVoices()} size="sm" variant="outline">
            <RefreshCw className={`size-4 ${syncing ? "animate-spin" : ""}`} />
            Sync voices
          </Button>
          {tenant ? <Badge variant="secondary">Volcengine</Badge> : null}
        </div>
      </div>

      {notice !== "" ? <NoticeBanner message={notice} tone="success" /> : null}
      {error !== "" ? <ErrorBanner message={error} /> : null}

      {loading ? (
        <div className="space-y-4">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-80 w-full" />
        </div>
      ) : tenant === null ? (
        <EmptyState description="This Volcengine tenant could not be loaded." title="Volcengine tenant not found" />
      ) : (
        <Tabs defaultValue="summary">
          <TabsList>
            <TabsTrigger value="summary">Summary</TabsTrigger>
            <TabsTrigger value="edit">Edit</TabsTrigger>
            <TabsTrigger value="debug">Debug</TabsTrigger>
          </TabsList>

          <TabsContent className="space-y-4" value="summary">
            <div className="grid gap-4 xl:grid-cols-2">
              <DetailBlock
                items={[
                  ["Name", tenant.name],
                  ["App ID", tenant.app_id],
                  ["Credential", tenant.credential_name],
                  ["Description", tenant.description],
                  ["Last sync", tenant.last_synced_at],
                ]}
                title="Tenant"
              />
              <DetailBlock
                items={[
                  ["Region", tenant.region],
                  ["Endpoint", tenant.endpoint],
                  ["Resource IDs", tenant.resource_ids?.join(", ")],
                  ["Created", tenant.created_at],
                  ["Updated", tenant.updated_at],
                ]}
                title="Volcengine"
              />
            </div>
          </TabsContent>

          <TabsContent value="edit">
            <Card>
              <CardHeader>
                <CardTitle>Edit Volcengine Tenant</CardTitle>
                <CardDescription>Update tenant credential binding and speech sync options. The tenant name is the resource identity and is not editable here.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 lg:grid-cols-2">
                  <FormField description="Resource identity. Rename via resource replacement if needed." label="Name">
                    <Input disabled value={tenant.name} />
                  </FormField>
                  <FormField description="Volcengine speech AppID used to list purchased and cloned SpeakerIDs." label="App ID">
                    <Input onChange={(event) => setForm((current) => ({ ...current, appID: event.target.value }))} placeholder="9476442538" value={form.appID} />
                  </FormField>
                  <FormField description="Stored credential used when syncing Volcengine voices." label="Credential">
                    <Select disabled={saving || credentialOptions.length === 0} onValueChange={(value) => setForm((current) => ({ ...current, credentialName: value }))} value={form.credentialName}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select credential" />
                      </SelectTrigger>
                      <SelectContent>
                        {credentialOptions.map((credential) => (
                          <SelectItem key={credential.name} value={credential.name}>
                            {credential.name} · {credential.provider}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </FormField>
                  <FormField description="Optional Volcengine region override." label="Region">
                    <Input onChange={(event) => setForm((current) => ({ ...current, region: event.target.value }))} placeholder="cn-north-1" value={form.region} />
                  </FormField>
                  <FormField description="Optional Volcengine endpoint override." label="Endpoint">
                    <Input onChange={(event) => setForm((current) => ({ ...current, endpoint: event.target.value }))} placeholder="https://..." value={form.endpoint} />
                  </FormField>
                  <FormField description="Comma or newline separated ResourceIDs for purchased or cloned voices. Leave empty to sync public timbres only." label="Resource IDs">
                    <textarea
                      className="min-h-24 w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 disabled:cursor-not-allowed disabled:opacity-50"
                      onChange={(event) => setForm((current) => ({ ...current, resourceIDs: event.target.value }))}
                      placeholder={"seed-tts-2.0"}
                      value={form.resourceIDs}
                    />
                  </FormField>
                  <FormField description="Human-readable note for operators." label="Description">
                    <textarea
                      className="min-h-24 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 disabled:cursor-not-allowed disabled:opacity-50"
                      onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
                      value={form.description}
                    />
                  </FormField>
                </div>
                <div className="flex justify-end border-t pt-4">
                  <Button disabled={saving} onClick={() => void save()} type="button">
                    <Save className="size-4" />
                    {saving ? "Saving..." : "Save"}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="debug">
            <Card>
              <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
                <div className="space-y-1">
                  <CardTitle>Raw Tenant JSON</CardTitle>
                  <CardDescription>Exact API shape returned by the admin service.</CardDescription>
                </div>
                <Button className="min-w-fit shrink-0 whitespace-nowrap" onClick={() => void copyJSON()} size="sm" variant="outline">
                  <Copy className="size-4" />
                  {copied ? "Copied" : "Copy JSON"}
                </Button>
              </CardHeader>
              <CardContent>
                <pre className="max-h-[36rem] overflow-auto rounded-md bg-muted p-4 text-xs leading-5">
                  {JSON.stringify(tenant, null, 2)}
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

function emptyForm(): VolcTenantForm {
  return { appID: "", credentialName: "", description: "", endpoint: "", region: "", resourceIDs: "" };
}

function formFromTenant(tenant: VolcTenant): VolcTenantForm {
  return {
    appID: tenant.app_id,
    credentialName: tenant.credential_name,
    description: tenant.description ?? "",
    endpoint: tenant.endpoint ?? "",
    region: tenant.region ?? "",
    resourceIDs: tenant.resource_ids?.join("\n") ?? "",
  };
}

function optionalString(value: string): string | undefined {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function optionalStringList(value: string): string[] | undefined {
  const items = value
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter((item) => item !== "");
  return items.length === 0 ? undefined : items;
}

function mergeCredentialOptions(credentials: Credential[], currentName: string): Credential[] {
  if (currentName === "" || credentials.some((credential) => credential.name === currentName)) {
    return credentials;
  }
  return [{ name: currentName, provider: "unknown", method: "api_key", body: {}, created_at: "", updated_at: "" }, ...credentials];
}
