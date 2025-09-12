import {
  createNetwork,
  createRoutingAgent,
  createTool,
  openai,
  type AgentResult,
  type Network,
  type State,
  type TextMessage,
  type ToolCallMessage,
  type ToolResultMessage,
  type UserMessage,
} from '@inngest/agent-kit';
import { z } from 'zod';

import { billingAgent } from './billing';
import type { CustomerSupportState } from './state';
import { technicalSupportAgent } from './technical-support';

// Utility functions for checking message types and getting last result
function lastResult(results: AgentResult[] | undefined) {
  if (!results) {
    return undefined;
  }
  return results[results.length - 1];
}

type MessageType = TextMessage['type'] | ToolCallMessage['type'] | ToolResultMessage['type'];

function isLastMessageOfType(result: AgentResult, type: MessageType) {
  return result.output[result.output.length - 1]?.type === type;
}

// Create a routing agent that mirrors the default routing agent's interface
const customerSupportRouter = createRoutingAgent<CustomerSupportState>({
  name: 'Customer Support Router',
  description:
    'Selects which support agent should handle the next step, or completes the conversation.',
  model: openai({ model: 'gpt-5-nano-2025-08-07' }),
  lifecycle: {
    onRoute: ({ result, network }) => {
      // State-driven short-circuit: if an agent was already invoked this turn,
      // end the turn to wait for a new user message.
      if (network?.state?.data?.invoked) {
        console.log(
          '🔧 [ROUTER-STATE] Agent already invoked this turn:',
          network.state.data.invokedAgentName
        );
        return undefined;
      }

      const tool = result.toolCalls[0] as any;
      if (!tool) {
        return;
      }
      if (tool.tool.name === 'done') {
        return undefined;
      }
      if (tool.tool.name === 'select_agent') {
        if (
          typeof tool.content === 'object' &&
          tool.content !== null &&
          'data' in tool.content &&
          typeof (tool.content as any).data === 'string'
        ) {
          const next = (tool.content as any).data as string;
          // Mark state as invoked so subsequent router calls in this run end
          try {
            if (network) {
              network.state.data.invoked = true;
              network.state.data.invokedAgentName = next;
            }
          } catch {}
          return [next];
        }
      }
      return;
    },
  },

  tools: [
    // Agent selection tool, identical schema to the default router
    (createTool as any)({
      name: 'select_agent',
      description: 'Select an agent to handle the next step of the conversation',
      parameters: z
        .object({
          name: z.string().describe('The name of the agent that should handle the request'),
          reason: z.string().describe('Brief explanation of why this agent was chosen'),
        })
        .strict() as unknown as z.ZodType<any>,
      handler: (args: any, ctx: any) => {
        const { name, reason } = args as { name: string; reason: string };
        const { network } = ctx as { network: Network<CustomerSupportState> };
        if (typeof name !== 'string') {
          throw new Error('The routing agent requested an invalid agent');
        }

        const agent = network.agents.get(name);
        if (agent === undefined) {
          throw new Error(`The routing agent requested an agent that doesn't exist: ${name}`);
        }

        // Record routing decision in state for observability
        const now = new Date().toISOString();
        network.state.data.lastRoutingDecision = {
          nextAgent: agent.name,
          reason: reason || `Routing to ${agent.name}`,
          timestamp: now,
        };
        const prior = Array.isArray(network.state.data.routingDecisions)
          ? network.state.data.routingDecisions
          : [];
        network.state.data.routingDecisions = [
          ...prior,
          { agent: agent.name, reason: reason || `Routing to ${agent.name}`, timestamp: now },
        ];

        // Returning the agent name mirrors the default router's behavior
        return agent.name;
      },
    }),

    // Done tool to explicitly end the conversation when appropriate
    (createTool as any)({
      name: 'done',
      description: 'Signal that the conversation is complete and no more agents need to be called',
      parameters: z
        .object({
          summary: z.string().describe('Brief summary of what was accomplished'),
        })
        .strict() as unknown as z.ZodType<any>,
      handler: (args: any, ctx: any) => {
        const { summary } = args as { summary?: string };
        const { network } = ctx as { network: Network<CustomerSupportState> };
        network.state.data.conversationComplete = true;
        network.state.data.lastRoutingDecision = {
          nextAgent: 'DONE',
          reason: summary || 'Conversation completed successfully',
          timestamp: new Date().toISOString(),
        };
        return summary || 'Conversation completed successfully';
      },
    }),
  ] as any,

  // Allow the model to choose between selecting an agent or finishing
  tool_choice: 'any',

  system: async ({ network }) => {
    if (!network) {
      throw new Error('The routing agent can only be used within a network of agents');
    }

    const agents = await network.availableAgents();

    return `You are the orchestrator between a group of customer support agents. Each agent is suited for specific tasks and has a name, description, and tools.

The following agents are available:
<agents>
  ${agents
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    .map((a: any) => {
      return `
    <agent>
      <name>${a.name}</name>
      <description>${a.description}</description>
      <tools>${JSON.stringify(Array.from(a.tools.values()))}</tools>
    </agent>`;
    })
    .join('\n')}
</agents>

Your responsibilities:
1. Analyze the conversation history and current state
2. Determine if the request has been completed or if more work is needed
3. Either:
   - Call select_agent to route to the appropriate agent for the next step
   - Call done if the conversation is complete or the user's request has been fulfilled

Turn handling and completion policy:
- If the last agent's reply is asking the user for information or confirmation, end this turn by calling the done tool so the system can wait for the user's reply. Do not immediately route to any agent again until a new user message arrives.
- Do not route the same agent twice in a row without a new user message that changes context.
- Prefer the minimal number of agent hops necessary to satisfy the user.
- When an agent makes a tool call, the system will automatically route back to that same agent to handle the tool result.

**CRITICAL: When to call DONE:**
- If a billing agent has processed a refund (status: "pending_approval" or "completed")
- If a technical agent has provided a solution or answer
- If the customer's original request has been addressed
- If an agent has taken the requested action (refund, subscription change, etc.)

**Agent Specializations:**
- Billing Support: payment, subscription, invoice, and refund issues
- Technical Support: bugs, errors, feature questions, and integrations

**BE EFFICIENT:** Most customer requests can be resolved by ONE agent. Only route to multiple agents if the request genuinely requires different specializations. After an agent has processed the customer's request, call DONE.
`;
  },
});

