'use server';

import { testAuth, testLogicalReplication } from '@/components/PostgresIntegration/neonData';
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

export async function verifyLogicalReplication(input: CdcConnectionInput) {
  try {
    const response = await testLogicalReplication(input);

    const isSuccessful = response.error === null;

    return isSuccessful;
  } catch (error) {
    console.error('Error verifying logical replication:', error);
    return false;
  }
}
