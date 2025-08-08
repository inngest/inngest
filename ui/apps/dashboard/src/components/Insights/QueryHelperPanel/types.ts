export interface Query {
  id: string;
  name: string;
  query: string;
  type: 'recent' | 'saved' | 'template';
}
