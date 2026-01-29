import {
  Activity,
  AlertCircle,
  AlertTriangle,
  ArrowLeft,
  ArrowRight,
  Award,
  Ban,
  CheckCircle,
  CheckCircle2,
  CircleDashed,
  CircleDot,
  CircleHelp,
  CircleSlash,
  Clock,
  Command,
  Database,
  FileCode2,
  Globe,
  Info,
  Layers,
  Loader2,
  Lock,
  MinusCircle,
  Play,
  Plus,
  RefreshCw,
  Search,
  Server,
  Square,
  Trash2,
  Webhook,
  Wifi,
  WifiOff,
  Workflow,
  X,
  XCircle,
  type LucideIcon,
} from "lucide-react";
import { StepType } from "@/app/api";

export type ArgType = "required" | "optional" | "const" | "output";

export interface ArgIconConfig {
  Icon: LucideIcon;
  className: string;
}

export const IconAdd = Plus;
export const IconAddStep = Plus;
export const IconRemove = Trash2;
export const IconSearch = Search;
export const IconStartFlow = Play;
export const IconCreateFlow = Play;
export const IconNavigateOverview = Activity;
export const IconEmptyState = Server;
export const IconDiagramEmptyState = Server;
export const IconDiagramLoading = Server;
export const IconInfo = Info;
export const IconDuration = Clock;
export const IconClose = X;
export const IconCommandKey = Command;
export const IconPageNotFound = AlertTriangle;
export const IconError = AlertCircle;
export const IconRetry = RefreshCw;
export const IconFlowNotFound = AlertCircle;
export const IconConnectionOnline = Wifi;
export const IconConnectionOffline = WifiOff;
export const IconConnectionReconnecting = RefreshCw;
export const IconConnectionError = AlertCircle;
export const IconArraySingle = Square;
export const IconArrayMultiple = Layers;
export const IconMemoizable = Database;

export const IconProgressPending = Clock;
export const IconProgressActive = Loader2;
export const IconProgressCompleted = CheckCircle;
export const IconProgressFailed = XCircle;
export const IconProgressSkipped = MinusCircle;

export const IconAttributeRequired = ArrowRight;
export const IconAttributeOptional = CircleHelp;
export const IconAttributeConst = Lock;
export const IconAttributeOutput = ArrowLeft;

export const IconAttributeStatusSatisfied = CheckCircle2;
export const IconAttributeStatusFailed = XCircle;
export const IconAttributeStatusNotWinner = XCircle;
export const IconAttributeStatusWinner = Award;
export const IconAttributeStatusBlocked = Ban;
export const IconAttributeStatusPending = CircleDashed;
export const IconAttributeStatusProvided = CheckCircle;
export const IconAttributeStatusDefaulted = CircleDot;
export const IconAttributeStatusSkipped = CircleSlash;

export const IconStepTypeSync = Globe;
export const IconStepTypeAsync = Webhook;
export const IconStepTypeScript = FileCode2;
export const IconStepTypeFlow = Workflow;

const argIconMap: Record<ArgType, ArgIconConfig> = {
  required: { Icon: IconAttributeRequired, className: "arg-icon input" },
  optional: { Icon: IconAttributeOptional, className: "arg-icon optional" },
  const: { Icon: IconAttributeConst, className: "arg-icon const" },
  output: { Icon: IconAttributeOutput, className: "arg-icon output" },
};

export const getArgIcon = (argType: ArgType): ArgIconConfig => {
  return argIconMap[argType];
};

const stepTypeIconMap: Record<StepType, LucideIcon> = {
  sync: IconStepTypeSync,
  async: IconStepTypeAsync,
  script: IconStepTypeScript,
  flow: IconStepTypeFlow,
};

export const getStepTypeIcon = (stepType: StepType): LucideIcon => {
  return stepTypeIconMap[stepType];
};

export type { LucideIcon };
