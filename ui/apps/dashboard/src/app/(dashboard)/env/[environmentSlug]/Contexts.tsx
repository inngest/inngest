'use client';

import { EnvContext } from '@/contexts/env';
import { useEnvironment } from '@/queries';

type Props = React.PropsWithChildren<{
  envSlug: string;
}>;

export function Contexts({ envSlug, children }: Props) {
  const [{ data: environment, error, fetching }] = useEnvironment({
    environmentSlug: envSlug,
  });
  if (error) {
    throw error;
  }
  if (fetching) {
    // TODO: Add loading state
    return null;
  }
  if (!environment) {
    throw new Error('failed to fetch environment');
  }

  return <EnvContext.Provider value={{ id: environment.id }}>{children}</EnvContext.Provider>;
}
