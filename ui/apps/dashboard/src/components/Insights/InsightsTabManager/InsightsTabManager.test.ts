import { describe, expect, it } from 'vitest';

import { openQueryTabInState, type TabsState } from './InsightsTabManager';
import { HOME_TAB } from './constants';

describe('openQueryTabInState', () => {
  it('atomically creates and activates a missing saved-query tab', () => {
    const state: TabsState = {
      tabs: [
        HOME_TAB,
        {
          id: 'tab-A',
          name: 'Query A',
          query: 'SELECT 11',
          savedQueryId: 'saved-A',
        },
      ],
      activeTabId: 'tab-A',
    };

    expect(
      openQueryTabInState(state, {
        id: 'new-tab-B',
        name: 'Query B',
        query: 'SELECT 29',
        savedQueryId: 'saved-B',
      }),
    ).toEqual({
      tabs: [
        ...state.tabs,
        {
          id: 'new-tab-B',
          name: 'Query B',
          query: 'SELECT 29',
          savedQueryId: 'saved-B',
        },
      ],
      activeTabId: 'new-tab-B',
    });
  });

  it('atomically focuses an existing saved-query tab without replacing it', () => {
    const existingB = {
      id: 'restored-B',
      name: 'Locally edited B',
      query: 'SELECT 31',
      savedQueryId: 'saved-B',
    };
    const state: TabsState = {
      tabs: [HOME_TAB, existingB],
      activeTabId: HOME_TAB.id,
    };

    expect(
      openQueryTabInState(state, {
        id: 'discarded-new-ID',
        name: 'Server B',
        query: 'SELECT 29',
        savedQueryId: 'saved-B',
      }),
    ).toEqual({ tabs: [HOME_TAB, existingB], activeTabId: 'restored-B' });
  });
});
