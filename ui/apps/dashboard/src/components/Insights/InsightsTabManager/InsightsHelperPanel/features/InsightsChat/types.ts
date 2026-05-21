// Local message types replacing @inngest/use-agent types

export type { InsightsRealtimeEvent } from '@/lib/inngest/realtime';

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

export type Message = {
  id: string;
  role: 'user' | 'assistant';
  threadId: string;
  parts: MessagePart[];
};
