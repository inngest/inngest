export interface Query {
  id: string;
  isSavedQuery: boolean;
  name: string;
  query: string;
}

export type Template = Omit<Query, 'type'> & { explanation: string; type: 'template' };
