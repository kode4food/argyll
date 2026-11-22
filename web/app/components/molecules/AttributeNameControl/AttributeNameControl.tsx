import React from "react";
import { InputCollect } from "@/app/api";
import CollectIcon from "@/app/components/atoms/CollectIcon";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
import InlineSelectDropdown from "@/app/components/molecules/InlineSelectDropdown";
import { useT } from "@/app/i18n";
import { IconArrayMultiple, IconArraySingle } from "@/utils/iconRegistry";
import styles from "./AttributeNameControl.module.css";
import { useAttributeNameControl } from "./useAttributeNameControl";

export interface AttributeNameControlProps {
  name: string;
  className: string;
  canCollect: boolean;
  collect?: InputCollect;
  forEach?: boolean;
  onNameChange: (name: string) => void;
  onCollectChange: (collect: string) => void;
  onForEachChange: (forEach: boolean) => void;
}

const AttributeNameControl: React.FC<AttributeNameControlProps> = ({
  name,
  className,
  canCollect,
  collect: rawCollect,
  forEach,
  onNameChange,
  onCollectChange,
  onForEachChange,
}) => {
  const t = useT();
  const { collect, collectOptions, shouldShowForEach } =
    useAttributeNameControl(rawCollect, canCollect);

  return canCollect ? (
    <SegmentedGroup className={className}>
      <input
        type="text"
        value={name}
        onChange={(e) => onNameChange(e.target.value)}
        placeholder={t("stepEditor.attributeNamePlaceholder")}
        className={styles.nameInputInline}
      />
      {shouldShowForEach && (
        <InlineSelectDropdown
          ariaLabel={t("stepEditor.forEachSelect")}
          faceIcon={
            forEach ? (
              <IconArrayMultiple className={styles.icon} />
            ) : (
              <IconArraySingle className={styles.icon} />
            )
          }
          onChange={(v) => onForEachChange(v === "multi")}
          options={[
            {
              value: "single",
              label: t("stepEditor.arraySingleLabel"),
              icon: <IconArraySingle className={styles.icon} />,
            },
            {
              value: "multi",
              label: t("stepEditor.arrayMultiLabel"),
              icon: <IconArrayMultiple className={styles.icon} />,
            },
          ]}
          value={forEach ? "multi" : "single"}
        />
      )}
      <InlineSelectDropdown
        ariaLabel={t("stepEditor.collectSelect")}
        faceIcon={
          <CollectIcon collect={collect} className={styles.collectIcon} />
        }
        onChange={onCollectChange}
        options={collectOptions}
        value={collect}
      />
    </SegmentedGroup>
  ) : (
    <input
      type="text"
      value={name}
      onChange={(e) => onNameChange(e.target.value)}
      placeholder={t("stepEditor.attributeNamePlaceholder")}
      className={className}
    />
  );
};

export default AttributeNameControl;
