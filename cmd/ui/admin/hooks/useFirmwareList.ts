import { useCallback, useEffect, useState } from "react";

import { expectData, toMessage } from "../components/api";
import { listDepots, type Depot } from "@gizclaw/adminservice";

export interface FirmwareListState {
  depots: Depot[];
  error: string;
  loading: boolean;
}

export function useFirmwareList(): FirmwareListState & { reload: () => Promise<void> } {
  const [state, setState] = useState<FirmwareListState>({
    depots: [],
    error: "",
    loading: false,
  });

  const load = useCallback(async () => {
    setState((previous) => ({ ...previous, error: "", loading: true }));
    try {
      const list = await expectData(listDepots());
      setState({
        depots: list.items ?? [],
        error: "",
        loading: false,
      });
    } catch (error) {
      setState({
        depots: [],
        error: toMessage(error),
        loading: false,
      });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return { ...state, reload: load };
}
