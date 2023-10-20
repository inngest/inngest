import getAllEnvironments from '@/queries/server-only/getAllEnvironments';
import { type Environment } from '@/utils/environments';

type GetEnvironmentParams = {
  environmentSlug: string;
};

export async function getEnvironment({
  environmentSlug,
}: GetEnvironmentParams): Promise<Environment> {
  const environments = await getAllEnvironments();
  const environment = environments.find((e) => e.slug === environmentSlug);
  if (!environment) {
    throw new Error(`Environment ${environmentSlug} not found`);
  }
  return environment;
}