// Hybrid router: code-first guard, then delegate to routing agent for selection
const hybridRouter: Network.Router<CustomerSupportState> = async ({
  input,
  userMessage,
  network,
}) => {
  // If an agent has already been scheduled/invoked in this run, end the turn
  if (network.state.data.invoked) {
    // console.log("🔧 [ROUTER] Agent already invoked this turn, ending:", network.state.data.invokedAgentName);
    return undefined;
  }

  // Check if the last result was a tool call, if so route back to the previous agent
  const lastAgentResult = lastResult(network.state.results);

  if (lastAgentResult && isLastMessageOfType(lastAgentResult, 'tool_call')) {
    const previousAgent = network.agents.get(lastAgentResult.agentName);
    if (previousAgent) {
      // Mark state so subsequent router call (same run) exits
      network.state.data.invoked = true;
      network.state.data.invokedAgentName = previousAgent.name;

      return previousAgent;
    } else {
      console.log('🔧 [ROUTER] Could not find previous agent:', lastAgentResult.agentName);
    }
  }

  // Delegate selection to the routing agent (always uses string input for LLM routing)
  const result = await customerSupportRouter.run(input, {
    network,
    model: customerSupportRouter.model || network.defaultModel,
  });

  const agentNames = customerSupportRouter.lifecycles.onRoute({
    result,
    agent: customerSupportRouter,
    network,
  });

  const nextName = Array.isArray(agentNames) ? agentNames[0] : undefined;
  if (!nextName) {
    return undefined;
  }

  const next = network.agents.get(nextName);
  if (!next) {
    return undefined;
  }

  // Mark state so subsequent router call (same run) exits
  network.state.data.invoked = true;
  network.state.data.invokedAgentName = next.name;

  return next;
};

// Factory function to create the customer support network with routing logic only
export function createCustomerSupportNetwork(
  threadId: string,
  initialState: State<CustomerSupportState>,
  historyAdapter?: any
) {
  return createNetwork<CustomerSupportState>({
    name: 'Customer Support Network',
    description: 'Handles customer support inquiries with specialized agents',
    agents: [billingAgent, technicalSupportAgent],
    defaultModel: openai({ model: 'gpt-5-nano-2025-08-07' }),
    maxIter: 10, // Allow proper conversation flow
    defaultState: initialState,
    router: hybridRouter, // Hybrid: code-based guard + routing agent selection
    history: historyAdapter,
  });
}
