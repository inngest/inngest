import type { FunctionDescriptor } from "./types";

export const ARITHMETIC_FUNCTIONS: FunctionDescriptor[] = [
  { name: "abs", signature: "abs(${1:val})" },
  { name: "byteSwap", signature: "byteSwap(${1:val})" },
  { name: "divide", signature: "divide(${1:a}, ${2:b})" },
  { name: "gcd", signature: "gcd(${1:a}, ${2:b})" },
  { name: "ifNotFinite", signature: "ifNotFinite(${1:a}, ${2:b})" },
  { name: "intDiv", signature: "intDiv(${1:a}, ${2:b})" },
  { name: "intDivOrZero", signature: "intDivOrZero(${1:a}, ${2:b})" },
  { name: "isFinite", signature: "isFinite(${1:val})" },
  { name: "isInfinite", signature: "isInfinite(${1:val})" },
  { name: "isNaN", signature: "isNaN(${1:val})" },
  { name: "lcm", signature: "lcm(${1:a}, ${2:b})" },
  { name: "max2", signature: "max2(${1:a}, ${2:b})" },
  { name: "min2", signature: "min2(${1:a}, ${2:b})" },
  { name: "minus", signature: "minus(${1:a}, ${2:b})" },
  { name: "modulo", signature: "modulo(${1:a}, ${2:b})" },
  { name: "moduloOrZero", signature: "moduloOrZero(${1:a}, ${2:b})" },
  { name: "multiply", signature: "multiply(${1:a}, ${2:b})" },
  { name: "negate", signature: "negate(${1:val})" },
  { name: "plus", signature: "plus(${1:a}, ${2:b})" },
  { name: "positiveModulo", signature: "positiveModulo(${1:a}, ${2:b})" },
];
