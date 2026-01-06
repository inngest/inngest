import { getProductionEnvironment } from '@/queries/server/getEnvironment';
import {
  getInvokeFunctionLookups,
  invokeFn,
  preloadInvokeFunctionLookups,
} from '@/components/Onboarding/data';
import { createServerFn } from '@tanstack/react-start';

export const invokeFunction = createServerFn({ method: 'POST' })
  .inputValidator(
    (data: {
      functionSlug: string;
      user: Record<string, unknown> | null;
      data: Record<string, unknown>;
    }) => data,
  )
  .handler(async ({ data }) => {
    try {
      await invokeFn({
        functionSlug: data.functionSlug,
        user: data.user,
        data: data.data,
      });

      return {
        success: true,
      };
    } catch (error) {
      console.error('Error invoking function:', error);

      if (error instanceof Error) {
        return {
          success: false,
          error: error.message,
        };
      }

      return {
        success: false,
        error: 'Unknown error occurred while invoking function',
      };
    }
  });

export const prefetchFunctions = createServerFn({ method: 'GET' }).handler(
  async () => {
    const environment = await getProductionEnvironment();

    preloadInvokeFunctionLookups(environment.slug);
    const {
      envBySlug: {
        workflows: { data: functions },
      },
    } = await getInvokeFunctionLookups(environment.slug);

    return functions;
  },
);
