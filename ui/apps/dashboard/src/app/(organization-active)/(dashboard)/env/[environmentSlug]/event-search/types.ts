import type { Event as FullEvent } from '@inngest/components/types/event';

export type Event = Pick<FullEvent, 'id' | 'name' | 'receivedAt'>;
