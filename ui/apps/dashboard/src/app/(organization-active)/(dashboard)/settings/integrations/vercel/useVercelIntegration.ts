'use client';

import { useMemo } from 'react';
import { useQuery } from 'urql';

import { useEnvironments } from '@/queries';
import { getProductionEnvironment } from '@/utils/environments';
import { useRestAPIRequest } from '@/utils/useRestAPIRequest';
import { type VercelIntegration, type VercelProjectAPIResponse } from './VercelIntegration';
import mergeVercelProjectData from './mergeVercelProjectData';
import { GetSavedVercelProjectsDocument } from './queries';

const notEnabledVercelIntegration: VercelIntegration = {
  id: 'not-enabled',
  name: 'Vercel',
  slug: 'vercel',
  projects: [],
  enabled: false,
};

export function useVercelIntegration(): {
  data: VercelIntegration;
  fetching: boolean;
  error: Error | undefined;
} {
  const [{ data: environments, fetching, error: environmentError }] = useEnvironments();

  const productionEnvironmentId = useMemo(() => {
    if (!environments) return null;
    const env = getProductionEnvironment(environments);
    return env?.id;
  }, [environments]);

  // Use memo as the URL object will change on every render
  const url = useMemo(() => {
    if (!productionEnvironmentId) {
      return null;
    }
    const url = new URL('/v1/integrations/vercel/projects', process.env.NEXT_PUBLIC_API_URL);
    url.searchParams.set('workspaceID', productionEnvironmentId);
    return url;
  }, [productionEnvironmentId]);

  // Fetch data from REST and GQL and merge
  const {
    data,
    isLoading: isLoadingSavedProjects,
    error,
  } = useRestAPIRequest<VercelProjectAPIResponse>({ url, method: 'GET' });
  const [{ data: savedVercelProjects }] = useQuery({
    query: GetSavedVercelProjectsDocument,
    variables: {
      environmentID: productionEnvironmentId || '',
    },
    pause: !productionEnvironmentId,
  });

  const projects = mergeVercelProjectData({
    vercelProjects: data?.projects || [],
    savedProjects: savedVercelProjects?.environment.savedVercelProjects || [],
  });

  const vercelIntegration = data
    ? {
        id: 'enabled-integration-id',
        name: 'Vercel',
        slug: 'vercel',
        projects,
        enabled: true,
      }
    : notEnabledVercelIntegration;

  return {
    data: vercelIntegration,
    fetching: fetching || isLoadingSavedProjects,
    error: environmentError || error,
  };
}
