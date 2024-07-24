import { getEnvironment } from '@/queries/server-only/getEnvironment';
import { type Environment } from '@/utils/environments';

export const getEnv = async (slug: string): Promise<Environment> => {
  const env = await getEnvironment({ environmentSlug: slug });

  return env;
};
