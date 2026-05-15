import type { Trace } from '../RunDetailsV3/types';

//
// A set of helpers to constuct a debug run trace from a set of related debug runs
// that may contains partials because our run from step traces do not include prior steps.
// Currently it simply chooses the latest step trace, but it would probably be better to
// send them all back as well so the UI can show step over step comps.

export const findSpan = (span: Trace, targetStepID: string): Trace | undefined => {
  if (span.stepID === targetStepID) {
    return span;
  }
  return span.childrenSpans?.reduce<Trace | undefined>(
    (found, child) => found || findSpan(child, targetStepID),
    undefined
  );
};

export const latestDebugSpan = (originalSpan: Trace, debugRuns: Trace[]): Trace | undefined => {
  const stepID = originalSpan.stepID;
  if (!stepID) {
    return undefined;
  }
  return [...debugRuns]
    .reverse()
    .reduce<Trace | undefined>((latest, run) => latest || findSpan(run, stepID), undefined);
};

export const overlayDebugRuns = (original: Trace, debugRuns: Trace[]): Trace => ({
  ...original,
  childrenSpans: original.childrenSpans?.map((child) => {
    const latest = latestDebugSpan(child, debugRuns);
    return latest ? overlayDebugRuns(latest, debugRuns) : overlayDebugRuns(child, debugRuns);
  }),
});
