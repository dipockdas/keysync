import { describe, it, expect, vi, beforeEach } from "vitest";
import { KeySyncError } from "../index.js";

describe("KeySyncError", () => {
  it("has correct name and code for notFound", () => {
    const err = new KeySyncError("notFound", "secret not found");
    expect(err.name).toBe("KeySyncError");
    expect(err.code).toBe("notFound");
    expect(err.message).toBe("secret not found");
  });

  it("has correct code for keychainError", () => {
    const err = new KeySyncError("keychainError", "something broke");
    expect(err.code).toBe("keychainError");
  });

  it("has correct code for unsupportedPlatform", () => {
    const err = new KeySyncError("unsupportedPlatform", "not supported");
    expect(err.code).toBe("unsupportedPlatform");
  });
});
