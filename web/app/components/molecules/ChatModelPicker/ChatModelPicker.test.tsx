import { fireEvent, render, screen } from "@testing-library/react";
import ChatModelPicker from "./ChatModelPicker";

describe("ChatModelPicker", () => {
  test("selects a provider and model", () => {
    const onProviderChange = jest.fn();
    const onModelChange = jest.fn();
    render(
      <ChatModelPicker
        providers={["anthropic", "openai"]}
        models={["claude"]}
        provider="anthropic"
        model=""
        providerLabel="Provider"
        modelLabel="Model"
        onProviderChange={onProviderChange}
        onModelChange={onModelChange}
      />
    );

    fireEvent.click(screen.getByLabelText("Provider"));
    fireEvent.click(screen.getByText("openai"));
    expect(onProviderChange).toHaveBeenCalledWith("openai");

    fireEvent.click(screen.getByLabelText("Model"));
    fireEvent.click(screen.getByText("claude"));
    expect(onModelChange).toHaveBeenCalledWith("claude");
  });
});
