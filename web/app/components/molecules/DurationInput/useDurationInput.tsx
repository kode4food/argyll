import React, { useState, useEffect, useCallback } from "react";
import ms from "ms";
import type { StringValue } from "ms";

export interface DurationInputState {
  inputValue: string;
  isValid: boolean;
  isFocused: boolean;
  handlers: {
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
    onFocus: () => void;
    onBlur: () => void;
  };
}

/**
 * Hook that manages all state and logic for duration input
 * Handles parsing, validation, and formatting using the `ms` library
 *
 * @param value - Current duration value in milliseconds
 * @param onChange - Callback when user changes the duration
 * @returns Object containing input state and event handlers
 */
export const useDurationInput = (
  value: number,
  onChange: (milliseconds: number) => void
): DurationInputState => {
  const [inputValue, setInputValue] = useState("");
  const [isValid, setIsValid] = useState(true);
  const [isFocused, setIsFocused] = useState(false);

  useEffect(() => {
    if (!isFocused) {
      if (value) {
        setInputValue(ms(value, { long: true }));
      } else {
        setInputValue("");
      }
      setIsValid(true);
    }
  }, [value, isFocused]);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const input = e.target.value;
      setInputValue(input);

      if (!input.trim()) {
        setIsValid(true);
        onChange(0);
        return;
      }

      try {
        const parsed = ms(input as StringValue);
        if (parsed >= 0 && parsed !== undefined) {
          setIsValid(true);
          onChange(parsed);
        } else {
          setIsValid(false);
        }
      } catch (err) {
        setIsValid(false);
      }
    },
    [onChange]
  );

  const handleFocus = useCallback(() => {
    setIsFocused(true);
  }, []);

  const handleBlur = useCallback(() => {
    setIsFocused(false);
    if (isValid && inputValue.trim()) {
      try {
        const parsed = ms(inputValue as StringValue);
        if (parsed >= 0 && parsed !== undefined) {
          setInputValue(ms(parsed, { long: true }));
        }
      } catch (err) {}
    }
  }, [isValid, inputValue]);

  return {
    inputValue,
    isValid,
    isFocused,
    handlers: {
      onChange: handleChange,
      onFocus: handleFocus,
      onBlur: handleBlur,
    },
  };
};
