export const HIDE_TOOLTIPS_EVENT = "hideTooltips";

export const hideTooltips = () => {
  document.dispatchEvent(new CustomEvent(HIDE_TOOLTIPS_EVENT));
};
