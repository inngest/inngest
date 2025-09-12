// State type for the customer support network
export interface CustomerSupportState {
  userId?: string;
  department?: 'billing' | 'technical';
  triageComplete?: boolean;
  conversationComplete?: boolean;
  issueResolved?: boolean;
  // Routing control flags for single-agent-per-turn policy
  invoked?: boolean;
  invokedAgentName?: string;
  lastRoutingDecision?: {
    nextAgent: string;
    reason: string;
    timestamp: string;
  };
  routingDecisions?: Array<{
    agent: string;
    reason: string;
    timestamp: string;
  }>;
}
