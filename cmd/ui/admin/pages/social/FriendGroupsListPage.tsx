import { Plus, RefreshCw, UsersRound } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { createFriendGroup, deleteFriendGroup, getFriendGroup, listFriendGroups, type FriendGroupObject } from "@gizclaw/adminservice";
import { expectData, toMessage } from "../../components/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";

import { ErrorBanner, NoticeBanner } from "../../components/banners";
import { DeleteConfirmButton } from "../../components/delete-confirm-button";
import { EmptyState } from "../../components/empty-state";
import { FormField } from "../../components/form-field";
import { PageHeader, PageSummaryCard } from "../../components/page-layout";
import { useCursorListPage } from "../../hooks/useCursorListPage";
import { formatDate, formatShortKey } from "../../lib/format";
import { friendGroupDetailPath, socialWorkspaceName } from "./social-utils";

export function FriendGroupsListPage(): JSX.Element {
  const navigate = useNavigate();
  const { error, hasNext, items, loading, nextPage, pageNumber, prevPage, refresh } = useCursorListPage<FriendGroupObject>(async (query) => {
    const result = await expectData(listFriendGroups({ query }));
    return {
      hasNext: result.has_next,
      items: result.items ?? [],
      nextCursor: result.next_cursor ?? null,
    };
  });
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [notice, setNotice] = useState<{ message: string; tone: "error" | "success" } | null>(null);
  const [busy, setBusy] = useState("");

  const create = async (): Promise<void> => {
    setBusy("create");
    setNotice(null);
    const groupID = name.trim();
    try {
      const group = await expectData(createFriendGroup({ body: { name: groupID, description: description.trim() || undefined } }));
      setName("");
      setDescription("");
      navigate(friendGroupDetailPath(group));
    } catch (err) {
      try {
        const group = await expectData(getFriendGroup({ path: { id: groupID } }));
        setName("");
        setDescription("");
        navigate(friendGroupDetailPath(group));
        return;
      } catch {
        // Keep the original create error when the group was not created.
      }
      setNotice({ message: toMessage(err), tone: "error" });
    } finally {
      setBusy("");
    }
  };

  const remove = async (group: FriendGroupObject): Promise<void> => {
    const id = group.id ?? "";
    if (id === "") {
      return;
    }
    setBusy(`delete:${id}`);
    setNotice(null);
    try {
      await expectData(deleteFriendGroup({ path: { id } }));
      await refresh();
      setNotice({ message: "Friend group deleted.", tone: "success" });
    } catch (err) {
      setNotice({ message: toMessage(err), tone: "error" });
    } finally {
      setBusy("");
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <Button className="h-8 min-w-fit shrink-0 whitespace-nowrap px-3 text-sm" disabled={loading} onClick={() => void refresh()} variant="outline">
            <RefreshCw className="size-4" />
            Refresh
          </Button>
        }
        items={[{ href: "/overview", label: "Overview" }, { label: "Friend Groups" }]}
      />

      <PageSummaryCard
        description="Global group resources with backing chatroom workspaces. Admin-created groups start without an implicit owner member."
        eyebrow="Social"
        meta={
          <>
            <Badge variant="outline">Page {pageNumber}</Badge>
            <Badge variant="secondary">{items.length} loaded</Badge>
            {hasNext ? <Badge variant="outline">More Available</Badge> : null}
          </>
        }
        title="Friend Groups"
      />

      {error !== "" ? <ErrorBanner message={error} /> : null}
      {notice !== null ? <NoticeBanner message={notice.message} tone={notice.tone} /> : null}

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Create Friend Group</CardTitle>
          <CardDescription>Create group metadata and a backing chatroom workspace. Members are added separately.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)_auto] lg:items-end">
            <FormField label="Name">
              <Input onChange={(event) => setName(event.target.value)} placeholder="story-club" value={name} />
            </FormField>
            <FormField label="Description">
              <Textarea className="min-h-10" onChange={(event) => setDescription(event.target.value)} placeholder="Group description" value={description} />
            </FormField>
            <Button disabled={busy !== "" || name.trim() === ""} onClick={() => void create()} type="button">
              <Plus className="size-4" />
              Create
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
          <div className="space-y-1">
            <CardTitle>Groups</CardTitle>
            <CardDescription>Cursor-paginated friend group resources.</CardDescription>
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
            <EmptyState description="Friend groups will appear here after they are created." title="No friend groups" />
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Group</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead>Workspace</TableHead>
                    <TableHead className="text-right">Updated</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((group) => {
                    const id = group.id ?? "";
                    return (
                      <TableRow className="hover:bg-muted/40" key={id}>
                        <TableCell>
                          <div className="font-medium">{group.name?.trim() || id}</div>
                          <div className="break-all font-mono text-xs text-muted-foreground">{id}</div>
                        </TableCell>
                        <TableCell className="max-w-[28rem] text-sm leading-6 text-muted-foreground">{group.description?.trim() || "—"}</TableCell>
                        <TableCell className="font-mono text-xs">{socialWorkspaceName(group.workspace_name)}</TableCell>
                        <TableCell className="text-right text-sm text-muted-foreground">{formatDate(group.updated_at)}</TableCell>
                        <TableCell className="text-right">
                          <div className="flex flex-wrap justify-end gap-2">
                            <Button asChild className="h-8 min-w-fit shrink-0 whitespace-nowrap px-3 text-sm" disabled={id === ""} variant="outline">
                              <Link to={friendGroupDetailPath(group)}>
                                <UsersRound className="size-4" />
                                Open
                              </Link>
                            </Button>
                            <DeleteConfirmButton
                              description={`Delete group ${formatShortKey(id)} and its backing social workspace.`}
                              disabled={busy !== "" || id === ""}
                              onConfirm={() => void remove(group)}
                              size="sm"
                              title="Delete friend group?"
                            >
                              Delete
                            </DeleteConfirmButton>
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
