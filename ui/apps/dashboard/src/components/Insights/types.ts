export interface QuerySnapshot {
  createdAt: number;
  id: string;
  name: string;
  query: string;
}

export interface QueryTemplate extends Omit<QuerySnapshot, 'createdAt'> {
  explanation: string;
  templateKind: 'error' | 'success' | 'time' | 'warning';
}

export interface Tab {
  id: string;
  name: string;
  query: string;
  savedQueryId?: string;
}
