import { renderHook, waitFor } from "@testing-library/react";
import { api } from "@/app/api";
import { useAgentCapabilities } from "./useAgentCapabilities";

jest.mock("@/app/api", () => ({
  api: { listAgentCapabilities: jest.fn() },
}));

describe("useAgentCapabilities", () => {
  test("keeps partial results and discovery problems", async () => {
    const capabilities = {
      providers: [{ provider: "openai", models: [], mcp: [] }],
      problems: [
        {
          instance: "provider/bedrock",
          status: 403,
          detail: "bad credentials",
        },
      ],
    };
    jest.mocked(api.listAgentCapabilities).mockResolvedValue(capabilities);
    const { result } = renderHook(useAgentCapabilities);
    await waitFor(() => expect(result.current).toEqual(capabilities));
  });

  test("exposes request failures", async () => {
    jest
      .mocked(api.listAgentCapabilities)
      .mockRejectedValue(new Error("gateway unavailable"));
    const { result } = renderHook(useAgentCapabilities);
    await waitFor(() =>
      expect(result.current.problems?.[0].detail).toContain(
        "gateway unavailable"
      )
    );
    expect(result.current.providers).toEqual([]);
  });

  test("ignores a response after unmounting", async () => {
    let resolve!: (value: { providers: [] }) => void;
    jest.mocked(api.listAgentCapabilities).mockReturnValue(
      new Promise((done) => {
        resolve = done;
      })
    );
    const { result, unmount } = renderHook(useAgentCapabilities);
    const signal = jest
      .mocked(api.listAgentCapabilities)
      .mock.calls.at(-1)?.[0];
    unmount();
    expect(signal?.aborted).toBe(true);
    resolve({ providers: [] });
    await Promise.resolve();
    expect(result.current).toEqual({ providers: [] });
  });
});
