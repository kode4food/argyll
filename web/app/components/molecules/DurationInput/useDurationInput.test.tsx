import { renderHook, act } from "@testing-library/react";
import { useDurationInput } from "./useDurationInput";
import { useI18nStore } from "@/app/store/i18nStore";

describe("useDurationInput", () => {
  afterEach(() => {
    act(() => {
      useI18nStore.setState({ locale: "en-US" });
    });
  });

  it("initializes with empty input when value is 0", () => {
    const { result } = renderHook(() => useDurationInput(0, jest.fn()));

    expect(result.current.inputValue).toBe("");
    expect(result.current.isValid).toBe(true);
    expect(result.current.isFocused).toBe(false);
  });

  it("initializes with formatted duration when value is provided", () => {
    const { result } = renderHook(
      () => useDurationInput(60000, jest.fn()) // 1 minute
    );

    expect(result.current.inputValue).toBe("1 minute");
    expect(result.current.isValid).toBe(true);
  });

  it("formats duration in human-readable format", () => {
    const { result } = renderHook(
      () => useDurationInput(86400000, jest.fn()) // 1 day
    );

    expect(result.current.inputValue).toBe("1 day");
  });

  it("handles complex durations", () => {
    const { result } = renderHook(
      () => useDurationInput(90061000, jest.fn()) // 1 day, 1 hour, 1 minute, 1 second
    );

    expect(result.current.inputValue).toContain("day");
  });

  describe("onChange handler", () => {
    it("parses valid duration strings", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "2h" },
        } as any);
      });

      expect(result.current.isValid).toBe(true);
      expect(onChange).toHaveBeenCalledWith(7200000); // 2 hours in ms
    });

    it("handles clear input (empty string)", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(60000, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "" },
        } as any);
      });

      expect(result.current.isValid).toBe(true);
      expect(onChange).toHaveBeenCalledWith(0);
    });

    it("handles whitespace input", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(60000, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "   " },
        } as any);
      });

      expect(result.current.isValid).toBe(true);
      expect(onChange).toHaveBeenCalledWith(0);
    });

    it("marks invalid input as invalid", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "invalid123xyz" },
        } as any);
      });

      expect(result.current.isValid).toBe(false);
      expect(onChange).not.toHaveBeenCalled();
    });

    it("handles human-readable duration strings", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "2 hours" },
        } as any);
      });

      expect(result.current.isValid).toBe(true);
      expect(onChange).toHaveBeenCalledWith(7200000); // 2h in ms
    });

    it("parses localized duration strings", () => {
      useI18nStore.setState({ locale: "fr-CH" });
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "2 heures" },
        } as any);
      });

      expect(result.current.isValid).toBe(true);
      expect(onChange).toHaveBeenCalledWith(7200000);
    });

    it("handles negative durations as invalid", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "-5h" },
        } as any);
      });

      expect(result.current.isValid).toBe(false);
      expect(onChange).not.toHaveBeenCalled();
    });

    it("updates inputValue on change", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "5 days" },
        } as any);
      });

      expect(result.current.inputValue).toBe("5 days");
    });
  });

  describe("onFocus handler", () => {
    it("sets isFocused to true", () => {
      const { result } = renderHook(() => useDurationInput(60000, jest.fn()));

      act(() => {
        result.current.handlers.onFocus();
      });

      expect(result.current.isFocused).toBe(true);
    });
  });

  describe("onBlur handler", () => {
    it("sets isFocused to false", () => {
      const { result } = renderHook(() => useDurationInput(60000, jest.fn()));

      act(() => {
        result.current.handlers.onFocus();
      });

      expect(result.current.isFocused).toBe(true);

      act(() => {
        result.current.handlers.onBlur();
      });

      expect(result.current.isFocused).toBe(false);
    });

    it("formats valid input on blur", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "2h" },
        } as any);
      });

      act(() => {
        result.current.handlers.onBlur();
      });

      // Should reformat to human-readable format
      expect(result.current.inputValue).toBe("2 hours");
    });

    it("syncs back to value on blur when input is invalid", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onFocus();
        result.current.handlers.onChange({
          target: { value: "invalid" },
        } as any);
      });

      act(() => {
        result.current.handlers.onBlur();
      });

      expect(result.current.inputValue).toBe("");
    });

    it("syncs back to value on blur when input is cleared", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(60000, onChange));

      act(() => {
        result.current.handlers.onFocus();
        result.current.handlers.onChange({
          target: { value: "" },
        } as any);
      });

      act(() => {
        result.current.handlers.onBlur();
      });

      expect(result.current.inputValue).toBe("1 minute");
    });
  });

  describe("value synchronization", () => {
    it("syncs value when not focused", () => {
      const { result, rerender } = renderHook(
        ({ value }) => useDurationInput(value, jest.fn()),
        { initialProps: { value: 60000 } }
      );

      expect(result.current.inputValue).toBe("1 minute");

      rerender({ value: 3600000 }); // 1 hour

      expect(result.current.inputValue).toBe("1 hour");
    });

    it("does not sync value when focused after local edit", () => {
      const { result, rerender } = renderHook(
        ({ value }) => useDurationInput(value, jest.fn()),
        { initialProps: { value: 60000 } }
      );

      act(() => {
        result.current.handlers.onFocus();
        result.current.handlers.onChange({
          target: { value: "2h" },
        } as any);
      });

      const focusedValue = result.current.inputValue;

      rerender({ value: 3600000 });

      // Should not update while focused
      expect(result.current.inputValue).toBe(focusedValue);
    });

    it("syncs value after blur", () => {
      const { result, rerender } = renderHook(
        ({ value }) => useDurationInput(value, jest.fn()),
        { initialProps: { value: 60000 } }
      );

      act(() => {
        result.current.handlers.onFocus();
      });

      rerender({ value: 3600000 });

      act(() => {
        result.current.handlers.onBlur();
      });

      // After blur, should sync with new value
      expect(result.current.inputValue).toBe("1 hour");
    });
  });

  describe("handler stability", () => {
    it("maintains stable handler references", () => {
      const onChange = jest.fn();
      const { result, rerender } = renderHook(() =>
        useDurationInput(60000, onChange)
      );

      const firstHandlers = result.current.handlers;

      rerender();

      expect(result.current.handlers.onChange).toBe(firstHandlers.onChange);
      expect(result.current.handlers.onFocus).toBe(firstHandlers.onFocus);
      expect(result.current.handlers.onBlur).toBe(firstHandlers.onBlur);
    });
  });

  describe("edge cases", () => {
    it("handles blur with no value changes", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(60000, onChange));

      act(() => {
        result.current.handlers.onFocus();
      });

      act(() => {
        result.current.handlers.onBlur();
      });

      expect(result.current.isFocused).toBe(false);
      expect(onChange).not.toHaveBeenCalled();
    });

    it("handles blur after focus without value change", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(60000, onChange));

      const initialValue = result.current.inputValue;

      act(() => {
        result.current.handlers.onFocus();
      });

      act(() => {
        result.current.handlers.onBlur();
      });

      expect(result.current.inputValue).toBe(initialValue);
    });

    it("handles multiple consecutive blurs", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "3h" },
        } as any);
      });

      act(() => {
        result.current.handlers.onBlur();
      });

      const firstBlurValue = result.current.inputValue;

      act(() => {
        result.current.handlers.onBlur();
      });

      expect(result.current.inputValue).toBe(firstBlurValue);
    });

    it("handles blur after invalid then valid input", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "invalid" },
        } as any);
      });

      act(() => {
        result.current.handlers.onChange({
          target: { value: "1h" },
        } as any);
      });

      act(() => {
        result.current.handlers.onBlur();
      });

      expect(result.current.inputValue).toBe("1 hour");
      expect(result.current.isValid).toBe(true);
    });

    it("handles very large durations", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "365d" },
        } as any);
      });

      expect(result.current.isValid).toBe(true);
      expect(onChange).toHaveBeenCalledWith(31536000000); // 365 days in ms
    });

    it("handles blur with very large valid duration", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "30d" },
        } as any);
      });

      act(() => {
        result.current.handlers.onBlur();
      });

      expect(result.current.inputValue).toContain("day");
      expect(result.current.isValid).toBe(true);
    });

    it("handles special duration formats on blur", () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useDurationInput(0, onChange));

      act(() => {
        result.current.handlers.onChange({
          target: { value: "1.5h" },
        } as any);
      });

      act(() => {
        result.current.handlers.onBlur();
      });

      expect(result.current.isValid).toBe(true);
    });
  });
});
