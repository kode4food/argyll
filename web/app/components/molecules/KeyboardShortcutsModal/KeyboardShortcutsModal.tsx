import React from "react";
import { IconClose, IconCommandKey } from "@/utils/iconRegistry";
import { useEscapeKey } from "@/app/hooks/useEscapeKey";
import styles from "./KeyboardShortcutsModal.module.css";
import { useT } from "@/app/i18n";

interface KeyboardShortcut {
  keys: string[];
  description: string;
  section?: string;
}

interface KeyboardShortcutsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const KeyboardShortcutsModal: React.FC<KeyboardShortcutsModalProps> = ({
  isOpen,
  onClose,
}) => {
  const t = useT();
  const generalSection = t("keyboardShortcuts.sectionGeneral");
  const navigationSection = t("keyboardShortcuts.sectionNavigation");
  const shortcuts: KeyboardShortcut[] = [
    {
      keys: ["?"],
      description: t("keyboardShortcuts.showShortcuts"),
      section: generalSection,
    },
    {
      keys: ["/"],
      description: t("keyboardShortcuts.focusSearch"),
      section: generalSection,
    },
    {
      keys: ["Esc"],
      description: t("keyboardShortcuts.closeModals"),
      section: generalSection,
    },
    {
      keys: ["↑", "↓"],
      description: t("keyboardShortcuts.navigateWithinLevel"),
      section: navigationSection,
    },
    {
      keys: ["←", "→"],
      description: t("keyboardShortcuts.navigateBetweenLevels"),
      section: navigationSection,
    },
    {
      keys: ["Enter"],
      description: t("keyboardShortcuts.openStepEditor"),
      section: navigationSection,
    },
  ];

  useEscapeKey(isOpen, onClose);

  if (!isOpen) return null;

  const sections = Array.from(new Set(shortcuts.map((s) => s.section)));

  return (
    <>
      <div className={styles.modal}>
        <div className={styles.header}>
          <div className={styles.headerContent}>
            <IconCommandKey className={styles.headerIcon} />
            <h2 className={styles.title}>{t("keyboardShortcuts.title")}</h2>
          </div>
          <button
            onClick={onClose}
            className={styles.closeButton}
            aria-label={t("keyboardShortcuts.close")}
          >
            <IconClose className={styles.headerIcon} />
          </button>
        </div>

        <div className={styles.body}>
          {sections.map((section) => (
            <div key={section} className={styles.section}>
              <h3 className={styles.sectionTitle}>{section}</h3>
              <div className={styles.shortcutsList}>
                {shortcuts
                  .filter((s) => s.section === section)
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
