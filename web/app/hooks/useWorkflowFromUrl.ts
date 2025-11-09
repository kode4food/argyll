import { useParams, usePathname } from "next/navigation";
import { useEffect } from "react";
import { useSelectWorkflow } from "../store/workflowStore";

export const useWorkflowFromUrl = () => {
  const params = useParams();
  const pathname = usePathname();
  const flowId = params?.flowId as string;
  const selectWorkflow = useSelectWorkflow();

  useEffect(() => {
    if (pathname.startsWith("/workflow/") && flowId) {
      selectWorkflow(flowId);
    } else if (pathname === "/") {
      selectWorkflow(null);
    }
  }, [flowId, pathname, selectWorkflow]);

  return flowId || null;
};
