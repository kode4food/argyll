import { useEffect, useState } from "react";
import { api, type AgentCapabilities } from "@/app/api";

export const useAgentCapabilities = (): AgentCapabilities => {
  const [capabilities, setCapabilities] = useState<AgentCapabilities>({
    providers: [],
  });

  useEffect(() => {
    const controller = new AbortController();
    api
      .listAgentCapabilities(controller.signal)
      .then((list) => !controller.signal.aborted && setCapabilities(list))
      .catch((error: unknown) => {
        if (!controller.signal.aborted) {
          setCapabilities({
            providers: [],
            problems: [{ detail: String(error) }],
          });
        }
      });
    return () => {
      controller.abort();
    };
  }, []);

  return capabilities;
};
