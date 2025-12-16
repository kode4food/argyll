import { useParams, usePathname } from "next/navigation";
import { useEffect } from "react";
import { useSelectFlow } from "@/app/store/flowStore";

export const useFlowFromUrl = () => {
  const params = useParams();
  const pathname = usePathname();
  const flowId = params?.flowId as string;
  const selectFlow = useSelectFlow();

  useEffect(() => {
    if (pathname.startsWith("/flow/") && flowId) {
      selectFlow(flowId);
    } else if (pathname === "/") {
      selectFlow(null);
    }
  }, [flowId, pathname, selectFlow]);

  return flowId || null;
};
