import React, {
  useRef,
  useEffect,
  useState,
  useCallback,
  useMemo,
} from "react";
import { Position, NodeProps } from "@xyflow/react";
import { Step, FlowContext, ExecutionResult, AttributeRole } from "../../api";
import StepWidget from "./StepWidget";
import InvisibleHandle from "../atoms/InvisibleHandle";
import { useDiagramSelection } from "../../contexts/DiagramSelectionContext";

interface StepNodeData {
  step: Step;
  selected: boolean;
  flowData?: FlowContext | null;
  executions?: ExecutionResult[];
  resolvedAttributes?: string[];
  isGoalStep?: boolean;
  isInPreviewPlan?: boolean;
  isPreviewMode?: boolean;
  isStartingPoint?: boolean;
  onStepClick?: (stepId: string, options?: { additive?: boolean }) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
  disableEdit?: boolean;
}

const StepNode: React.FC<NodeProps> = ({ data }) => {
  const nodeData = data as unknown as StepNodeData;
  const {
    step,
    flowData,
    executions = [],
    resolvedAttributes = [],
    onStepClick,
  } = nodeData;
  const { setGoalSteps } = useDiagramSelection();
  const stepWidgetRef = useRef<HTMLDivElement>(null);

  // Memoize the click handler to prevent unnecessary re-renders
  const handleClick = useCallback(
    (event: React.MouseEvent) => {
      const additive = event.ctrlKey || event.metaKey;
      if (onStepClick) {
        onStepClick(step.id, { additive });
      } else if (!additive) {
        setGoalSteps([step.id]);
      }
    },
    [onStepClick, setGoalSteps, step.id]
  );
  const [handlePositions, setHandlePositions] = useState<{
    required: Array<{
      id: string;
      top: number;
      argName: string;
      handleType: "input" | "output";
    }>;
    optional: Array<{
      id: string;
      top: number;
      argName: string;
      handleType: "input" | "output";
    }>;
    output: Array<{
      id: string;
      top: number;
      argName: string;
      handleType: "input" | "output";
    }>;
  }>({ required: [], optional: [], output: [] });

  const execution = useMemo(
    () => executions.find((exec) => exec.step_id === step.id),
    [executions, step.id]
  );

  const resolved = useMemo(
    () => new Set(resolvedAttributes),
    [resolvedAttributes]
  );

  const provenance = useMemo(() => {
    const map = new Map<string, string>();
    if (flowData?.state) {
      Object.entries(flowData.state).forEach(([attrName, attrValue]) => {
        if (attrValue.step) {
          map.set(attrName, attrValue.step);
        }
      });
    }
    return map;
  }, [flowData?.state]);

  const satisfied = useMemo(() => {
    const set = new Set<string>();
    Object.entries(step.attributes || {}).forEach(([argName, spec]) => {
      if (
        (spec.role === AttributeRole.Required ||
          spec.role === AttributeRole.Optional) &&
        resolved.has(argName)
      ) {
        set.add(argName);
      }
    });
    return set;
  }, [step.attributes, resolved]);

  const updateHandlePositions = useCallback(() => {
    const sortedAttrs = Object.entries(step.attributes || {}).sort(([a], [b]) =>
      a.localeCompare(b)
    );

    const requiredArgs = sortedAttrs
      .filter(([_, spec]) => spec.role === AttributeRole.Required)
      .map(([name]) => name);
    const optionalArgs = sortedAttrs
      .filter(([_, spec]) => spec.role === AttributeRole.Optional)
      .map(([name]) => name);
    const outputArgs = sortedAttrs
      .filter(([_, spec]) => spec.role === AttributeRole.Output)
      .map(([name]) => name);

    if (!stepWidgetRef.current) return;

    const getHandlePosition = (
      element: Element,
      type: string,
      name: string,
      handleType: "input" | "output"
    ) => {
      const relativeTop =
        (element as HTMLElement).offsetTop +
        (element as HTMLElement).offsetHeight / 2;
      return {
        id: type === "output" ? `output-${name}` : `input-${type}-${name}`,
        top: relativeTop,
        argName: name,
        handleType,
      };
    };

    const findHandles = (
      argType: string,
      argNames: string[],
      handleType: "input" | "output"
    ) => {
      return argNames
        .map((name) => {
          const element = stepWidgetRef.current?.querySelector(
            `[data-arg-type="${argType}"][data-arg-name="${name}"]`
          );
          return element
            ? getHandlePosition(element, argType, name, handleType)
            : null;
        })
        .filter(
          (
            handle
          ): handle is {
            id: string;
            top: number;
            argName: string;
            handleType: "input" | "output";
          } => handle !== null
        );
    };

    setHandlePositions({
      required: findHandles("required", requiredArgs, "input"),
      optional: findHandles("optional", optionalArgs, "input"),
      output: findHandles("output", outputArgs, "output"),
    });
  }, [step.attributes]);

  useEffect(() => {
    updateHandlePositions();
  }, [updateHandlePositions]);

  const allHandles = useMemo(
    () => [
      ...handlePositions.required,
      ...handlePositions.optional,
      ...handlePositions.output,
    ],
    [handlePositions]
  );

  return (
    <div className="step-node relative">
      {allHandles.map((handle) => (
        <InvisibleHandle
          key={handle.id}
          id={handle.id}
          type={handle.handleType === "output" ? "source" : "target"}
          position={
            handle.handleType === "output" ? Position.Right : Position.Left
          }
          top={handle.top}
          argName={handle.argName}
        />
      ))}

      <div ref={stepWidgetRef}>
        <StepWidget
          step={step}
          selected={nodeData.selected}
          onClick={handleClick}
          mode="diagram"
          className={`${nodeData.isGoalStep ? "goal" : ""} ${nodeData.isStartingPoint ? "start-point" : ""}`}
          execution={execution}
          satisfiedArgs={satisfied}
          attributeProvenance={provenance}
          isInPreviewPlan={nodeData.isInPreviewPlan}
          isPreviewMode={nodeData.isPreviewMode}
          flowId={flowData?.id}
          diagramContainerRef={nodeData.diagramContainerRef}
          disableEdit={nodeData.disableEdit}
        />
      </div>
    </div>
  );
};

export default React.memo(StepNode);
