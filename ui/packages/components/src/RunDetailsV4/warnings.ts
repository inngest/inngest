/**
 * @module
 * Extracts `inngest.warnings` span metadata (a flat map of key to message)
 * into a list the trace UI can render. Library-agnostic: any step carrying the
 * kind gets the same treatment.
 */

import type { SpanMetadata } from './types';

export type StepWarning = {
  key: string;
  message: string;
};

/** Collects warnings from every `inngest.warnings` entry, sorted by key. Later entries win on duplicate keys. */
export function getStepWarnings(metadata?: SpanMetadata[]): StepWarning[] {
  const byKey = new Map<string, string>();

  for (const md of metadata ?? []) {
    if (md.kind !== 'inngest.warnings') {
      continue;
    }

    for (const [key, value] of Object.entries(md.values ?? {})) {
      const message = typeof value === 'string' ? value : String(value ?? '');
      if (message === '') {
        continue;
      }

      byKey.set(key, message);
    }
  }

  return [...byKey.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, message]) => ({ key, message }));
}

/** Short summary for tooltips: the message itself for one warning, otherwise "N warnings". */
export function summarizeWarnings(warnings: StepWarning[]): string {
  if (warnings.length === 1) {
    return warnings[0]!.message;
  }

  return `${warnings.length} warnings`;
}
