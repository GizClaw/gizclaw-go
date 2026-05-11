import { RefreshCw } from "lucide-react";

import { Badge } from "../../components/badge";
import { Button } from "../../components/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/card";
import { Skeleton } from "../../components/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../components/table";
import { expectData } from "../../components/api";
import { listWorkflows, type WorkflowDocument } from "@gizclaw/adminservice";

import { ErrorBanner } from "../../components/banners";
import { EmptyState } from "../../components/empty-state";
import { PageBreadcrumb } from "../../components/page-breadcrumb";
import { useCursorListPage } from "../../hooks/useCursorListPage";

export function WorkflowsListPage(): JSX.Element {
  const { error, hasNext, items, loading, nextPage, pageNumber, prevPage, refresh } = useCursorListPage<WorkflowDocument>(
    async (query) => {
      const result = await expectData(listWorkflows({ query }));
      return {
        hasNext: result.has_next,
        items: result.items ?? [],
        nextCursor: result.next_cursor ?? null,
      };
    },
  );

  return (
    <div className="space-y-6">
      <PageBreadcrumb items={[{ href: "/overview", label: "Overview" }, { label: "Workflows" }]} />

      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-2">
          <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">AI</div>
          <h1 className="text-3xl font-semibold tracking-tight">Workflows</h1>
          <p className="max-w-3xl text-sm leading-6 text-muted-foreground lg:text-base">
            Declarative workflow documents that workspaces load when running agents.
          </p>
        </div>
        <Button className="h-8 min-w-fit shrink-0 whitespace-nowrap px-3 text-sm" onClick={() => void refresh()} variant="outline">
          <span className="inline-flex items-center gap-2 whitespace-nowrap">
            <RefreshCw className="size-4" />
            Refresh
          </span>
        </Button>
      </div>

      {error !== "" ? <ErrorBanner message={error} /> : null}

      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
          <div className="space-y-1">
            <CardTitle>Workflow catalog</CardTitle>
            <CardDescription>Workflow documents grouped by top-level kind and metadata.</CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            <Badge variant="outline">Page {pageNumber}</Badge>
            <Badge variant="secondary">{items.length} loaded</Badge>
            {hasNext ? <Badge variant="outline">More Available</Badge> : null}
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex justify-end gap-2">
            <Button
              className="h-8 min-w-fit shrink-0 whitespace-nowrap px-3 text-sm disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100 disabled:shadow-none"
              disabled={loading || pageNumber === 1}
              onClick={prevPage}
              type="button"
              variant="outline"
            >
              Previous
            </Button>
            <Button
              className="h-8 min-w-fit shrink-0 whitespace-nowrap px-3 text-sm disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100 disabled:shadow-none"
              disabled={loading || !hasNext}
              onClick={nextPage}
              type="button"
              variant="outline"
            >
              Next
            </Button>
          </div>

          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 6 }).map((_, index) => (
                <Skeleton className="h-14 w-full" key={index} />
              ))}
            </div>
          ) : items.length === 0 ? (
            <EmptyState description="Workflow documents will appear here after they are created." title="No workflows" />
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Kind</TableHead>
                    <TableHead>API Version</TableHead>
                    <TableHead>Description</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((workflow) => (
                    <TableRow key={workflow.metadata.name}>
                      <TableCell className="font-medium">{workflow.metadata.name}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{workflow.kind}</Badge>
                      </TableCell>
                      <TableCell>{workflow.apiVersion}</TableCell>
                      <TableCell className="max-w-[28rem] text-sm text-muted-foreground">{workflow.metadata.description?.trim() || "—"}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
