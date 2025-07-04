import { useMemo } from 'react';
import type { PathCreator } from '@inngest/components/SharedContext/usePathCreator';

import { useEnvironment } from '@/components/Environments/environment-context';
import { pathCreator as internalPathCreator } from '@/utils/urls';

export const usePathCreator = () => {
  const env = useEnvironment();

  const pathCreator = useMemo((): PathCreator => {
    return {
      app: (params: { externalAppID: string }) =>
        internalPathCreator.app({ envSlug: env.slug, externalAppID: params.externalAppID }),
      eventPopout: ({ eventID }: { eventID: string }) =>
        internalPathCreator.eventPopout({ envSlug: env.slug, eventID }),
      eventType: ({ eventName }: { eventName: string }) =>
        internalPathCreator.eventType({ envSlug: env.slug, eventName }),
      function: (params: { functionSlug: string }) =>
        internalPathCreator.function({ envSlug: env.slug, functionSlug: params.functionSlug }),
      runPopout: (params: { runID: string }) =>
        internalPathCreator.runPopout({ envSlug: env.slug, runID: params.runID }),
      debugger: (params: { functionSlug: string; runID?: string }) =>
        internalPathCreator.debugger({
          envSlug: env.slug,
          functionSlug: params.functionSlug,
          runID: params.runID,
        }),
    };
  }, [env.slug]);

  return pathCreator;
};
