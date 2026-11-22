import { META_KEYS } from "@/app/api";
import type { InlineSelectOption } from "@/app/components/molecules/InlineSelectDropdown";
import { useT } from "@/app/i18n";
import { useFeeds } from "@/app/store/flowStore";

export const useAttributeValueControls = () => {
  const t = useT();
  const feeds = useFeeds();
  const feedOptions = feeds.map((f) => ({ value: f.id, label: f.name }));
  const metaKeyOptions: InlineSelectOption[] = META_KEYS.map((key) => ({
    value: key,
    label: t(`stepEditor.metaKey.${key}`),
  }));
  return { feedOptions, metaKeyOptions };
};
