'use server';

import { syncNewApp } from './data';

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
