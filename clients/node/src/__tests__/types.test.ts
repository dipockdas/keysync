import { describe, it, expect } from "vitest";
import { serviceName, parseServiceName } from "../types.js";

describe("serviceName", () => {
  it("returns keysync/global for global scope", () => {
    expect(serviceName("global")).toBe("keysync/global");
  });

  it("returns keysync/project/<name> for project scope", () => {
    expect(serviceName("project", "my-app")).toBe("keysync/project/my-app");
  });

  it("ignores project parameter for global scope", () => {
    expect(serviceName("global", "my-app")).toBe("keysync/global");
  });
});

describe("parseServiceName", () => {
  it("parses global scope", () => {
    expect(parseServiceName("keysync/global")).toEqual({ scope: "global" });
  });

  it("parses project scope", () => {
    expect(parseServiceName("keysync/project/my-app")).toEqual({
      scope: "project",
      project: "my-app",
    });
  });

  it("parses deep project paths", () => {
    expect(parseServiceName("keysync/project/my/deep/app")).toEqual({
      scope: "project",
      project: "my/deep/app",
    });
  });

  it("returns global for unprefixed input", () => {
    expect(parseServiceName("other/global")).toEqual({ scope: "global" });
  });

  it("returns global for empty input", () => {
    expect(parseServiceName("")).toEqual({ scope: "global" });
  });

  it("handles just keysync prefix", () => {
    expect(parseServiceName("keysync")).toEqual({ scope: "global" });
  });
});
