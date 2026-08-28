import {
  Activity,
  AlertCircle,
  AlertTriangle,
  ArrowLeft,
  ArrowRightLeft,
  ArrowRight,
  Ban,
  Boxes,
  Braces,
  ChevronDown,
  ChevronUp,
  Award,
  CheckCircle,
  CheckCircle2,
  CircleDashed,
  CircleDot,
  CircleHelp,
  CircleSlash,
  Clock,
  Ellipsis,
  Command,
  NotebookPen,
  FileCode2,
  Filter,
  GitBranch,
  Globe,
  Info,
  HeartPulse,
  Layers,
  Link2,
  Loader2,
  Lock,
  Maximize,
  MinusCircle,
  Moon,
  Play,
  Plus,
  RefreshCw,
  Search,
  Server,
  Square,
  Sun,
  Tag,
  Tags,
  Target,
  Trash2,
  Undo2,
  Webhook,
  Wifi,
  WifiOff,
  Workflow,
  X,
  XCircle,
  type LucideIcon,
} from "lucide-react";
import { Step, StepType } from "@/app/api";

export type ArgType = "required" | "optional" | "const" | "meta" | "output";

export interface ArgIconConfig {
  Icon: LucideIcon;
  className: string;
}

export const IconAdd = Plus;
export const IconAddStep = Plus;
export const IconFitView = Maximize;
export const IconRemove = Trash2;
export const IconSearch = Search;
export const IconStartFlow = Play;
export const IconNavigateOverview = Activity;
export const IconThemeDark = Moon;
export const IconThemeLight = Sun;
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
export const IconStandard = Play;
export const IconMemoized = NotebookPen;
export const IconCompensate = Undo2;
export const IconEndpoint = Link2;
export const IconHealthCheck = HeartPulse;
export const IconMapping = ArrowRightLeft;
export const IconAttributeMatch = Filter;
export const IconExpandDown = ChevronDown;
export const IconExpandUp = ChevronUp;

export const IconProgressPending = Clock;
export const IconProgressActive = Loader2;
export const IconProgressCompleted = CheckCircle;
export const IconProgressFailed = XCircle;
export const IconProgressSkipped = MinusCircle;
export const IconCompensateFailed = AlertTriangle;

export const IconAttributeRequired = ArrowRight;
export const IconAttributeOptional = CircleHelp;
export const IconAttributeConst = Lock;
export const IconAttributeMeta = Tag;
export const IconAttributeOutput = ArrowLeft;
export const IconAttributeLabel = Tags;
export const IconSpace = Boxes;
export const IconManage = Ellipsis;

/* Step editor section headers */
export const IconAttributes = Braces;
export const IconFlowGoals = Target;
export const IconPredicate = GitBranch;

export const IconAttributeStatusSatisfied = CheckCircle2;
export const IconAttributeStatusMissing = AlertCircle;
export const IconAttributeStatusFailed = XCircle;
export const IconAttributeStatusNotWinner = XCircle;
export const IconAttributeStatusWinner = Award;
export const IconAttributeStatusBlocked = Ban;
export const IconAttributeStatusPending = CircleDashed;
export const IconAttributeStatusProvided = CheckCircle;
export const IconAttributeStatusDefaulted = CircleDot;
export const IconAttributeStatusSkipped = CircleSlash;

export const IconStepTypeService = Server;
export const IconStepTypeScript = FileCode2;
export const IconStepTypeFlow = Workflow;

export const IconActionModeSync = Globe;
export const IconActionModeAsync = Webhook;

const ARG_ICON_MAP: Record<ArgType, ArgIconConfig> = {
  required: { Icon: IconAttributeRequired, className: "arg-icon input" },
  optional: { Icon: IconAttributeOptional, className: "arg-icon optional" },
  const: { Icon: IconAttributeConst, className: "arg-icon const" },
  meta: { Icon: IconAttributeMeta, className: "arg-icon meta" },
  output: { Icon: IconAttributeOutput, className: "arg-icon output" },
};

export const getArgIcon = (argType: ArgType): ArgIconConfig => {
  return ARG_ICON_MAP[argType];
};

const STEP_TYPE_ICON_MAP: Record<StepType, LucideIcon> = {
  service: IconStepTypeService,
  script: IconStepTypeScript,
  flow: IconStepTypeFlow,
};

export const getStepTypeIcon = (stepType: StepType): LucideIcon => {
  return STEP_TYPE_ICON_MAP[stepType];
};

// Service Steps show how their invocation reports its result, since that is
// what distinguishes one from another at a glance
export const getStepActionIcon = (step: Step): LucideIcon => {
  if (step.type !== "service") {
    return getStepTypeIcon(step.type);
  }
  return step.http?.invoke?.mode === "async"
    ? IconActionModeAsync
    : IconActionModeSync;
};

export type { LucideIcon };
