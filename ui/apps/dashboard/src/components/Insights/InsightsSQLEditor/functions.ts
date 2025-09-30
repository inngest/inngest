type FunctionDescriptor = {
  name: string;
  signature: string;
};

const AGGREGATE_FUNCTIONS: FunctionDescriptor[] = [
  { name: 'array_agg', signature: 'array_agg(${1:expr})' },
  { name: 'avg', signature: 'avg(${1:expr})' },
  { name: 'count', signature: 'count(${1:expr})' },
  { name: 'max', signature: 'max(${1:expr})' },
  { name: 'median', signature: 'median(${1:expr})' },
  { name: 'min', signature: 'min(${1:expr})' },
  { name: 'stddev_pop', signature: 'stddev_pop(${1:expr})' },
  { name: 'stddev_samp', signature: 'stddev_samp(${1:expr})' },
  { name: 'sum', signature: 'sum(${1:expr})' },
  { name: 'var_pop', signature: 'var_pop(${1:expr})' },
  { name: 'var_samp', signature: 'var_samp(${1:expr})' },
];

const COMPARISON_FUNCTIONS: FunctionDescriptor[] = [
  { name: 'equals', signature: 'equals(${1:a}, ${2:b})' },
  { name: 'greater', signature: 'greater(${1:a}, ${2:b})' },
  { name: 'greaterOrEquals', signature: 'greaterOrEquals(${1:a}, ${2:b})' },
  { name: 'less', signature: 'less(${1:a}, ${2:b})' },
  { name: 'lessOrEquals', signature: 'lessOrEquals(${1:a}, ${2:b})' },
  { name: 'notEquals', signature: 'notEquals(${1:a}, ${2:b})' },
];

export const SUPPORTED_FUNCTIONS = sortByName([...AGGREGATE_FUNCTIONS, ...COMPARISON_FUNCTIONS]);

function sortByName(fns: FunctionDescriptor[]): FunctionDescriptor[] {
  return fns.sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
}
