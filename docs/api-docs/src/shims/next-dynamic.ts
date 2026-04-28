/**
 * Shim for `next/dynamic` used by fumadocs-openapi/ui.
 * Maps to React.lazy so the package works outside of Next.js.
 */
import { lazy, type ComponentType } from 'react';

type DynamicOptions = {
  ssr?: boolean;
  loading?: ComponentType;
};

export default function dynamic<P = Record<string, never>>(
  fn: () => Promise<{ default: ComponentType<P> }>,
  _opts?: DynamicOptions
): ComponentType<P> {
  return lazy(fn);
}
