# TypeScript/React Style Guide

## Naming Conventions

### Components

PascalCase, noun describing what it renders:

```typescript
// Good
const FlowSelector: React.FC<Props> = () => { ... };
const StepHeader: React.FC<Props> = () => { ... };
const HealthDot: React.FC<Props> = () => { ... };

// Bad
const RenderFlow: React.FC<Props> = () => { ... };  // Verb prefix
const flowSelector: React.FC<Props> = () => { ... }; // camelCase
```

### Props

`ComponentNameProps` suffix:

```typescript
// Good
interface FlowSelectorProps {
  flowId: string;
  onSelect: (id: string) => void;
}

interface StepHeaderProps {
  stepId: string;
  status: StepStatus;
}

// Bad
interface FlowSelectorOptions { ... }  // Wrong suffix
interface IFlowSelectorProps { ... }   // No I prefix
```

### Hooks

`use` prefix, describes what it provides:

```typescript
// Good
const useFlowState = (flowId: string) => { ... };
const useStepProgress = (stepId: string) => { ... };
const useWebSocket = (url: string) => { ... };

// Bad
const getFlowState = (flowId: string) => { ... };    // Missing use prefix
const flowStateHook = (flowId: string) => { ... };   // Wrong pattern
```

### Event Handlers

`handle` prefix for implementations, `on` prefix for props:

```typescript
// Good - handler implementation
const handleClick = () => { ... };
const handleSubmit = (e: FormEvent) => { ... };
const handleFlowSelect = (flowId: string) => { ... };

// Good - prop definition
interface Props {
  onClick: () => void;
  onSubmit: (data: FormData) => void;
  onFlowSelect: (flowId: string) => void;
}

// Bad
const clickHandler = () => { ... };     // Wrong pattern
const onClickHandler = () => { ... };   // Mixed patterns
```

### Boolean Variables

`is`, `has`, `should`, `can` prefixes:

```typescript
// Good
const isLoading = status === "loading";
const hasError = error !== null;
const shouldRefresh = staleTime > threshold;
const canSubmit = isValid && !isLoading;

// Bad
const loading = status === "loading";    // Ambiguous
const error = error !== null;            // Shadows type
const refresh = staleTime > threshold;   // Unclear it's boolean
```

### Constants

SCREAMING_SNAKE_CASE for true constants, camelCase for derived values:

```typescript
// Good - true constants
const MAX_RETRIES = 3;
const DEFAULT_TIMEOUT_MS = 5000;
const API_BASE_URL = "/engine";

// Good - derived values
const defaultTimeout = config.timeout ?? DEFAULT_TIMEOUT_MS;
const maxRetries = options.retries ?? MAX_RETRIES;

// Bad
const MaxRetries = 3;          // Wrong case
const max_retries = 3;         // Wrong case
```

### Types and Interfaces

PascalCase. Prefer `interface` for object shapes, `type` for unions/aliases:

```typescript
// Good
interface FlowState {
  id: string;
  status: FlowStatus;
}

type FlowStatus = "active" | "completed" | "failed";
type FlowId = string;

// Bad
type FlowStateType = { ... };  // Use interface for objects
interface FlowStatusInterface { ... }  // Use type for unions
```

### File Names

kebab-case for utilities, PascalCase for components:

```
// Good
FlowSelector.tsx
FlowSelector.test.tsx
FlowSelector.module.css
flowSelectorUtils.ts
useFlowState.ts

// Bad
flow-selector.tsx       // Components use PascalCase
FlowSelectorUtils.ts    // Utils use camelCase
flow_selector.tsx       // No underscores
```

### Abbreviations

Common abbreviations are preferred when they're clear:

```typescript
// Good - common abbreviations
const stepIdx = 0;
const urlPfx = "/api/v1";
const nameSfx = "-test";
const ctx = useContext(FlowContext);
const props = { ...defaultProps };
const handleClick = (e: MouseEvent) => {};
const config = loadConfig();
const opts = { timeout: 5000 };

// Good - full names for clarity at boundaries
interface FlowSelectorProps {  // Props suffix stays full
  flowId: string;              // Id suffix stays full
  onSelect: (id: string) => void;
}

// Bad - awkward or unclear
const flId = "flow-123";       // Weird truncation
const stp = steps[0];          // Too terse
const configuration = load();  // Just use config
```

**Common abbreviations**:

| Abbreviation | Full Name |
|--------------|-----------|
| `idx` | index |
| `pfx` | prefix |
| `sfx` | suffix |
| `ctx` | context |
| `config` | configuration |
| `opts` | options |
| `props` | properties |
| `e` | event |
| `el` | element |
| `ref` | reference |
| `prev` | previous |
| `curr` | current |

## Pre-Commit Checklist

Run before every commit:

```bash
npm run format && npm test && npm run lint && npm run type-check
```

## Directory Structure (Atomic Design)

```
app/
├── api/              # API client and types
├── components/
│   ├── atoms/        # Smallest reusable elements (Button, Spinner, Dot)
│   ├── molecules/    # Simple combinations (InputGroup, HeaderRow)
│   ├── organisms/    # Complex features (FlowSelector, StepEditor)
│   └── templates/    # Page layouts
├── contexts/         # React Context providers
├── hooks/            # Custom hooks
├── store/            # Zustand stores
└── types/            # Shared TypeScript types
utils/                # Pure utility functions
```

## Component Principles

### Pure Components

Components should be presentational with minimal logic:

```typescript
// Good - pure component
const HealthDot: React.FC<{ health: Health }> = ({ health }) => (
  <span className={styles[health]} />
);

// Bad - business logic in component
const HealthDot: React.FC<{ step: Step }> = ({ step }) => {
  const health = calculateHealthFromStep(step); // Move to hook or parent
  return <span className={styles[health]} />;
};
```

