import { HealthStatus, StepType } from "@/app/api";

export const getHealthIconClass = (
  status: HealthStatus,
  _stepType?: StepType
) => {
  switch (status) {
    case "healthy":
      return "healthy";
    case "unhealthy":
      return "unhealthy";
    case "unconfigured":
      return "unconfigured";
    default:
      return "unknown";
  }
};
