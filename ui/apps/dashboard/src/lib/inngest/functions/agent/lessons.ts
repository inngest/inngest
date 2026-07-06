import type { ValidationFailure } from './loop';

// Self-learning across sessions: every SQL the agent wrote that failed
// validation is recorded as an event in this app's own Inngest environment
// (write: the existing event key; read: the /v1/events REST API with the
// signing key serve() already requires). Lessons age out with event retention.

export const LESSON_EVENT = 'insights-agent/lesson.recorded';

const MAX_LESSONS = 10;
const MAX_LESSON_SQL_CHARS = 200;

export interface Lesson {
  code: string;
  message: string;
  sql: string;
}

/** Fetch recent lessons, deduplicated by diagnostic code, newest first. */
export async function fetchLessons(): Promise<Lesson[]> {
  const signingKey = process.env.INNGEST_SIGNING_KEY;
  if (!signingKey) return [];

  const baseUrl = process.env.INNGEST_BASE_URL ?? 'https://api.inngest.com';
  try {
    const res = await fetch(
      `${baseUrl}/v1/events?name=${encodeURIComponent(LESSON_EVENT)}&limit=50`,
      { headers: { Authorization: `Bearer ${signingKey}` } },
    );
    if (!res.ok) return [];

    const body = (await res.json()) as {
      data?: { data?: Record<string, unknown> }[];
    };
    const byCode = new Map<string, Lesson>();
    for (const event of body.data ?? []) {
      const { code, message, sql } = event.data ?? {};
      if (typeof code !== 'string' || typeof message !== 'string') continue;
      if (byCode.has(code)) continue; // newest first — keep the first seen
      byCode.set(code, {
        code,
        message,
        sql: typeof sql === 'string' ? sql : '',
      });
      if (byCode.size >= MAX_LESSONS) break;
    }
    return [...byCode.values()];
  } catch {
    return []; // lessons are best-effort; never block a run on them
  }
}

/** Turn this run's validation failures into lesson events, skipping codes we already know. */
export function newLessonEvents(
  failures: ValidationFailure[],
  known: Lesson[],
): { name: typeof LESSON_EVENT; data: Lesson }[] {
  const knownCodes = new Set(known.map((lesson) => lesson.code));
  const events: { name: typeof LESSON_EVENT; data: Lesson }[] = [];
  for (const failure of failures) {
    if (knownCodes.has(failure.code)) continue;
    knownCodes.add(failure.code);
    events.push({
      name: LESSON_EVENT,
      data: {
        code: failure.code,
        message: failure.message,
        sql: failure.sql.slice(0, MAX_LESSON_SQL_CHARS),
      },
    });
  }
  return events;
}
