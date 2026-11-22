import React from "react";
import type { Handling } from "@/app/api";
import type { InlineSelectOption } from "@/app/components/molecules/InlineSelectDropdown";
import { useT } from "@/app/i18n";
import { getArgIcon } from "@/utils/iconRegistry";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import type { Attribute } from "./stepEditorUtils";
import { ATTRIBUTE_ROLE_TYPES } from "./stepEditorConstants";

const NO_MAPPING_ROLES = new Set<Attribute["role"]>([
  "const",
  "meta",
  "feed",
  "metric",
]);

export interface AttributeRowOptions {
  attr: Attribute;
  handling: Handling;
  compensatedInnerNames: Set<string>;
}

export const useAttributeRow = ({
  attr,
  handling,
  compensatedInnerNames,
}: AttributeRowOptions) => {
  const t = useT();
  const [isMappingExpanded, setIsMappingExpanded] = React.useState(false);
  const toggleMappingExpanded = () =>
    setIsMappingExpanded((current) => !current);

  const hasMapping = !NO_MAPPING_ROLES.has(attr.role);
  const hasMappingConfigured = Boolean(
    attr.mappingName?.trim() || attr.mappingScript?.trim()
  );
  const isCompensable = handling === "compensated" && attr.role !== "metric";
  const innerName = attr.mappingName?.trim() || attr.name.trim();
  const hasCompensationConflict =
    !attr.compensated && compensatedInnerNames.has(innerName);

  const roleOptions: InlineSelectOption[] = ATTRIBUTE_ROLE_TYPES.map((type) => {
    const { Icon, className } = getArgIcon(type);
    return {
      value: type,
      label: t(`stepEditor.attrRole.${type}`),
      icon: <Icon className={`${styles.iconSm} ${className}`} />,
    };
  });

  const { Icon: RoleIcon, className: roleClassName } = getArgIcon(attr.role);

  const canCollect = attr.role === "required" || attr.role === "optional";
  const hasValueControl = attr.role !== "output";
  const nameClass = hasValueControl
    ? formStyles.argNameInputFixed
    : formStyles.argNameInput;
  const nameControlClass = canCollect
    ? nameClass
    : `${formStyles.argInput} ${nameClass}`;

  return {
    isMappingExpanded,
    toggleMappingExpanded,
    hasMapping,
    hasMappingConfigured,
    isCompensable,
    innerName,
    hasCompensationConflict,
    roleOptions,
    RoleIcon,
    roleClassName,
    canCollect,
    nameClass,
    nameControlClass,
  };
};
