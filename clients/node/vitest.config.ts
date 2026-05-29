import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    // PowerShell inline C# compilation is slow on Windows VMs,
    // especially under x86-to-ARM translation. 30s gives ample headroom.
    testTimeout: 30_000,
  },
});
