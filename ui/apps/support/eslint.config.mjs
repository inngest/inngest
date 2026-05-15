import { tanstackConfig } from "@tanstack/eslint-config";

export default [
  {
    ignores: [".output/**", "dist/**", "node_modules/**"],
  },
  ...tanstackConfig,
];
