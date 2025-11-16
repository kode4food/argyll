import { isValidTimestamp } from "./dates";

describe("isValidTimestamp", () => {
  it("returns false for undefined", () => {
    expect(isValidTimestamp(undefined)).toBe(false);
  });

  it("returns false for null", () => {
    expect(isValidTimestamp(null)).toBe(false);
  });

  it("returns false for empty string", () => {
    expect(isValidTimestamp("")).toBe(false);
  });

  it("returns false for Go zero value (year 1)", () => {
    expect(isValidTimestamp("0001-01-01T00:00:00Z")).toBe(false);
  });

  it("returns true for valid timestamp", () => {
    expect(isValidTimestamp("2024-01-01T00:00:00Z")).toBe(true);
  });

  it("returns true for current date", () => {
    expect(isValidTimestamp(new Date().toISOString())).toBe(true);
  });

  it("returns true for year 2", () => {
    expect(isValidTimestamp("0002-01-01T00:00:00Z")).toBe(true);
  });
});
