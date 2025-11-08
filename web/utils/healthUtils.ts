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

export const getHealthStatusText = (status: HealthStatus, error?: string) => {
  switch (status) {
    case "healthy":
      return "Healthy";
    case "unhealthy":
      return error || "Unhealthy";
    case "unconfigured":
      return "No health check configured";
    default:
      return "Unknown";
  }
};
