export const formOptions = [
  {
    label: 'Report a bug or issue',
    value: 'bug' as const,
  },
  {
    label: 'Book a demo',
    value: 'demo' as const,
  },
  {
    label: 'Billing or payment issue',
    value: 'billing' as const,
  },
  {
    label: 'Suggest a feature',
    value: 'feature' as const,
  },
  {
    label: 'Report a security issue',
    value: 'security' as const,
  },
  {
    label: 'General question or request',
    value: 'question' as const,
  },
];
export type TicketType = (typeof formOptions)[number]['value'] | null;
export const ticketTypeTitles: { [K in Exclude<TicketType, null>]: string } = {
  bug: 'Bug report',
  demo: 'Demo request',
  billing: 'Billing issue',
  feature: 'Feature request',
  security: 'Security report',
  question: 'General question',
};

export const labelTypeIDs: { [K in Exclude<TicketType, null>]: string } = {
  bug: process.env.PLAIN_LABEL_TYPE_ID_BUG || '',
  demo: process.env.PLAIN_LABEL_TYPE_ID_DEMO || '',
  billing: process.env.PLAIN_LABEL_TYPE_ID_BILLING || '',
  feature: process.env.PLAIN_LABEL_TYPE_ID_FEATURE_REQUEST || '',
  security: process.env.PLAIN_LABEL_TYPE_ID_SECURITY || '',
  question: process.env.PLAIN_LABEL_TYPE_ID_QUESTION || '',
} as const;

export function getLabelTitleByTypeId(typeId: string) {
  return Object.entries(labelTypeIDs).find(([, id]) => id === typeId)?.[0] || '';
}

type SeverityOption = {
  label: string;
  description: string;
  value: string;
  paidOnly?: boolean;
  enterpriseOnly?: boolean;
};
export const severityOptions: SeverityOption[] = [
  {
    label: 'Technical guidance',
    description: 'A bug or general question',
    value: '4' as const,
  },
  {
    label: 'Low impact',
    description: 'Service fully usable',
    paidOnly: true,
    value: '3' as const,
  },
  {
    label: 'Medium impact',
    description: 'Production system impaired',
    paidOnly: true,
    value: '2' as const,
  },
  {
    label: 'High impact',
    description: 'Production system down',
    paidOnly: true,
    value: '1' as const,
  },
  {
    label: 'Major impact',
    description: 'Business critical systems down',
    enterpriseOnly: true,
    value: '0' as const,
  },
];
export type BugSeverity = (typeof severityOptions)[number]['value'];
export const DEFAULT_BUG_SEVERITY_LEVEL = '4';
