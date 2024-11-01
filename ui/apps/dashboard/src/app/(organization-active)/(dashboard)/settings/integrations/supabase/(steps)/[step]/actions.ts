'use server';

import { testAuth, testAutoSetup } from '@/components/PostgresIntegration/neonData';
import { type CdcConnectionInput } from '@/gql/graphql';

export async function verifyCredentials(input: CdcConnectionInput) {
  try {
    const response = await testAuth(input);
    const error = response.cdcTestCredentials.error;
    if (error) {
      return { success: false, error: error };
    }
    return { success: true, error: null };
  } catch (error) {
    console.error('Error verifying credentials:', error);
    return { success: false, error: null };
  }
}

export async function verifyAutoSetup(input: CdcConnectionInput) {
  try {
    const response = await testAutoSetup(input);
    const error = response.cdcAutoSetup.error;
    if (error) {
      return { success: false, error: error, steps: response.cdcAutoSetup.steps };
    }
    return { success: true, error: null, steps: response.cdcAutoSetup.steps };
  } catch (error) {
    console.error('Error connecting:', error);
    return { success: false, error: null };
  }
}
