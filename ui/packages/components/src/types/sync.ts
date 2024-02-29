const syncStatuses = ['duplicate', 'error', 'pending', 'success'] as const;
export type SyncStatus = (typeof syncStatuses)[number];
