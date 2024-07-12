import getAllEnvironments from '@/queries/server-only/getAllEnvironments';
import { getActiveEnvironment, type Environment } from '@/utils/environments';

export const getEnvs = async (slug: string): Promise<{ env: Environment; envs: Environment[] }> => {
  const envs = await getAllEnvironments();

  const env = getActiveEnvironment(envs, slug);

  if (!env) {
    throw new Error(`Environments not found for slug ${slug}`);
  }

  return { envs, env };
};
