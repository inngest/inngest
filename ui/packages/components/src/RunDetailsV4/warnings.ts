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

/**
 * Collects warnings from every `inngest.warnings` entry, sorted by key.
 * Entries merge oldest `updatedAt` first so newer values win on duplicate
 * keys. Non-string and blank values are skipped; messages are trimmed.
 */
export function getStepWarnings(metadata?: SpanMetadata[]): StepWarning[] {
  const byKey = new Map<string, string>();

  const entries = (metadata ?? [])
    .filter((md) => {
      return md.kind === 'inngest.warnings';
    })
    .map((md, index) => {
      return { md, index, time: Date.parse(md.updatedAt ?? '') };
    })
    .sort((a, b) => {
      // Empty or unparseable timestamps sort first, keeping input order among ties
      const aTime = Number.isNaN(a.time) ? -Infinity : a.time;
      const bTime = Number.isNaN(b.time) ? -Infinity : b.time;
      if (aTime === bTime) {
        return a.index - b.index;
      }

      return aTime < bTime ? -1 : 1;
    });

  for (const { md } of entries) {
    for (const [key, value] of Object.entries(md.values ?? {})) {
      if (typeof value !== 'string') {
        continue;
      }

      const message = value.trim();
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
