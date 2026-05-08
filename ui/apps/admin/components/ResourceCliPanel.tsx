import { Copy } from "lucide-react";
import { useState } from "react";

import type { Resource } from "../../../packages/adminservice";
import { Button } from "../../../packages/components/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../../packages/components/card";

type ResourceCliPanelProps = {
  commands: string;
  resource: Resource | null;
  resourceDescription: string;
  resourceTitle: string;
};

export function ResourceCliPanel({
  commands,
  resource,
  resourceDescription,
  resourceTitle,
}: ResourceCliPanelProps): JSX.Element {
  const [copied, setCopied] = useState(false);

  const copyJSON = async (): Promise<void> => {
    if (resource === null) {
      return;
    }
    await navigator.clipboard.writeText(JSON.stringify(resource, null, 2));
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>CLI Commands</CardTitle>
          <CardDescription>Declarative admin resources use JSON. Use a separate CLI context instead of reusing the UI context.</CardDescription>
        </CardHeader>
        <CardContent>
          <pre className="overflow-auto rounded-md bg-muted p-4 text-xs leading-5">{commands}</pre>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
          <div className="space-y-1">
            <CardTitle>{resourceTitle}</CardTitle>
            <CardDescription>{resourceDescription}</CardDescription>
          </div>
          <Button className="min-w-fit shrink-0 whitespace-nowrap" disabled={resource === null} onClick={() => void copyJSON()} size="sm" variant="outline">
            <Copy className="size-4" />
            {copied ? "Copied" : "Copy Spec"}
          </Button>
        </CardHeader>
        <CardContent>
          <pre className="max-h-[36rem] overflow-auto rounded-md bg-muted p-4 text-xs leading-5">
            {JSON.stringify(resource, null, 2)}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}
