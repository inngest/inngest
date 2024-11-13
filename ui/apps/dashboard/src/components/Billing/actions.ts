'use server';

import { billingDetails, currentPlan, entitlementUsage } from './data';

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

export async function getBillingDetails() {
  try {
    const response = (await billingDetails()).account;
    return response;
  } catch (error) {
    console.error('Error fetching billing details:', error);
    return null;
  }
}
