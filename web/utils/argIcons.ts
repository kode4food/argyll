import { ArrowRight, ArrowLeft, CircleHelp, LucideIcon } from "lucide-react";

export type ArgType = "required" | "optional" | "output";

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
    case "output":
      return { Icon: ArrowLeft, className: "arg-icon output" };
  }
};
