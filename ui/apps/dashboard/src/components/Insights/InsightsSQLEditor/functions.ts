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

const ARITHMETIC_FUNCTIONS: FunctionDescriptor[] = [
  { name: 'abs', signature: 'abs(${1:val})' },
  { name: 'byteswap', signature: 'byteswap(${1:val})' },
  { name: 'divide', signature: 'divide(${1:a}, ${2:b})' },
  // { name: 'divideDecimal', signature: 'divideDecimal(${1:a}, ${2:b})' }, error
  //{ name: 'divideOrNull', signature: 'divideOrNull(${1:a}, ${2:b})' }, base 16
  { name: 'gcd', signature: 'gcd(${1:a}, ${2:b})' },
  { name: 'ifNotFinite', signature: 'ifNotFinite(${1:a}, ${2:b})' },
  { name: 'intDiv', signature: 'intDiv(${1:a}, ${2:b})' },
  // { name: 'intDivOrNull', signature: 'intDivOrNull(${1:a}, ${2:b})' }, base 16
  { name: 'intDivOrZero', signature: 'intDivOrZero(${1:a}, ${2:b})' },
  { name: 'isFinite', signature: 'isFinite(${1:val})' },
  { name: 'isInfinite', signature: 'isInfinite(${1:val})' },
  { name: 'isNaN', signature: 'isNaN(${1:val})' },
  { name: 'lcm', signature: 'lcm(${1:a}, ${2:b})' },
  { name: 'max2', signature: 'max2(${1:a}, ${2:b})' },
  { name: 'min2', signature: 'min2(${1:a}, ${2:b})' },
  { name: 'minus', signature: 'minus(${1:a}, ${2:b})' },
  { name: 'modulo', signature: 'modulo(${1:a}, ${2:b})' },
  // { name: 'moduloOrNull', signature: 'moduloOrNull(${1:a}, ${2:b})' }, base 16
  { name: 'moduloOrZero', signature: 'moduloOrZero(${1:a}, ${2:b})' },
  { name: 'multiply', signature: 'multiply(${1:a}, ${2:b})' },
  // { name: 'multiplyDecimal', signature: 'multiplyDecimal(${1:a}, ${2:b})' }, error
  { name: 'negate', signature: 'negate(${1:val})' },
  { name: 'plus', signature: 'plus(${1:a}, ${2:b})' },
  { name: 'positivemodulo', signature: 'positivemodulo(${1:a}, ${2:b})' },
  // { name: 'positivemoduloornull', signature: 'positivemoduloornull(${1:a}, ${2:b})' }, base 16
];

const COMPARISON_FUNCTIONS: FunctionDescriptor[] = [
  { name: 'equals', signature: 'equals(${1:a}, ${2:b})' },
  { name: 'greater', signature: 'greater(${1:a}, ${2:b})' },
  { name: 'greaterOrEquals', signature: 'greaterOrEquals(${1:a}, ${2:b})' },
  { name: 'less', signature: 'less(${1:a}, ${2:b})' },
  { name: 'lessOrEquals', signature: 'lessOrEquals(${1:a}, ${2:b})' },
  { name: 'notEquals', signature: 'notEquals(${1:a}, ${2:b})' },
];

export const SUPPORTED_FUNCTIONS = sortByName([
  ...AGGREGATE_FUNCTIONS,
  ...ARITHMETIC_FUNCTIONS,
  ...COMPARISON_FUNCTIONS,
]);

function sortByName(fns: FunctionDescriptor[]): FunctionDescriptor[] {
  return fns.sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
}
