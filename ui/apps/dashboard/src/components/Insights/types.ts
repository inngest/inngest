export interface InsightTableRow {
  id: string;
  row: number;
  properties: Record<
    string,
    {
      value: unknown;
      type: 'string' | 'number' | 'date';
    }
  >;
}
