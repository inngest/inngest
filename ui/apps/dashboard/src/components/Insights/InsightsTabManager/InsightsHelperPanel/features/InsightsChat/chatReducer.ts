import type { Message, MessagePart } from './types';

export type ChatState = {
  messagesByThread: Record<string, Message[]>;
  // Threads waiting on an agent run: threadId -> the id of the event that
  // triggered it, or '' when /api/chat didn't return one. Presence is the
  // single source of truth for the spinner, for whether the recovery poll
  // runs, and for whether a result still has somewhere to land.
  pendingByThread: Record<string, string>;
};

export type ChatAction =
  | { type: 'send'; threadId: string; message: Message }
  | { type: 'sent'; threadId: string; eventId: string }
  | {
      type: 'sendFailed';
      threadId: string;
      messageId: string;
      message: Message;
    }
  | { type: 'result'; threadId: string; message: Message | null }
  | { type: 'clearThread'; threadId: string };

export const initialChatState: ChatState = {
  messagesByThread: {},
  pendingByThread: {},
};

export function assistantMessage(
  threadId: string,
  parts: MessagePart[],
): Message {
  return { id: crypto.randomUUID(), role: 'assistant', threadId, parts };
}

function omit<T>(map: Record<string, T>, key: string): Record<string, T> {
  if (!(key in map)) return map;
  const next = { ...map };
  delete next[key];
  return next;
}

export function chatReducer(state: ChatState, action: ChatAction): ChatState {
  const { threadId } = action;
  const messages = state.messagesByThread[threadId] ?? [];

  switch (action.type) {
    case 'send':
      return {
        messagesByThread: {
          ...state.messagesByThread,
          [threadId]: [...messages, action.message],
        },
        pendingByThread: { ...state.pendingByThread, [threadId]: '' },
      };

    case 'sent':
      // A fast run can answer before /api/chat even returns, leaving nothing
      // left to recover.
      if (!(threadId in state.pendingByThread)) return state;
      return {
        ...state,
        pendingByThread: {
          ...state.pendingByThread,
          [threadId]: action.eventId,
        },
      };

    case 'sendFailed':
      return {
        messagesByThread: {
          ...state.messagesByThread,
          [threadId]: [
            ...messages.filter((m) => m.id !== action.messageId),
            action.message,
          ],
        },
        pendingByThread: omit(state.pendingByThread, threadId),
      };

    case 'result':
      // Realtime and the recovery poll carry the same result, and a cleared
      // thread wants neither: whichever arrives first empties the pending
      // entry and everything after it no-ops here.
      if (!(threadId in state.pendingByThread)) return state;
      return {
        messagesByThread: action.message
          ? {
              ...state.messagesByThread,
              [threadId]: [...messages, action.message],
            }
          : state.messagesByThread,
        pendingByThread: omit(state.pendingByThread, threadId),
      };

    case 'clearThread':
      return {
        messagesByThread: omit(state.messagesByThread, threadId),
        pendingByThread: omit(state.pendingByThread, threadId),
      };
  }
}
