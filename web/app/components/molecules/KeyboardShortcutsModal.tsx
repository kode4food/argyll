import React from "react";
import { X, Command } from "lucide-react";
import { useEscapeKey } from "../../hooks/useEscapeKey";
import styles from "./KeyboardShortcutsModal.module.css";

interface KeyboardShortcut {
  keys: string[];
  description: string;
  section?: string;
}

interface KeyboardShortcutsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const shortcuts: KeyboardShortcut[] = [
  { keys: ["?"], description: "Show keyboard shortcuts", section: "General" },
  { keys: ["/"], description: "Focus search", section: "General" },
  {
    keys: ["Esc"],
    description: "Close modals / Deselect step",
    section: "General",
  },

  {
    keys: ["↑", "↓"],
    description: "Navigate within dependency level",
    section: "Navigation",
  },
  {
    keys: ["←", "→"],
    description: "Navigate between dependency levels",
    section: "Navigation",
  },
  {
    keys: ["Enter"],
    description: "Open step editor (script steps)",
    section: "Navigation",
  },
];

const KeyboardShortcutsModal: React.FC<KeyboardShortcutsModalProps> = ({
  isOpen,
  onClose,
}) => {
  useEscapeKey(isOpen, onClose);

  if (!isOpen) return null;

  const sections = Array.from(
    new Set(shortcuts.map((s) => s.section || "General"))
  );

  return (
    <>
      <div className={styles.modal}>
        <div className={styles.header}>
          <div className={styles.headerContent}>
            <Command className="h-5 w-5" />
            <h2 className={styles.title}>Keyboard Shortcuts</h2>
          </div>
          <button
            onClick={onClose}
            className={styles.closeButton}
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className={styles.body}>
          {sections.map((section) => (
            <div key={section} className={styles.section}>
              <h3 className={styles.sectionTitle}>{section}</h3>
              <div className={styles.shortcutsList}>
                {shortcuts
                  .filter((s) => (s.section || "General") === section)
                  .map((shortcut, index) => (
                    <div key={index} className={styles.shortcut}>
                      <span className={styles.shortcutDescription}>
                        {shortcut.description}
                      </span>
                      <div className={styles.shortcutKeys}>
                        {shortcut.keys.map((key, i) => (
                          <kbd key={i} className={styles.key}>
                            {key}
                          </kbd>
                        ))}
                      </div>
                    </div>
                  ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </>
  );
};

export default KeyboardShortcutsModal;
