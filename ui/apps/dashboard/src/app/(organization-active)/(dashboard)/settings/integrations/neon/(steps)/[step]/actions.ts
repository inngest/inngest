'use server';

import {
  testAuth,
  testAutoSetup,
  testLogicalReplication,
} from '@/components/PostgresIntegration/neonData';
import { type CdcConnectionInput } from '@/gql/graphql';

export async function verifyCredentials(input: CdcConnectionInput) {
  try {
    const response = await testAuth(input.input);

    const isSuccessful = response.cdcTestCredentials.error === null;

    return isSuccessful;
  } catch (error) {
    console.error('Error verifying credentials:', error);
    return false;
  }
}

export async function verifyLogicalReplication(input: CdcConnectionInput) {
  try {
    const response = await testLogicalReplication(input.input);

    const isSuccessful = response.cdcTestLogicalReplication.error === null;

    return isSuccessful;
  } catch (error) {
    console.error('Error verifying logical replication:', error);
    return false;
  }
}

export async function verifyAutoSetup(input: CdcConnectionInput) {
  try {
    const response = await testAutoSetup(input.input);

    const isSuccessful = response.cdcAutoSetup.error === null;

    return isSuccessful;
  } catch (error) {
    console.error('Error connecting:', error);
    return false;
  }
}
