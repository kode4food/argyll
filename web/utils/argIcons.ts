import {
  ArrowRight,
  ArrowLeft,
  CircleHelp,
  Lock,
  LucideIcon,
} from "lucide-react";

export type ArgType = "required" | "optional" | "const" | "output";

export interface ArgIconConfig {
  Icon: LucideIcon;
  className: string;
}

export const getArgIcon = (argType: ArgType): ArgIconConfig => {
  switch (argType) {
    case "required":
      return { Icon: ArrowRight, className: "arg-icon input" };
    case "optional":
      return { Icon: CircleHelp, className: "arg-icon optional" };
    case "const":
      return { Icon: Lock, className: "arg-icon const" };
    case "output":
      return { Icon: ArrowLeft, className: "arg-icon output" };
  }
};
