import React, { useState, useEffect, useCallback, useMemo } from "react";
import { createMs, getLanguage, languages } from "enhanced-ms";
import { defaultLanguage, useLocale } from "@/app/store/i18nStore";

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

const getLanguageKey = (locale: string): keyof typeof languages => {
  const base = locale.split("-")[0]?.toLowerCase();
  if (base && Object.prototype.hasOwnProperty.call(languages, base)) {
    return base as keyof typeof languages;
  }
  if (Object.prototype.hasOwnProperty.call(languages, defaultLanguage)) {
    return defaultLanguage as keyof typeof languages;
  }
  return "en";
};

const isNumericOnly = (value: string, decimalSeparator: "." | ",") => {
  const escaped = decimalSeparator === "," ? "," : "\\.";
  const regex = new RegExp(`^\\d+(?:[${escaped}]\\d+)?$`);
  return regex.test(value);
};

const tryParseNumber = (value: string, decimalSeparator: "." | ",") => {
  const normalized = decimalSeparator === "," ? value.replace(",", ".") : value;
  const parsed = Number(normalized);
  return Number.isNaN(parsed) ? null : parsed;
};

const hasUnitToken = (value: string, matcherRegex: RegExp) => {
  if (matcherRegex.global) {
    matcherRegex.lastIndex = 0;
  }
  const matches = value.match(matcherRegex) ?? [];
  return matches.some((match) => !/[0-9]/.test(match));
};

const hasNumberAndUnit = (value: string, matcherRegex: RegExp) => {
  if (!/[0-9]/.test(value)) {
    return false;
  }
  return hasUnitToken(value, matcherRegex);
};

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
  const locale = useLocale();
  const languageKey = useMemo(() => getLanguageKey(locale), [locale]);
  const language = useMemo(() => getLanguage(languageKey), [languageKey]);
  const ms = useMemo(
    () =>
      createMs({
        language: languageKey,
        formatOptions: { unitLimit: 1, includeMs: true },
      }),
    [languageKey]
  );
  const [inputValue, setInputValue] = useState("");
  const [isValid, setIsValid] = useState(true);
  const [isFocused, setIsFocused] = useState(false);
  const hasLocalEditRef = React.useRef(false);

  useEffect(() => {
    if (!isFocused || !hasLocalEditRef.current) {
      if (value) {
        const formatted = ms(value);
        setInputValue(formatted ?? "");
      } else {
        setInputValue("");
      }
      setIsValid(true);
      hasLocalEditRef.current = false;
    }
  }, [value, isFocused, ms]);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const input = e.target.value;
      setInputValue(input);
      hasLocalEditRef.current = true;

      if (!input.trim()) {
        setIsValid(true);
        onChange(0);
        return;
      }

      try {
        const trimmed = input.trim();
        if (trimmed.startsWith("-")) {
          setIsValid(false);
          return;
        }

        if (isNumericOnly(trimmed, language.decimalSeparator)) {
          const numeric = tryParseNumber(trimmed, language.decimalSeparator);
          if (numeric === null || numeric < 0) {
            setIsValid(false);
            return;
          }
          setIsValid(true);
          onChange(numeric);
          return;
        }

        const hasUnitMatch = hasNumberAndUnit(
          trimmed.toLowerCase(),
          language.matcherRegex
        );
        if (!hasUnitMatch) {
          setIsValid(false);
          return;
        }

        const parsed = ms(trimmed);
        if (parsed >= 0) {
          setIsValid(true);
          onChange(parsed);
          return;
        }

        setIsValid(false);
      } catch (err) {
        setIsValid(false);
      }
    },
    [language, ms, onChange]
  );

  const handleFocus = useCallback(() => {
    setIsFocused(true);
  }, []);

  const handleBlur = useCallback(() => {
    setIsFocused(false);
    hasLocalEditRef.current = false;
    if (isValid && inputValue.trim()) {
      try {
        const parsed = ms(inputValue.trim());
        if (parsed >= 0) {
          const formatted = ms(parsed);
          if (formatted) {
            setInputValue(formatted);
            return;
          }
        }
      } catch (err) {}
    }
    if (value) {
      const formatted = ms(value);
      setInputValue(formatted ?? "");
    } else {
      setInputValue("");
    }
    setIsValid(true);
  }, [inputValue, isValid, ms, value]);

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
