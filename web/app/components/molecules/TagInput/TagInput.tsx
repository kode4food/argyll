import React from "react";
import useClickOutside from "@/app/hooks/useClickOutside";
import { createPortal } from "react-dom";
import { IconClose, type LucideIcon } from "@/utils/iconRegistry";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import styles from "./TagInput.module.css";

const MAX_SUGGESTIONS = 8;

export interface TagInputProps {
  Icon: LucideIcon;
  // Omitted when the caller heads a group of inputs with a label of its own
  label?: string;
  onChange: (tags: string[]) => void;
  placeholder: string;
  removeLabel: string;
  suggestions: readonly string[];
  tags: string[];
}

interface KeyContext {
  available: string[];
  commit: (value: string) => void;
  draft: string;
  event: React.KeyboardEvent;
  highlightedIndex: number;
  isOpen: boolean;
  onChange: (tags: string[]) => void;
  setHighlightedIndex: React.Dispatch<React.SetStateAction<number>>;
  setOpen: (open: boolean) => void;
  tags: string[];
}

const moveHighlight = (ctx: KeyContext, step: number) => {
  ctx.event.preventDefault();
  ctx.setHighlightedIndex((current) => {
    const next = current + step;
    if (next < 0) return ctx.available.length - 1;
    return next >= ctx.available.length ? 0 : next;
  });
};

const commitHighlighted = (ctx: KeyContext) => {
  ctx.event.preventDefault();
  ctx.commit(ctx.available[ctx.highlightedIndex] ?? ctx.draft);
};

const openOrMoveDown = (ctx: KeyContext) => {
  if (!ctx.isOpen && ctx.available.length > 0) {
    ctx.event.preventDefault();
    ctx.setOpen(true);
    ctx.setHighlightedIndex(0);
    return;
  }
  moveHighlight(ctx, 1);
};

const dropLastTag = (ctx: KeyContext) => {
  if (ctx.draft === "" && ctx.tags.length > 0) {
    ctx.onChange(ctx.tags.slice(0, -1));
  }
};

const KEY_HANDLERS: Record<string, (ctx: KeyContext) => void> = {
  ",": commitHighlighted,
  ArrowDown: openOrMoveDown,
  ArrowUp: (ctx) => moveHighlight(ctx, -1),
  Backspace: dropLastTag,
  Enter: commitHighlighted,
  Escape: (ctx) => ctx.setOpen(false),
};

const TagInput: React.FC<TagInputProps> = ({
  Icon,
  label,
  onChange,
  placeholder,
  removeLabel,
  suggestions,
  tags,
}) => {
  const [draft, setDraft] = React.useState("");
  const [isOpen, setOpen] = React.useState(false);
  // Stays -1 until the user arrows onto a suggestion, so typing and pressing
  // Enter always commits what was typed
  const [highlightedIndex, setHighlightedIndex] = React.useState(-1);
  const wrapperRef = React.useRef<HTMLDivElement>(null);
  const inputRef = React.useRef<HTMLInputElement>(null);
  const available = suggestions
    .filter((tag) => !tags.includes(tag) && tag.includes(draft.trim()))
    .slice(0, MAX_SUGGESTIONS);

  useClickOutside(wrapperRef, () => setOpen(false), isOpen);

  const commit = (value: string) => {
    const tag = value.trim();
    if (tag && !tags.includes(tag)) onChange([...tags, tag]);
    setDraft("");
    setHighlightedIndex(-1);
    setOpen(false);
  };

  // The field lives inside scrolling panels, so the list is portalled out and
  // aligned under the input rather than clipped by them. A dialog renders
  // in the top layer, so the list has to be portalled into it to stay on top
  const [listStyle, setListStyle] = React.useState<React.CSSProperties>();
  const listHost = wrapperRef.current?.closest("dialog") ?? document.body;

  React.useLayoutEffect(() => {
    if (!isOpen) return;
    const position = () => {
      const field = wrapperRef.current?.getBoundingClientRect();
      const input = inputRef.current?.getBoundingClientRect();
      if (field && input) {
        setListStyle({ top: field.bottom + 4, left: input.left });
      }
    };
    position();
    window.addEventListener("resize", position);
    window.addEventListener("scroll", position, true);
    return () => {
      window.removeEventListener("resize", position);
      window.removeEventListener("scroll", position, true);
    };
  }, [isOpen]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    KEY_HANDLERS[e.key]?.({
      available,
      commit,
      draft,
      event: e,
      highlightedIndex,
      isOpen,
      onChange,
      setHighlightedIndex,
      setOpen,
      tags,
    });
  };

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
        ref={wrapperRef}
        className={styles.field}
        onClick={() => inputRef.current?.focus()}
      >
        {tags.map((tag) => (
          <span key={tag} className={styles.tag}>
            <span className={styles.tagText}>{tag}</span>
            <button
              type="button"
              tabIndex={-1}
              onClick={() => onChange(tags.filter((t) => t !== tag))}
              className={styles.removeButton}
              aria-label={`${removeLabel} ${tag}`}
            >
              <IconClose className={styles.iconSm} />
            </button>
          </span>
        ))}
        <input
          ref={inputRef}
          type="text"
          className={styles.input}
          value={draft}
          onChange={(e) => {
            setDraft(e.target.value);
            setHighlightedIndex(-1);
            setOpen(true);
          }}
          onKeyDown={handleKeyDown}
          onBlur={() => commit(draft)}
          placeholder={tags.length === 0 ? placeholder : ""}
          aria-label={placeholder}
          aria-autocomplete="list"
          aria-expanded={isOpen}
        />
      </div>
      {isOpen &&
        available.length > 0 &&
        createPortal(
          <div
            className={dropdownStyles.list}
            style={{ position: "fixed", ...listStyle }}
            role="listbox"
            data-ui-overlay="dropdown"
          >
            {available.map((suggestion, index) => (
              <button
                key={suggestion}
                type="button"
                role="option"
                aria-selected={index === highlightedIndex}
                className={[
                  dropdownStyles.item,
                  index === highlightedIndex && dropdownStyles.itemHighlighted,
                ]
                  .filter(Boolean)
                  .join(" ")}
                onMouseEnter={() => setHighlightedIndex(index)}
                // Committing on mousedown, since the click-outside handler
                // unmounts the portalled list before a click can land
                onMouseDown={(e) => {
                  e.preventDefault();
                  commit(suggestion);
                }}
              >
                {suggestion}
              </button>
            ))}
          </div>,
          listHost
        )}
    </div>
  );
};

export default TagInput;
