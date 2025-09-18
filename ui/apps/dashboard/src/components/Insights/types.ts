export interface QuerySnapshot {
  createdAt: number;
  id: string;
  name: string;
  query: string;
}

export interface Query extends Omit<QuerySnapshot, 'createdAt'> {
  saved: boolean;
}

export interface UnsavedQuery extends Omit<Query, 'saved'> {
  saved: false;
}

export interface QueryTemplate extends Omit<QuerySnapshot, 'createdAt'> {
  explanation: string;
  templateKind: 'error' | 'success' | 'time' | 'warning';
}
