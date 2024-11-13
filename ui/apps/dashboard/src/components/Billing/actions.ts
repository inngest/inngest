'use server';

import { currentPlan, entitlementUsage } from './data';

export async function getEntitlementUsage() {
  try {
    const response = (await entitlementUsage()).account.entitlementUsage;
    return response;
  } catch (error) {
    console.error('Error fetching entitlements:', error);
    return null;
  }
}

export async function getCurrentPlan() {
  try {
    const response = (await currentPlan()).account;
    return response;
  } catch (error) {
    console.error('Error fetching plan:', error);
    return null;
  }
}
