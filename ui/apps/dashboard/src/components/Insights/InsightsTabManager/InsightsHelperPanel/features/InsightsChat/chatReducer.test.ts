import { describe, expect, it } from 'vitest';

import {
  assistantMessage,
  chatReducer,
  initialChatState,
  type ChatState,
} from './chatReducer';
import type { Message } from './types';

const THREAD = 'thread_1';

const userMessage: Message = {
  id: 'msg_1',
  role: 'user',
  threadId: THREAD,
  parts: [{ type: 'text', content: 'how many runs failed?' }],
};

function afterSend(): ChatState {
  return chatReducer(initialChatState, {
    type: 'send',
    threadId: THREAD,
    message: userMessage,
  });
}

const answer = () =>
  assistantMessage(THREAD, [{ type: 'text', content: '12 runs' }]);

describe('chatReducer', () => {
  it('marks the thread pending as soon as the message is sent', () => {
    const state = afterSend();

    expect(state.pendingByThread).toEqual({ [THREAD]: '' });
    expect(state.messagesByThread[THREAD]).toEqual([userMessage]);
  });

  it('records the event id so the result can be recovered', () => {
    const state = chatReducer(afterSend(), {
      type: 'sent',
      threadId: THREAD,
      eventId: 'evt_1',
    });

    expect(state.pendingByThread).toEqual({ [THREAD]: 'evt_1' });
  });

  it('ignores the event id once the run has already answered', () => {
    const answered = chatReducer(afterSend(), {
      type: 'result',
      threadId: THREAD,
      message: answer(),
    });

    const state = chatReducer(answered, {
      type: 'sent',
      threadId: THREAD,
      eventId: 'evt_1',
    });

    expect(state).toBe(answered);
    expect(state.pendingByThread).toEqual({});
  });

  it('applies the first result and clears the thread', () => {
    const state = chatReducer(afterSend(), {
      type: 'result',
      threadId: THREAD,
      message: answer(),
    });

    expect(state.messagesByThread[THREAD]).toHaveLength(2);
    expect(state.pendingByThread).toEqual({});
  });

  it('drops a second result for the same run', () => {
    // Realtime and the recovery poll both deliver; only one may land.
    const first = chatReducer(afterSend(), {
      type: 'result',
      threadId: THREAD,
      message: answer(),
    });

    const second = chatReducer(first, {
      type: 'result',
      threadId: THREAD,
      message: answer(),
    });

    expect(second).toBe(first);
    expect(second.messagesByThread[THREAD]).toHaveLength(2);
  });

  it('drops a result for a thread the user cleared', () => {
    const cleared = chatReducer(afterSend(), {
      type: 'clearThread',
      threadId: THREAD,
    });

    const state = chatReducer(cleared, {
      type: 'result',
      threadId: THREAD,
      message: answer(),
    });

    expect(state.messagesByThread[THREAD]).toBeUndefined();
    expect(state.pendingByThread).toEqual({});
  });

  it('stops the spinner but keeps the thread when a result has no parts', () => {
    const state = chatReducer(afterSend(), {
      type: 'result',
      threadId: THREAD,
      message: null,
    });

    expect(state.messagesByThread[THREAD]).toEqual([userMessage]);
    expect(state.pendingByThread).toEqual({});
  });

  it('rolls back the optimistic user message when the send fails', () => {
    const state = chatReducer(afterSend(), {
      type: 'sendFailed',
      threadId: THREAD,
      messageId: userMessage.id,
      message: assistantMessage(THREAD, [
        { type: 'text', content: 'Error: nope' },
      ]),
    });

    expect(state.messagesByThread[THREAD]).toHaveLength(1);
    expect(state.messagesByThread[THREAD]?.[0]?.role).toBe('assistant');
    expect(state.pendingByThread).toEqual({});
  });

  it('keeps threads independent', () => {
    const other = chatReducer(afterSend(), {
      type: 'send',
      threadId: 'thread_2',
      message: { ...userMessage, id: 'msg_2', threadId: 'thread_2' },
    });

    const state = chatReducer(other, {
      type: 'result',
      threadId: THREAD,
      message: answer(),
    });

    expect(state.pendingByThread).toEqual({ thread_2: '' });
    expect(state.messagesByThread['thread_2']).toHaveLength(1);
  });
});
