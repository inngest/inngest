import { pathCreator } from '@/utils/urls';

export function createInternalPathCreator(envSlug: string) {
  // The shared component library is environment-agnostic, so it needs a way to
  // generate URLs without knowing about environments
  return {
    eventType: ({ eventName }: { eventName: string }) =>
      pathCreator.eventType({ envSlug, eventName }),
    runPopout: ({ runID }: { runID: string }) => pathCreator.runPopout({ envSlug, runID }),
    eventPopout: ({ eventID }: { eventID: string }) =>
      pathCreator.eventPopout({ envSlug, eventID }),
  };
}
