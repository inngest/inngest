import { insightsSchemaCatalog } from '@/lib/insights/schema/catalog';
import { renderInsightsSchemaForExplorer } from '@/lib/insights/schema/renderExplorer';

const rendered = renderInsightsSchemaForExplorer(insightsSchemaCatalog);

export const tableEntries = rendered.entries;
export const tableMetadataByPath = rendered.metadataByPath;
