import type { AdminFriendObject, FriendGroupObject } from "@gizclaw/adminservice";

import { formatShortKey } from "../../lib/format";

export function decodeRouteParam(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

export function friendDetailPath(friend: AdminFriendObject): string {
  return `/social/friends/${encodeURIComponent(friend.owner_public_key)}/${encodeURIComponent(friend.id)}`;
}

export function friendGroupDetailPath(group: FriendGroupObject): string {
  return `/social/friend-groups/${encodeURIComponent(group.id ?? "")}`;
}

export function socialWorkspaceName(value: string | undefined): string {
  return value?.trim() ? value : "—";
}

export function socialPeerLabel(value: string | undefined): string {
  return value?.trim() ? formatShortKey(value) : "No peer";
}
