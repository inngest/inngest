export interface QuerySnapshot {
  createdAt: number;
  id: string;
  name: string;
  query: string;
}

export interface Query extends Omit<QuerySnapshot, 'createdAt'> {
  savedQueryId?: string;
}

export interface UnsavedQuery extends Query {
  savedQueryId?: undefined;
}

export interface QueryTemplate extends Omit<QuerySnapshot, 'createdAt'> {
  explanation: string;
  templateKind: 'error' | 'time' | 'warning';
}
