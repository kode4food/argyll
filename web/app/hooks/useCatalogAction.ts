import React from "react";
import { useT } from "@/app/i18n";

export const useCatalogAction = (
  setError: React.Dispatch<React.SetStateAction<string | null>>
) => {
  const t = useT();
  const [isSaving, setSaving] = React.useState(false);

  const run = async (
    action: () => Promise<unknown>,
    failureKey: string
  ): Promise<boolean> => {
    setSaving(true);
    setError(null);
    try {
      await action();
      return true;
    } catch (err: any) {
      setError(err?.message || t(failureKey));
      return false;
    } finally {
      setSaving(false);
    }
  };

  return { isSaving, run };
};
