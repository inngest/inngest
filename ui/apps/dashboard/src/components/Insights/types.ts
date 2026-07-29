export interface QuerySnapshot {
  id: string;
  isSnapshot: true;
  name: string;
  query: string;
}

export interface QueryTemplate extends Omit<QuerySnapshot, 'isSnapshot'> {
  explanation: string;
  templateKind: 'error' | 'success' | 'time' | 'warning';
}

export interface Tab {
  id: string;
  name: string;
  query: string;
  savedQueryId?: string;
  // Set transiently when a tab is created with a seeded query that should
  // execute automatically on first mount. Cleared by the state machine
  // provider after firing.
  runOnMount?: boolean;
}
