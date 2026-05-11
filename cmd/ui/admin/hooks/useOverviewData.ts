import { useEffect, useState } from "react";

import { expectData, toMessage } from "../components/api";
import { listDepots, listPeers } from "@gizclaw/adminservice";
import { getServerInfo, type ServerInfo } from "@gizclaw/serverpublic";

import type { Depot, Registration } from "@gizclaw/adminservice";

import { PEER_PAGE_LIMIT } from "./usePeersPage";

export interface OverviewData {
  depots: Depot[];
  error: string;
  peers: Registration[];
  loading: boolean;
  serverInfo: ServerInfo | null;
}

export function useOverviewData(): OverviewData {
  const [data, setData] = useState<OverviewData>({
    depots: [],
    error: "",
    peers: [],
    loading: true,
    serverInfo: null,
  });

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const [serverInfo, registrations, depots] = await Promise.all([
          expectData(getServerInfo()),
          expectData(
            listPeers({
              query: { limit: PEER_PAGE_LIMIT },
            }),
          ),
          expectData(listDepots()),
        ]);
        if (cancelled) {
          return;
        }
        setData({
          depots: depots.items ?? [],
          error: "",
          peers: registrations.items ?? [],
          loading: false,
          serverInfo,
        });
      } catch (error) {
        if (cancelled) {
          return;
        }
        setData((current) => ({
          ...current,
          error: toMessage(error),
          loading: false,
        }));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return data;
}
