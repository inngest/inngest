'use server';

import { testAuth } from '@/components/PostgresIntegration/neonData';
import { type CdcConnectionInput } from '@/gql/graphql';

export async function verifyCredentials(input: CdcConnectionInput) {
  try {
    const response = await testAuth(input);

    const isSuccessful = response.error === null;

    return isSuccessful;
  } catch (error) {
    console.error('Error verifying credentials:', error);
    return false;
  }
}
