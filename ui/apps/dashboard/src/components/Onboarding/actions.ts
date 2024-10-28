'use server';

import { type InvokeFunctionMutationVariables } from '@/gql/graphql';
import { invokeFn, syncNewApp } from './data';

export async function syncAppManually(appURL: string) {
  try {
    const response = await syncNewApp(appURL);
    const error = response.syncNewApp.error;
    if (error) {
      return { success: false, error: error, appName: null };
    }
    return {
      success: true,
      error: null,
      appName: response.syncNewApp.app?.externalID || 'Unknown App',
    };
  } catch (error) {
    console.error('Error syncing app:', error);
    return { success: false, error: null, appName: null };
  }
}

export async function invokeFunction({
  functionSlug,
  user,
  data,
}: Pick<InvokeFunctionMutationVariables, 'data' | 'functionSlug' | 'user'>) {
  try {
    const response = await invokeFn({ functionSlug, user, data });
    const isSuccess = response.invokeFn.invokeFunction;

    return {
      success: isSuccess,
    };
  } catch (error) {
    console.error('Error invoking:', error);
    if (!(error instanceof Error)) {
      return { success: false, error: 'Unknown error invoking' };
    }
    return { success: false, error: error };
  }
}
