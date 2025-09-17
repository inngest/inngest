export interface QuerySnapshot {
  createdAt: number;
  id: string;
  name: string;
  query: string;
}

export interface Query extends Omit<QuerySnapshot, 'createdAt'> {
  savedQueryId: string | undefined;
}

export interface UnsavedQuery extends Query {
  savedQueryId: undefined;
}

export interface QueryTemplate extends Omit<QuerySnapshot, 'createdAt'> {
  explanation: string;
  templateKind: 'error' | 'time' | 'warning';
}

export interface Tab {
  id: string;
  name: string;
  query: string;
  savedQueryId?: string;
}
