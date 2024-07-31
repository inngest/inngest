import { getEnvironment } from '@/queries/server-only/getEnvironment';
import { type Environment } from '@/utils/environments';

export const getEnv = async (slug: string): Promise<Environment | undefined> => {
  try {
    return await getEnvironment({ environmentSlug: slug });
  } catch (e: any) {
    console.error('error fetching env', e);

    if (e.message && e.message.includes('no rows')) {
      return undefined;
    }

    throw e;
  }
};
