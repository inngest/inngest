export const INSIGHTS_AI = "Insights AI" as const;
export const DOCUMENTATION = "Documentation" as const;
export const SCHEMA_EXPLORER = "Schema explorer" as const;
export const SUPPORT = "Support" as const;

export type HelperTitle =
  | typeof INSIGHTS_AI
  | typeof DOCUMENTATION
  | typeof SCHEMA_EXPLORER
  | typeof SUPPORT;
