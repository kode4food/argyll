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
  handlers: {
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
    onFocus: () => void;
    onBlur: () => void;
  };
}

const durationLanguages = { de, en, fr, it } as const;
type DurationLanguage = keyof typeof durationLanguages;

const getLanguageKey = (locale: string): DurationLanguage => {
  const base = locale.split("-")[0]?.toLowerCase();
  if (base && Object.prototype.hasOwnProperty.call(durationLanguages, base)) {
    return base as DurationLanguage;
  }
  const fallback = Object.prototype.hasOwnProperty.call(
    durationLanguages,
    defaultLanguage
  )
    ? defaultLanguage
    : "en";
  return fallback as DurationLanguage;
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
  const languageDefinition = durationLanguages[languageKey];
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
    if (isValid && inputValue.trim()) {
      try {
        const parsed = ms(inputValue.trim());
        if (parsed !== null && parsed >= 0) {
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
