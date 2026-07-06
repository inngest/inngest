import { afterEach, describe, expect, it, vi } from 'vitest';

import { fetchLessons, newLessonEvents } from './lessons';

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

function stubEventsApi(events: unknown[], ok = true) {
  vi.stubEnv('INNGEST_SIGNING_KEY', 'signkey-test');
  const fetchMock = vi.fn(async () => ({
    ok,
    json: async () => ({ data: events }),
  }));
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

describe('fetchLessons', () => {
  it('returns [] without a signing key', async () => {
    vi.stubEnv('INNGEST_SIGNING_KEY', '');
    expect(await fetchLessons()).toEqual([]);
  });

  it('fetches, dedupes by code (newest first), and caps the list', async () => {
    const fetchMock = stubEventsApi([
      { data: { code: 'bad_fn', message: 'newest message', sql: 'SELECT a' } },
      { data: { code: 'bad_fn', message: 'older message', sql: 'SELECT b' } },
      { data: { code: 'bad_col', message: 'no such column', sql: 'SELECT c' } },
      { data: { malformed: true } },
    ]);

    const lessons = await fetchLessons();

    expect(fetchMock).toHaveBeenCalledWith(
      'https://api.inngest.com/v1/events?name=insights-agent%2Flesson.recorded&limit=50',
      { headers: { Authorization: 'Bearer signkey-test' } },
    );
    expect(lessons).toEqual([
      { code: 'bad_fn', message: 'newest message', sql: 'SELECT a' },
      { code: 'bad_col', message: 'no such column', sql: 'SELECT c' },
    ]);
  });

  it('returns [] on API failure', async () => {
    stubEventsApi([], false);
    expect(await fetchLessons()).toEqual([]);
  });

  it('returns [] when fetch throws', async () => {
    vi.stubEnv('INNGEST_SIGNING_KEY', 'signkey-test');
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new Error('network down');
      }),
    );
    expect(await fetchLessons()).toEqual([]);
  });
});

describe('newLessonEvents', () => {
  it('skips already-known codes and dedupes within the run', async () => {
    const events = newLessonEvents(
      [
        { code: 'bad_fn', message: 'm1', sql: 'SELECT 1' },
        { code: 'bad_col', message: 'm2', sql: 'SELECT 2' },
        { code: 'bad_col', message: 'm3', sql: 'SELECT 3' },
      ],
      [{ code: 'bad_fn', message: 'known', sql: '' }],
    );

    expect(events).toEqual([
      {
        name: 'insights-agent/lesson.recorded',
        data: { code: 'bad_col', message: 'm2', sql: 'SELECT 2' },
      },
    ]);
  });

  it('truncates long SQL', () => {
    const events = newLessonEvents(
      [{ code: 'c', message: 'm', sql: 'x'.repeat(500) }],
      [],
    );
    expect(events[0]?.data.sql).toHaveLength(200);
  });
});
