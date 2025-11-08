import { useParams, usePathname } from "next/navigation";
import { useEffect } from "react";
import { useSelectWorkflow } from "../store/workflowStore";

export const useWorkflowFromUrl = () => {
  const params = useParams();
  const pathname = usePathname();
  const workflowId = params?.workflowId as string;
  const selectWorkflow = useSelectWorkflow();

  useEffect(() => {
    if (pathname.startsWith("/workflow/") && workflowId) {
      selectWorkflow(workflowId);
    } else if (pathname === "/") {
      selectWorkflow(null);
    }
  }, [workflowId, pathname, selectWorkflow]);

  return workflowId || null;
};
