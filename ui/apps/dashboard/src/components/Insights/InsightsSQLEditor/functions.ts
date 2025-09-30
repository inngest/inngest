export const SUPPORTED_FUNCTIONS = [
  { name: 'AVG', signature: 'AVG(${1:column})' },
  { name: 'COUNT', signature: 'COUNT(${1:column})' },
  { name: 'MAX', signature: 'MAX(${1:column})' },
  { name: 'MIN', signature: 'MIN(${1:column})' },
  { name: 'SUM', signature: 'SUM(${1:column})' },
] as const;
