import React, { useState, useEffect, useCallback, useMemo } from "react";
import { compileLanguage, createMs } from "enhanced-ms";
import de from "enhanced-ms/locales/de";
import en from "enhanced-ms/locales/en";
import fr from "enhanced-ms/locales/fr";
import it from "enhanced-ms/locales/it";
import { defaultLanguage, useLocale } from "@/app/store/i18nStore";
import { parseUserDuration } from "@/utils/durationUtils";

export interface DurationInputState {
  inputValue: string;
  isValid: boolean;
  isFocused: boolean;
  placeholder: string | null;
  handlers: {
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
    onFocus: () => void;
    onBlur: () => void;
  };
}

const DURATION_LANGUAGES = { de, en, fr, it } as const;
type DurationLanguage = keyof typeof DURATION_LANGUAGES;

const getLanguageKey = (locale: string): DurationLanguage => {
  const base = locale.split("-")[0]?.toLowerCase();
  if (base && Object.prototype.hasOwnProperty.call(DURATION_LANGUAGES, base)) {
    return base as DurationLanguage;
  }
  const fallback = Object.prototype.hasOwnProperty.call(
    DURATION_LANGUAGES,
    defaultLanguage
  )
    ? defaultLanguage
    : "en";
  return fallback as DurationLanguage;
};

type DurationMs = ReturnType<typeof createMs>;

const formatDuration = (ms: DurationMs, value: number): string =>
  value ? (ms(value) ?? "") : "";

const reformatInput = (ms: DurationMs, input: string): string | null => {
  try {
    const parsed = ms(input);
    if (parsed !== null && parsed >= 0) {
      return ms(parsed) || null;
    }
  } catch (err) {}
  return null;
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
  onChange: (milliseconds: number) => void,
  placeholderMs?: number
): DurationInputState => {
  const locale = useLocale();
  const languageKey = useMemo(() => getLanguageKey(locale), [locale]);
  const languageDefinition = DURATION_LANGUAGES[languageKey];
  const language = useMemo(
    () => compileLanguage(languageDefinition),
    [languageDefinition]
  );
  const ms = useMemo(
    () =>
      createMs({
        language: languageDefinition,
        formatOptions: { unitLimit: 1, includeMs: true },
      }),
    [languageDefinition]
  );
  const [inputValue, setInputValue] = useState("");
  const [isValid, setIsValid] = useState(true);
  const [isFocused, setIsFocused] = useState(false);
  const hasLocalEditRef = React.useRef(false);

  useEffect(() => {
    if (!isFocused || !hasLocalEditRef.current) {
      setInputValue(formatDuration(ms, value));
      setIsValid(true);
      hasLocalEditRef.current = false;
    }
  }, [value, isFocused, ms]);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const input = e.target.value;
      setInputValue(input);
      hasLocalEditRef.current = true;

      const result = parseUserDuration(input, language, ms);
      setIsValid(result.valid);
      if (result.valid) {
        onChange(result.ms);
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
    const trimmed = inputValue.trim();
    const reformatted = isValid && trimmed ? reformatInput(ms, trimmed) : null;
    if (reformatted) {
      setInputValue(reformatted);
      return;
    }
    setInputValue(formatDuration(ms, value));
    setIsValid(true);
  }, [inputValue, isValid, ms, value]);

  return {
    inputValue,
    isValid,
    isFocused,
    placeholder: placeholderMs ? (ms(placeholderMs) ?? null) : null,
    handlers: {
      onChange: handleChange,
      onFocus: handleFocus,
      onBlur: handleBlur,
    },
  };
};
