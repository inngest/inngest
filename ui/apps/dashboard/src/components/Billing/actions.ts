'use server';

import { entitlementUsage } from './data';

export async function getEntitlementUsage() {
  try {
    const response = (await entitlementUsage()).account.entitlementUsage;
    return response;
  } catch (error) {
    console.error('Error fetchign entitlements:', error);
    return null;
  }
}
