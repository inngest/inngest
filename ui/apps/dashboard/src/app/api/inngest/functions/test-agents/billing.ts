import { createAgent, createTool, openai } from '@inngest/agent-kit';
import { z } from 'zod';

import type { CustomerSupportState } from './state';

// Mock billing tools
const checkSubscriptionTool = createTool({
  name: 'check_subscription',
  description: 'Check the current subscription status for a customer',
  // Cast to generic ZodType to avoid deep type instantiation issues
  parameters: z.object({
    userId: z.string(),
  }) as unknown as z.ZodType<any>,
  handler: async ({ userId }) => {
    // Mock implementation
    return {
      userId,
      plan: 'Professional',
      status: 'active',
      nextBillingDate: '2024-02-01',
      amount: '$99/month',
    };
  },
});

const processRefundTool = createTool({
  name: 'process_refund',
  description: 'Process a refund request for a customer',
  parameters: z.object({
    userId: z.string(),
    amount: z.number(),
    reason: z.string(),
  }) as unknown as z.ZodType<any>,
  handler: async ({ userId, amount, reason }) => {
    // Mock implementation
    return {
      refundId: `ref_${Date.now()}`,
      userId,
      amount,
      reason,
      status: 'pending_approval',
      message: 'Refund request has been submitted for approval',
    };
  },
});

const getInvoiceHistoryTool = createTool({
  name: 'get_invoice_history',
  description: 'Get invoice history for a customer',
  parameters: z.object({
    userId: z.string(),
    // Optional params should use .nullable() for OpenAI tool schema compatibility
    limit: z.number().nullable(),
  }) as unknown as z.ZodType<any>,
  handler: async ({ userId, limit }) => {
    const effectiveLimit = typeof limit === 'number' ? limit : 5;
    // Mock implementation
    return {
      userId,
      invoices: [
        { id: 'inv_001', date: '2024-01-01', amount: '$99.00', status: 'paid' },
        { id: 'inv_002', date: '2023-12-01', amount: '$99.00', status: 'paid' },
        { id: 'inv_003', date: '2023-11-01', amount: '$99.00', status: 'paid' },
      ].slice(0, effectiveLimit),
    };
  },
});

export const billingAgent = createAgent<CustomerSupportState>({
  name: 'Billing Support',
  description: 'Handles billing, payment, subscription, and invoice-related inquiries',
  system: `You are a billing support specialist. You help customers with:
- Subscription management and upgrades/downgrades
- Payment issues and failed transactions
- Refund requests
- Invoice questions
- Billing cycle information

Be helpful, accurate with financial information, and empathetic when dealing with payment issues.
Always confirm customer details before making any changes.`,
  model: openai({
    model: 'gpt-5-nano-2025-08-07',
  }),
  tools: [checkSubscriptionTool, processRefundTool, getInvoiceHistoryTool],
});
