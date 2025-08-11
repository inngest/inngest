export interface Query {
  id: string;
  name: string;
  query: string;
  type: 'new' | 'recent' | 'saved' | 'template';
}
