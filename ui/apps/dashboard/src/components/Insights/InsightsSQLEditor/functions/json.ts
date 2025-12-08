import type { FunctionDescriptor } from "./types";

export const JSON_FUNCTIONS: FunctionDescriptor[] = [
  {
    name: "simpleJSONHas",
    signature: "simpleJSONHas(${1:json}, ${2:field_name})",
  },

  {
    name: "simpleJSONExtractUInt",
    signature: "simpleJSONExtractUInt(${1:json}, ${2:field_name})",
  },
  {
    name: "simpleJSONExtractInt",
    signature: "simpleJSONExtractInt(${1:json}, ${2:field_name})",
  },
  {
    name: "simpleJSONExtractFloat",
    signature: "simpleJSONExtractFloat(${1:json}, ${2:field_name})",
  },
  {
    name: "simpleJSONExtractBool",
    signature: "simpleJSONExtractBool(${1:json}, ${2:field_name})",
  },
  {
    name: "simpleJSONExtractRaw",
    signature: "simpleJSONExtractRaw(${1:json}, ${2:field_name})",
  },
  {
    name: "simpleJSONExtractString",
    signature: "simpleJSONExtractString(${1:json}, ${2:field_name})",
  },

  {
    name: "JSONExtract",
    signature: "JSONExtract(${1:json}, ${2:type}, ${3:indices_or_keys})",
  },
  {
    name: "JSONExtractUInt",
    signature: "JSONExtractUInt(${1:json}, ${2:indices_or_keys})",
  },
  {
    name: "JSONExtractInt",
    signature: "JSONExtractInt(${1:json}, ${2:indices_or_keys})",
  },
  {
    name: "JSONExtractFloat",
    signature: "JSONExtractFloat(${1:json}, ${2:indices_or_keys})",
  },
  {
    name: "JSONExtractBool",
    signature: "JSONExtractBool(${1:json}, ${2:indices_or_keys})",
  },
  {
    name: "JSONExtractRaw",
    signature: "JSONExtractRaw(${1:json}, ${2:indices_or_keys})",
  },
  {
    name: "JSONExtractString",
    signature: "JSONExtractString(${1:json}, ${2:indices_or_keys})",
  },

  {
    name: "JSONExtractKeysAndValues",
    signature:
      "JSONExtractKeysAndValues(${1:json}, ${2:value_type}, ${3:indices_or_keys})",
  },
  {
    name: "JSONExtractKeys",
    signature: "JSONExtractKeys(${1:json}, ${2:indices_or_keys})",
  },
  {
    name: "JSONExtractArrayRaw",
    signature: "JSONExtractArrayRaw(${1:json}, ${2:indices_or_keys})",
  },
  {
    name: "JSONExtractKeysAndValuesRaw",
    signature: "JSONExtractKeysAndValuesRaw(${1:json}, ${2:indices_or_keys})",
  },

  { name: "isValidJSON", signature: "isValidJSON(${1:json})" },
  { name: "JSONHas", signature: "JSONHas(${1:json}, ${2:path})" },
  {
    name: "JSONLength",
    signature: "JSONLength(${1:json}, ${2:indices_or_keys})",
  },
  { name: "JSONType", signature: "JSONType(${1:json}, ${2:path})" },

  { name: "toJSONString", signature: "toJSONString(${1:value})" },
];
