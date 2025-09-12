import { createAgent, createTool, openai } from '@inngest/agent-kit';
import { z } from 'zod';

import type { CustomerSupportState } from './state';

// Mock technical support tools
const checkSystemStatusTool = createTool({
  name: 'check_system_status',
  description: 'Check the current status of various system components',
  parameters: z.object({
    component: z
      .enum(['api', 'dashboard', 'webhooks', 'integrations'])
      .describe('System component to check'),
  }),
  handler: async ({ component }) => {
    // Mock implementation
    const statuses = {
      api: { status: 'operational', uptime: '99.98%', lastIncident: '30 days ago' },
      dashboard: { status: 'operational', uptime: '99.95%', lastIncident: '15 days ago' },
      webhooks: { status: 'degraded', uptime: '98.50%', lastIncident: '2 hours ago' },
      integrations: { status: 'operational', uptime: '99.99%', lastIncident: '45 days ago' },
    };
    return statuses[component];
  },
});

const createSupportTicketTool = createTool({
  name: 'create_support_ticket',
  description: 'Create a support ticket for complex technical issues',
  parameters: z.object({
    title: z.string().describe('Brief title of the issue'),
    description: z.string().describe('Detailed description of the issue'),
    priority: z.enum(['low', 'medium', 'high', 'critical']),
    category: z.enum(['bug', 'feature_request', 'integration', 'performance']),
  }),
  handler: async ({ title, description, priority, category }) => {
    // Mock implementation
    return {
      ticketId: `ticket_${Date.now()}`,
      title,
      priority,
      category,
      status: 'open',
      estimatedResponse:
        priority === 'critical' ? '1 hour' : priority === 'high' ? '4 hours' : '24 hours',
      message:
        'Your support ticket has been created. Our team will respond within the estimated time.',
    };
  },
});

const searchKnowledgeBaseTool = createTool({
  name: 'search_knowledge_base',
  description: 'Search the knowledge base for relevant articles and solutions',
  parameters: z.object({
    query: z.string().describe('Search query'),
    limit: z.number().optional(),
  }),
  handler: async ({ query, limit }) => {
    // Mock implementation
    const mockArticles = [
      {
        id: 'kb_001',
        title: 'How to integrate with webhooks',
        excerpt: 'Learn how to set up and configure webhooks for real-time events...',
        url: 'https://docs.example.com/webhooks',
        relevance: 0.95,
      },
      {
        id: 'kb_002',
        title: 'Troubleshooting API authentication',
        excerpt: 'Common issues with API keys and authentication tokens...',
        url: 'https://docs.example.com/auth',
        relevance: 0.87,
      },
      {
        id: 'kb_003',
        title: 'Performance optimization guide',
        excerpt: 'Best practices for optimizing your integration performance...',
        url: 'https://docs.example.com/performance',
        relevance: 0.82,
      },
    ];
    return {
      query,
      results: mockArticles.slice(0, limit),
      totalResults: mockArticles.length,
    };
  },
});

export const technicalSupportAgent = createAgent<CustomerSupportState>({
  name: 'Technical Support',
  description: 'Handles technical issues, bugs, feature requests, and integration help',
  system: `You are a technical support specialist. You help customers with:
- Technical issues and troubleshooting
- Bug reports and system errors
- Feature requests and product feedback
- Integration assistance
- API and webhook configuration
- Performance optimization

Be technical when needed but explain things clearly. Always gather enough information to properly diagnose issues.
If an issue requires engineering attention, create a support ticket with appropriate priority.`,
  model: openai({
    model: 'gpt-5-nano-2025-08-07',
  }),
  tools: [checkSystemStatusTool, createSupportTicketTool, searchKnowledgeBaseTool],
});
