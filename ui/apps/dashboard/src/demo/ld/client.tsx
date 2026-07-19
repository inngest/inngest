/**
 * Fake `launchdarkly-react-client-sdk` for the demo build (wired via
 * resolve.alias in vite.demo.config.ts). Avoids any LaunchDarkly network
 * traffic: the provider is a passthrough and no flags are supplied, so every
 * `useBooleanFlag(flag, default)` call falls back to its coded default (see
 * src/components/FeatureFlags/hooks.ts). Only the three imported symbols are
 * provided.
 */
import type { ComponentType } from 'react';

export const withLDProvider =
  (_config: unknown) =>
  <P extends object>(Component: ComponentType<P>): ComponentType<P> =>
    Component;

export const useLDClient = (): undefined => undefined;

export const useFlags = (): Record<string, unknown> => ({});
