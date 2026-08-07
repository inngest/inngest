// Local message types replacing @inngest/use-agent types

export type TextPart = {
  type: 'text';
  content: string;
};

export type ToolCallPart = {
  type: 'tool-call';
  toolName: 'generate_sql';
  data: { sql: string; title?: string; reasoning?: string };
  error?: string;
};

export type MessagePart = TextPart | ToolCallPart;

/** The agent run behind a waiting turn, and the proof that it's ours. */
export type RunRef = { eventId: string; receipt: string };

export type Message = {
  id: string;
  role: 'user' | 'assistant';
  threadId: string;
  // Empty on the assistant placeholder that stands in for a turn until its run
  // reports a result. See chatMessages.
  parts: MessagePart[];
  // Set on that placeholder once /api/chat returns the event it sent.
  run?: RunRef;
};