### Colocation

Keep one-time-use helpers close to their components:

```typescript
// In ComponentName.tsx, above the component
const formatDisplayValue = (value: number): string => {
  return value.toFixed(2);
};

const ComponentName: React.FC<Props> = ({ value }) => (
  <span>{formatDisplayValue(value)}</span>
);
```

Only move to `utils/` when reused across multiple components.

## Props and Types

### Props Interface

Define props interface at top of file, export if needed by parent:

```typescript
export interface StepHeaderProps {
  stepId: string;
  status: StepStatus;
  onExpand?: () => void;
}

const StepHeader: React.FC<StepHeaderProps> = ({
  stepId,
  status,
  onExpand,
}) => { ... };
```

### Naming

- Props: `ComponentNameProps`
- Context: `ComponentNameContextType`
- Store state: `StoreNameState`

## Hooks

### Custom Hooks

Extract complex logic into hooks:

```typescript
// Good - logic in hook
export const useStepProgress = (stepId: string) => {
  const executions = useExecutions();
  return useMemo(() => calculateProgress(executions, stepId), [
    executions,
    stepId,
  ]);
};

// Component stays pure
const StepProgress: React.FC<{ stepId: string }> = ({ stepId }) => {
  const progress = useStepProgress(stepId);
  return <ProgressBar value={progress} />;
};
```

### Memoization

Use `useMemo` for expensive calculations, `useCallback` for handlers:

```typescript
const handleClick = useCallback(() => {
  onSelect(item.id);
}, [onSelect, item.id]);

const sortedItems = useMemo(
  () => items.slice().sort((a, b) => a.name.localeCompare(b.name)),
  [items]
);
```

## State Management

### Context Pattern

Each context exports a provider and hook:

```typescript
const UIContext = createContext<UIContextType | undefined>(undefined);

export const useUI = (): UIContextType => {
  const ctx = useContext(UIContext);
  if (!ctx) {
    throw new Error("useUI must be used within UIProvider");
  }
  return ctx;
};

export const UIProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const value = useMemo(() => ({ ... }), [deps]);
  return <UIContext.Provider value={value}>{children}</UIContext.Provider>;
};
```

### Zustand Store

Use for global app state:

```typescript
interface FlowState {
  flows: Flow[];
  loadFlows: () => Promise<void>;
}

const useFlowStore = create<FlowState>()(
  devtools((set) => ({
    flows: [],
    loadFlows: async () => {
      const flows = await api.getFlows();
      set({ flows });
    },
  }))
);

// Export selectors
export const useFlows = () => useFlowStore((s) => s.flows);
```

## Styling

### CSS Modules

One module per component:

```typescript
import styles from "./Button.module.css";

<button className={styles.primary} />
```

### Conditional Classes

Use filter/join pattern:

```typescript
const className = [
  styles.item,
  isSelected && styles.selected,
  isDisabled && styles.disabled,
]
  .filter(Boolean)
  .join(" ");
```

## Imports

### Absolute Imports

Use `@/` prefix for all non-relative imports:

```typescript
import { useUI } from "@/app/contexts/UIContext";
import { Step } from "@/app/api";
import { formatDate } from "@/utils/dates";
```

### Relative for Siblings

Use relative for same-directory imports:

```typescript
import styles from "./Component.module.css";
import { helperFn } from "./utils";
```

## Testing

### Coverage Target

Minimum 90% test coverage.

### Test Location

Colocate tests with source files:

```
Component.tsx
Component.test.tsx
Component.module.css
```

### Test Naming

Describe block is component/hook name. Test descriptions concise:

```typescript
// Good - short, clear descriptions
describe("FlowSelector", () => {
  test("renders flow list", () => { ... });
  test("calls onSelect when clicked", () => { ... });
  test("shows loading state", () => { ... });
});

// Bad - descriptions are novels
describe("FlowSelector", () => {
  test("should render the flow list correctly when flows are provided", () => { ... });
  test("should call the onSelect callback when a flow item is clicked", () => { ... });
});
```

### Assertions

Never include message arguments in expect matchers:

```typescript
// Good
expect(result).toBe(expected);
expect(screen.getByText("Submit")).toBeInTheDocument();

// Bad - no message arguments
expect(result).toBe(expected, "should match expected value");
```

### Component Tests

```typescript
describe("ComponentName", () => {
  test("renders with props", () => {
    render(<ComponentName prop="value" />);
    expect(screen.getByText("value")).toBeInTheDocument();
  });
});
```

### Hook Tests

```typescript
describe("useCustomHook", () => {
  test("returns expected value", () => {
    const { result } = renderHook(() => useCustomHook());
    expect(result.current.value).toBe(expected);
  });
});
```

## Code Splitting

### Lazy Loading

Use for large organisms:

```typescript
const FlowEditor = lazy(() => import("./FlowEditor"));

<Suspense fallback={<Spinner />}>
  <FlowEditor />
</Suspense>
```

## Error Handling

### Error Boundaries

Wrap feature areas:

```typescript
<ErrorBoundary title="Flow Editor" onError={logError}>
  <FlowEditor />
</ErrorBoundary>
```

### Async Operations

Use AbortController for cancellable requests:

```typescript
useEffect(() => {
  const controller = new AbortController();
  fetchData({ signal: controller.signal });
  return () => controller.abort();
}, []);
```

## Formatting (Prettier)

- Double quotes for strings
- Semicolons required
- 80 character line width
- 2 space indentation
- Trailing commas (es5)
- Parentheses around arrow function params
