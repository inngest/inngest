// Register browser-consumed flags here. Before merging a new entry, enable
// "SDKs using client-side ID" for the flag in LaunchDarkly.
export const clientFeatureFlags = [] as const;

export type ClientFeatureFlagKey = (typeof clientFeatureFlags)[number];
