import { useEffect, useState } from "react";
import { api, type AgentProvider } from "@/app/api";

export const useAgentCapabilities = (): AgentProvider[] => {
  const [providers, setProviders] = useState<AgentProvider[]>([]);

  useEffect(() => {
    let active = true;
    api
      .listAgentCapabilities()
      .then((list) => active && setProviders(list))
      .catch(() => active && setProviders([]));
    return () => {
      active = false;
    };
  }, []);

  return providers;
};
