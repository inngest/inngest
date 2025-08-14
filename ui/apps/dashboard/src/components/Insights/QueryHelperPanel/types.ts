export interface Query {
  id: string;
  isSavedQuery: boolean;
  name: string;
  query: string;
}

export type TemplateKind = 'warning' | 'error' | 'time';

export type Template = Omit<Query, 'type'> & {
  explanation: string;
  templateKind: TemplateKind;
  type: 'template';
};
