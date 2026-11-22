import { InputCollect } from "@/app/api";
import CollectIcon from "@/app/components/atoms/CollectIcon";
import { useT } from "@/app/i18n";
import type { InlineSelectOption } from "@/app/components/molecules/InlineSelectDropdown";
import styles from "./AttributeNameControl.module.css";

export const INPUT_COLLECT_TYPES: InputCollect[] = [
  "first",
  "last",
  "some",
  "all",
  "none",
];

export const useAttributeNameControl = (
  rawCollect: InputCollect | undefined,
  canCollect: boolean
) => {
  const t = useT();
  const collect =
    rawCollect && INPUT_COLLECT_TYPES.includes(rawCollect)
      ? rawCollect
      : "first";

  const collectOptions: InlineSelectOption[] = INPUT_COLLECT_TYPES.map((c) => ({
    value: c,
    label: t(`stepEditor.collect.${c}`),
    icon: <CollectIcon collect={c} className={styles.collectIcon} />,
  }));

  const shouldShowForEach =
    canCollect && (collect === "all" || collect === "some");

  return { collect, collectOptions, shouldShowForEach };
};
