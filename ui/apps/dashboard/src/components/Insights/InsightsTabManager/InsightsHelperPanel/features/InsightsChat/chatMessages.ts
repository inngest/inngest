import type { Message, MessagePart, RunRef } from './types';

// Per-thread messages, and the only chat state there is. A turn in flight is an
// assistant message with no parts yet, so "is this thread waiting", "which runs
// should we poll" and "where does this result go" are all answered by the same
// array — there is no second map to keep in step with it.
export type Threads = Record<string, Message[]>;

function isWaitingTurn(message: Message): boolean {
  return message.parts.length === 0;
}

/** Applies `fn` to a turn that is still waiting, and does nothing otherwise. */
function mapWaitingTurn(
  threads: Threads,
  threadId: string,
  messageId: string,
  fn: (message: Message) => Message,
): Threads {
  const messages = threads[threadId];
  if (!messages?.some((m) => m.id === messageId && isWaitingTurn(m))) {
    return threads;
  }

  return {
    ...threads,
    [threadId]: messages.map((m) => (m.id === messageId ? fn(m) : m)),
  };
}

export function startTurn(
  threads: Threads,
  threadId: string,
  userMessage: Message,
  turnId: string,
): Threads {
  return {
    ...threads,
    [threadId]: [
      ...(threads[threadId] ?? []),
      userMessage,
      { id: turnId, role: 'assistant', threadId, parts: [] },
    ],
  };
}

export function attachRun(
  threads: Threads,
  threadId: string,
  turnId: string,
  run: RunRef,
): Threads {
  return mapWaitingTurn(threads, threadId, turnId, (m) => ({ ...m, run }));
}

/**
 * Replaces a waiting turn with its result, dropping the turn entirely when
 * there's nothing to render rather than leaving an empty bubble.
 *
 * A turn that already settled — or whose thread was cleared while the run was
 * in flight — is left alone, so applying the same result twice is harmless.
 */
export function settleTurn(
  threads: Threads,
  threadId: string,
  turnId: string,
  parts: MessagePart[],
): Threads {
  if (parts.length === 0) {
    const messages = threads[threadId];
    if (!messages) return threads;
    return {
      ...threads,
      [threadId]: messages.filter(
        (m) => !(m.id === turnId && isWaitingTurn(m)),
      ),
    };
  }

  return mapWaitingTurn(threads, threadId, turnId, (m) => ({
    id: m.id,
    role: m.role,
    threadId: m.threadId,
    parts,
  }));
}

export function clearThread(threads: Threads, threadId: string): Threads {
  if (!(threadId in threads)) return threads;
  const next = { ...threads };
  delete next[threadId];
  return next;
}

export function isThreadWaiting(threads: Threads, threadId: string): boolean {
  return (threads[threadId] ?? []).some(isWaitingTurn);
}

/** Every turn with a run to read back, which is what the UI polls for. */
export function pendingRuns(
  threads: Threads,
): { threadId: string; turnId: string; run: RunRef }[] {
  return Object.entries(threads).flatMap(([threadId, messages]) =>
    messages
      .filter((m) => isWaitingTurn(m) && m.run)
      .map((m) => ({ threadId, turnId: m.id, run: m.run as RunRef })),
  );
}
