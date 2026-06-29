export type InsightsSchemaCatalog = {
  version: string;
  tables: InsightsTable[];
};

export type InsightsTable = {
  name: string;
  description: string;
  notes: string[];
  defaultTimeColumn?: string;
  columns: InsightsColumn[];
};

export type InsightsColumn = {
  name: string;
  type: string;
  description: string;
  notes?: string[];
  examples?: string[];
  children?: InsightsColumn[];
};

export type InsightsSchemaMetadata = {
  title: string;
  description: string;
  notes?: string[];
  examples?: string[];
};
