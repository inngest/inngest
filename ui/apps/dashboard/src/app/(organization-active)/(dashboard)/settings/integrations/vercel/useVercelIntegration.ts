'use client';

import { useMemo } from 'react';
import { useQuery } from 'urql';

import { useDefaultEnvironment } from '@/queries';
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
  const [{ data: defaultEnv, fetching: isLoadingEnvironments, error: environmentError }] =
    useDefaultEnvironment();

  // Use memo as the URL object will change on every render
  const url = useMemo(() => {
    if (!defaultEnv?.id) {
      return null;
    }
    const url = new URL('/v1/integrations/vercel/projects', process.env.NEXT_PUBLIC_API_URL);
    url.searchParams.set('workspaceID', defaultEnv.id);
    return url;
  }, [defaultEnv?.id]);

  // Fetch data from REST and GQL and merge
  const {
    data,
    isLoading: isLoadingAllProjects,
    error,
  } = useRestAPIRequest<VercelProjectAPIResponse>({ url, method: 'GET', pause: !url });
  const [{ data: savedVercelProjects, fetching: isLoadingSavedProjects }] = useQuery({
    query: GetSavedVercelProjectsDocument,
    variables: {
      environmentID: defaultEnv?.id || '',
    },
    pause: !defaultEnv?.id,
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
    fetching: isLoadingEnvironments || isLoadingAllProjects || isLoadingSavedProjects,
    error: environmentError || error,
  };
}
