import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import Legend from "./Legend";
import { t } from "@/app/testUtils/i18n";

jest.mock("@/app/i18n", () => ({
  useT: () => t,
}));

describe("Legend", () => {
  test("renders diagram controls in footer when actions are provided", () => {
    const onZoomIn = jest.fn();
    const onZoomOut = jest.fn();
    const onFitView = jest.fn();
    const onToggleTheme = jest.fn();

    render(
      <Legend
        onZoomIn={onZoomIn}
        onZoomOut={onZoomOut}
        onFitView={onFitView}
        onToggleTheme={onToggleTheme}
        theme="light"
      />
    );

    fireEvent.click(screen.getByRole("button", { name: t("legend.zoomOut") }));
    fireEvent.click(screen.getByRole("button", { name: t("legend.zoomIn") }));
    fireEvent.click(screen.getByRole("button", { name: t("legend.autoZoom") }));
    fireEvent.click(
      screen.getByRole("button", { name: t("legend.switchToDarkMode") })
    );

    expect(onZoomOut).toHaveBeenCalled();
    expect(onZoomIn).toHaveBeenCalled();
    expect(onFitView).toHaveBeenCalled();
    expect(onToggleTheme).toHaveBeenCalled();
  });
});
