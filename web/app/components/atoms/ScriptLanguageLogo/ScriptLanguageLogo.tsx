import React from "react";
import { SCRIPT_LANGUAGE_ARGYLL, SCRIPT_LANGUAGE_LUA } from "@/app/api";
import styles from "./ScriptLanguageLogo.module.css";

export interface ScriptLanguageLogoProps {
  language: string;
  name: string;
  showName?: boolean;
}

const LOGO_SOURCES: Record<string, string> = {
  [SCRIPT_LANGUAGE_ARGYLL]: "/icons/argex-logo.svg",
  [SCRIPT_LANGUAGE_LUA]: "/icons/lua-logo.svg",
};

const ScriptLanguageLogo: React.FC<ScriptLanguageLogoProps> = ({
  language,
  name,
  showName = false,
}) => {
  const src = LOGO_SOURCES[language];
  if (!src) {
    return <>{name}</>;
  }
  if (!showName) {
    return <img src={src} alt={name} className={styles.logo} />;
  }
  return (
    <span className={styles.withName}>
      <img src={src} alt="" className={styles.logo} />
      {name}
    </span>
  );
};

export default ScriptLanguageLogo;
