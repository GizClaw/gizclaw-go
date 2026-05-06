import { useCallback, useEffect, useMemo, useState } from "react";

import { expectData, toMessage } from "../../../packages/components/api";
import { getGearRuntime, listDepots, listGears } from "../../../packages/adminservice";
import { getServerInfo, type ServerInfo } from "../../../packages/serverpublic";

import type { Depot, Registration, Runtime } from "../../../packages/adminservice";

export const PEER_PAGE_LIMIT = 50;

export type PeerRuntimeMap = Record<string, Runtime | null>;

export interface PeerListState {
  cursor: string | null;
  hasNext: boolean;
  history: Array<string | null>;
  nextCursor: string | null;
}

export interface PeersPageState {
  depots: Depot[];
  error: string;
  gears: Registration[];
  loading: boolean;
  runtimes: PeerRuntimeMap;
  serverInfo: ServerInfo | null;
}

export function usePeersPage(): {
  dashboard: PeersPageState;
  peerList: PeerListState;
  peerPageNumber: number;
  filter: string;
  filteredGears: Registration[];
  loadDashboard: (cursor: string | null, history: Array<string | null>) => Promise<void>;
  nextPage: () => void;
  prevPage: () => void;
  refreshDashboard: () => Promise<void>;
  setFilter: (value: string) => void;
} {
  const [filter, setFilter] = useState("");
  const [dashboard, setDashboard] = useState<PeersPageState>({
    depots: [],
    error: "",
    gears: [],
    loading: true,
    runtimes: {},
    serverInfo: null,
  });
  const [peerList, setPeerList] = useState<PeerListState>({
    cursor: null,
    hasNext: false,
    history: [],
    nextCursor: null,
  });

  const loadDashboard = useCallback(async (cursor: string | null, history: Array<string | null>) => {
    setDashboard((current) => ({ ...current, error: "", loading: true }));
    try {
      const [serverInfo, registrations, depots] = await Promise.all([
        expectData(getServerInfo()),
        expectData(
          listGears({
            query: {
              cursor: cursor ?? undefined,
              limit: PEER_PAGE_LIMIT,
            },
          }),
        ),
        expectData(listDepots()),
      ]);

      const gears = registrations.items ?? [];
      const runtimes = await loadPeerRuntimes(gears);

      setDashboard({
        depots: depots.items ?? [],
        error: "",
        gears,
        loading: false,
        runtimes,
        serverInfo,
      });
      setPeerList({
        cursor,
        hasNext: registrations.has_next,
        history,
        nextCursor: registrations.next_cursor ?? null,
      });
    } catch (error) {
      setDashboard((current) => ({
        ...current,
        error: toMessage(error),
        loading: false,
      }));
    }
  }, []);

  const refreshDashboard = useCallback(async () => {
    await loadDashboard(peerList.cursor, peerList.history);
  }, [peerList.cursor, peerList.history, loadDashboard]);

  useEffect(() => {
    void loadDashboard(null, []);
  }, [loadDashboard]);

  const filteredGears = useMemo(() => {
    if (filter.trim() === "") {
      return dashboard.gears;
    }
    const query = filter.trim().toLowerCase();
    return dashboard.gears.filter((gear) =>
      [
        gear.public_key,
        gear.role,
        gear.status,
        gear.auto_registered ? "auto" : "manual",
        dashboard.runtimes[gear.public_key]?.online ? "online" : "offline",
        dashboard.runtimes[gear.public_key]?.last_addr ?? "",
      ].some((value) =>
        value.toLowerCase().includes(query),
      ),
    );
  }, [dashboard.gears, dashboard.runtimes, filter]);

  const nextPage = useCallback(() => {
    if (peerList.nextCursor === null) {
      return;
    }
    void loadDashboard(peerList.nextCursor, [...peerList.history, peerList.cursor]);
  }, [peerList.cursor, peerList.history, peerList.nextCursor, loadDashboard]);

  const prevPage = useCallback(() => {
    if (peerList.history.length === 0) {
      return;
    }
    const previousCursor = peerList.history[peerList.history.length - 1] ?? null;
    void loadDashboard(previousCursor, peerList.history.slice(0, -1));
  }, [peerList.history, loadDashboard]);

  const peerPageNumber = peerList.history.length + 1;

  return {
    dashboard,
    peerList,
    peerPageNumber,
    filter,
    filteredGears,
    loadDashboard,
    nextPage,
    prevPage,
    refreshDashboard,
    setFilter,
  };
}

async function loadPeerRuntimes(gears: Registration[]): Promise<PeerRuntimeMap> {
  const entries = await Promise.all(
    gears.map(async (gear): Promise<[string, Runtime | null]> => {
      try {
        const runtime = await expectData(getGearRuntime({ path: { publicKey: gear.public_key } }));
        return [gear.public_key, runtime];
      } catch {
        return [gear.public_key, null];
      }
    }),
  );
  return Object.fromEntries(entries);
}
