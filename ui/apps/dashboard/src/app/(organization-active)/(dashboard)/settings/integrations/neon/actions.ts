'use server';

import { deleteConn } from '@/components/PostgresIntegration/neonData';

export async function deleteConnection(id: string) {
  try {
    await deleteConn(id);

    return { success: true, error: null };
  } catch (error) {
    console.error('Error deleting cdc connection:', error);
    return { success: false, error: 'Error removing Neon integration, please try again later.' };
  }
}
