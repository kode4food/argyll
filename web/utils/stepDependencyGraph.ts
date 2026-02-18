import { AttributeRole, Step } from "@/app/api";

export interface StepInput {
  name: string;
  isOptional: boolean;
}

export interface StepGraph {
  dependencies: Map<string, string[]>;
  stepsWithDependencies: Set<string>;
}

export const listStepInputs = (step: Step): StepInput[] => {
  return Object.entries(step.attributes || {})
    .filter(([, spec]) => {
      return (
        spec.role === AttributeRole.Required ||
        spec.role === AttributeRole.Optional
      );
    })
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, spec]) => ({
      name,
      isOptional: spec.role === AttributeRole.Optional,
    }));
};

export const buildOutputProducerMap = (
  steps: Step[]
): Map<string, string[]> => {
  const producerMap = new Map<string, string[]>();

  steps.forEach((step) => {
    Object.entries(step.attributes || {}).forEach(([name, spec]) => {
      if (spec.role !== AttributeRole.Output) {
        return;
      }

      const producers = producerMap.get(name);
      if (producers) {
        producers.push(step.id);
        return;
      }
      producerMap.set(name, [step.id]);
    });
  });

  return producerMap;
};

export const buildStepGraph = (
  steps: Step[],
  producerMap: Map<string, string[]>,
  activeStepIds?: Set<string> | null
): StepGraph => {
  const dependencies = new Map<string, string[]>();
  const stepsWithDependencies = new Set<string>();

  steps.forEach((step) => {
    dependencies.set(step.id, []);
  });

  steps.forEach((toStep) => {
    const toDeps = dependencies.get(toStep.id);
    if (!toDeps) {
      return;
    }

    listStepInputs(toStep).forEach((input) => {
      const producers = producerMap.get(input.name);
      if (!producers) {
        return;
      }

      producers.forEach((fromStepID) => {
        if (fromStepID === toStep.id) {
          return;
        }

        toDeps.push(fromStepID);
        if (
          activeStepIds &&
          activeStepIds.has(fromStepID) &&
          activeStepIds.has(toStep.id)
        ) {
          stepsWithDependencies.add(toStep.id);
        }
      });
    });
  });

  return { dependencies, stepsWithDependencies };
};

export const calculateStepLevels = (
  steps: Step[],
  dependencies: Map<string, string[]>
): Map<string, number> => {
  const levels = new Map<string, number>();
  const visited = new Set<string>();

  const calculateLevel = (stepID: string): number => {
    const cachedLevel = levels.get(stepID);
    if (cachedLevel !== undefined) {
      return cachedLevel;
    }
    if (visited.has(stepID)) {
      return 0;
    }

    visited.add(stepID);
    const deps = dependencies.get(stepID) || [];
    if (deps.length === 0) {
      levels.set(stepID, 0);
      return 0;
    }

    const maxDepLevel = Math.max(...deps.map((depID) => calculateLevel(depID)));
    const level = maxDepLevel + 1;
    levels.set(stepID, level);
    return level;
  };

  steps.forEach((step) => {
    calculateLevel(step.id);
  });

  return levels;
};
