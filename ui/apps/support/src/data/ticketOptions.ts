export const formOptions = [
  {
    label: "Report a bug or issue",
    value: "bug" as const,
  },
  {
    label: "Book a demo",
    value: "demo" as const,
  },
  {
    label: "Billing or payment issue",
    value: "billing" as const,
  },
  {
    label: "Suggest a feature",
    value: "feature" as const,
  },
  {
    label: "Report a security issue",
    value: "security" as const,
  },
  {
    label: "General question or request",
    value: "question" as const,
  },
];

export type TicketType = (typeof formOptions)[number]["value"] | null;

export const ticketTypeTitles: { [K in Exclude<TicketType, null>]: string } = {
  bug: "Bug report",
  question: "General question",
  billing: "Billing issue",
  demo: "Demo request",
  feature: "Feature request",
  security: "Security report",
};

export const labelTypeIDs: { [K in Exclude<TicketType, null>]: string } = {
  bug: process.env.PLAIN_LABEL_TYPE_ID_BUG || "",
  demo: process.env.PLAIN_LABEL_TYPE_ID_DEMO || "",
  billing: process.env.PLAIN_LABEL_TYPE_ID_BILLING || "",
  feature: process.env.PLAIN_LABEL_TYPE_ID_FEATURE_REQUEST || "",
  security: process.env.PLAIN_LABEL_TYPE_ID_SECURITY || "",
  question: process.env.PLAIN_LABEL_TYPE_ID_QUESTION || "",
} as const;

export const instructions: { [K in Exclude<TicketType, null>]: string } = {
  bug: "Please include any relevant run IDs, function names, event IDs in your message",
  demo: "Please include relevant info like your use cases, estimated volume or specific needs",
  billing: "What is your issue?",
  feature: "What's your idea?",
  security: "Please detail your concern",
  question: "What would you like to know?",
};

type SeverityOption = {
  label: string;
  description: string;
  value: string;
  paidOnly?: boolean;
  enterpriseOnly?: boolean;
};

export const severityOptions: Array<SeverityOption> = [
  {
    label: "P3 - Technical guidance",
    description: "A bug or general question",
    value: "3" as const,
  },
  {
    label: "P2 - Medium impact",
    description: "Production system impaired",
    paidOnly: true,
    value: "2" as const,
  },
  {
    label: "P1 - High impact",
    description: "Production system down",
    paidOnly: true,
    value: "1" as const,
  },
  {
    label: "P0 - Major impact",
    description: "Business critical systems down",
    enterpriseOnly: true,
    value: "0" as const,
  },
];

export type BugSeverity = (typeof severityOptions)[number]["value"];
export const DEFAULT_BUG_SEVERITY_LEVEL = "3";
