import React from "react";
import Tagify from "@yaireo/tagify";
import "@yaireo/tagify/dist/tagify.css";
import { type LucideIcon } from "@/utils/iconRegistry";
import styles from "./TagInput.module.css";

const MAX_SUGGESTIONS = 8;
const MIN_SUGGESTION_CHARACTERS = 1;

export interface TagInputProps {
  // Rejects any tag that is not one of the suggestions
  shouldEnforceSuggestions?: boolean;
  Icon: LucideIcon;
  // Omitted when the caller heads a group of inputs with a label of its own
  label?: string;
  onChange: (tags: string[]) => void;
  placeholder: string;
  removeLabel: string;
  shouldFocus?: boolean;
  shouldShowFieldIcon?: boolean;
  suggestions: readonly string[];
  tags: string[];
}

// Present on Tagify's dropdown, though its type declarations leave it out
interface DropdownInternals {
  highlightOption: (elm: Element) => void;
}

const valuesOf = (tagify: Tagify) => tagify.value.map((tag) => tag.value);

// The same close icon the rest of the app uses, since Tagify's own "×" glyph
// rides high in its em box and cannot be centred by alignment
const REMOVE_ICON = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"
  stroke-width="2" stroke-linecap="round" aria-hidden="true">
  <path d="M18 6 6 18M6 6l12 12" /></svg>`;

const tagTemplate = (removeLabel: string) => (tagData: Tagify.TagData) => {
  const value = tagData.value;
  return `<tag title="${value}" contenteditable="false" tabindex="-1"
      class="tagify__tag">
      <x role="button" class="tagify__tag__removeBtn"
        aria-label="${removeLabel} ${value}">${REMOVE_ICON}</x>
      <div><span class="tagify__tag-text">${value}</span></div>
    </tag>`;
};

const TagInput: React.FC<TagInputProps> = ({
  shouldEnforceSuggestions = false,
  Icon,
  label,
  onChange,
  placeholder,
  removeLabel,
  shouldFocus,
  shouldShowFieldIcon,
  suggestions,
  tags,
}) => {
  const inputRef = React.useRef<HTMLInputElement>(null);
  const tagifyRef = React.useRef<Tagify>(null);
  // Read at event time, so a new handler never tears the instance down
  const onChangeRef = React.useRef(onChange);
  onChangeRef.current = onChange;

  React.useEffect(() => {
    const input = inputRef.current!;
    const tagify = new Tagify(input, {
      editTags: false,
      // A repeat of a tag already there is dropped outright, rather than
      // added and then taken back a second later
      skipInvalid: true,
      enforceWhitelist: shouldEnforceSuggestions,
      // Seeded here, so the initial tags pass the whitelist they load against
      whitelist: [...suggestions],
      dropdown: {
        // A dialog renders in the top layer, so a body-level dropdown would
        // come up behind it
        appendTarget: input.closest("dialog") ?? document.body,
        closeOnSelect: true,
        // Suggestions wait for the first character: while the draft is empty
        // an open dropdown would swallow the arrows that walk the caret
        // between the tags
        enabled: MIN_SUGGESTION_CHARACTERS,
        maxItems: MAX_SUGGESTIONS,
        // Opens under what is being typed, not under the field's left edge
        position: "text",
      },
      placeholder,
      templates: { tag: tagTemplate(removeLabel) },
    });
    tagify.on("change", () => onChangeRef.current(valuesOf(tagify)));
    tagify.on("click", () => {
      tagify.DOM.input.focus();
      tagify.setRangeAtStartEnd(false, tagify.DOM.input);
    });
    // Tab completes only from a highlighted suggestion, and Tagify reads that
    // off the window, so highlighting the first one here, at the input, lets
    // Tab complete in every field while Enter still keeps the typed text
    tagify.DOM.input.addEventListener("keydown", (e) => {
      const { classNames } = tagify.settings;
      const list = tagify.DOM.dropdown;
      if (e.key !== "Tab" || !list.isConnected) return;
      if (list.querySelector(`.${classNames.dropdownItemActive}`)) return;
      const first = list.querySelector(`.${classNames.dropdownItem}`);
      const dropdown = tagify.dropdown as typeof tagify.dropdown &
        DropdownInternals;
      if (first) dropdown.highlightOption(first);
    });
    tagifyRef.current = tagify;
    if (shouldFocus) tagify.DOM.input.focus();
    return () => tagify.destroy();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  React.useEffect(() => {
    const tagify = tagifyRef.current;
    // Tagify loads the initial tags itself, off the input's own value, and
    // reloading them here would block the change event its own edits raise
    if (!tagify || valuesOf(tagify).join(",") === tags.join(",")) return;
    tagify.loadOriginalValues(tags);
  }, [tags]);

  React.useEffect(() => {
    if (tagifyRef.current) tagifyRef.current.whitelist = [...suggestions];
  }, [suggestions]);

  return (
    // Unlabelled inputs are rows inside someone else's section, so they skip
    // the section chrome rather than nesting a second bordered box in it
    <div className={label ? styles.section : undefined}>
      {label && (
        <div className={styles.sectionHeader}>
          <label className={styles.label}>
            <span className={styles.labelIcon}>
              <Icon aria-hidden="true" />
            </span>
            {label}
          </label>
        </div>
      )}
      <div
        className={[styles.field, !label && styles.fieldInline]
          .filter(Boolean)
          .join(" ")}
      >
        {!label && shouldShowFieldIcon && (
          <span className={styles.fieldIcon}>
            <Icon aria-hidden="true" />
          </span>
        )}
        <input
          ref={inputRef}
          aria-label={placeholder}
          defaultValue={tags.join(",")}
        />
      </div>
    </div>
  );
};

export default TagInput;
