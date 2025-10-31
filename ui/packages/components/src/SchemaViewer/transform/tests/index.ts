import { ARRAY_OF_OBJECTS_CASE } from './cases/arrayOfObjects';
import { ARRAY_SCALAR_CASE } from './cases/arrayScalar';
import { ARRAY_SCALAR_UNION_CASE } from './cases/arrayScalarUnion';
import { ARRAY_TUPLE_CASE } from './cases/arrayTuple';
import { ARRAY_TUPLE_WITH_BOOLEAN_UNKNOWN_CASE } from './cases/arrayTupleWithBooleanUnknown';
import { ARRAY_UNION_NULL_CASE } from './cases/arrayUnionNull';
import { ARRAY_UNKNOWN_ITEMS_BOOLEAN_CASE } from './cases/arrayUnknownItemsBoolean';
import { ARRAY_UNKNOWN_ITEMS_UNDEFINED_CASE } from './cases/arrayUnknownItemsUndefined';
import { BASIC_EVENT_CASE } from './cases/basicEvent';
import { MIXED_STRUCTURAL_UNION_CASE } from './cases/mixedStructuralUnion';
import { OBJECT_BOOLEAN_PROPERTY_UNKNOWN_CASE } from './cases/objectBooleanPropertyUnknown';
import { OBJECT_NESTING_CASE } from './cases/objectNesting';
import { OBJECT_SCALAR_UNION_CASE } from './cases/objectScalarUnion';
import { OBJECT_UNION_NULL_CASE } from './cases/objectUnionNull';
import { ONE_OF_OBJECT_CASE } from './cases/oneOfObject';
import { ROOT_ARRAY_CASE } from './cases/rootArray';
import { UNION_SCALAR_CASE } from './cases/unionScalar';
import { UNKNOWN_VALUE_CASE } from './cases/unknownValue';
import type { TransformCase } from './types';

export const TRANSFORM_TEST_CASES: TransformCase[] = [
  BASIC_EVENT_CASE,
  OBJECT_NESTING_CASE,
  UNION_SCALAR_CASE,
  ARRAY_SCALAR_CASE,
  ARRAY_TUPLE_CASE,
  ARRAY_TUPLE_WITH_BOOLEAN_UNKNOWN_CASE,
  ARRAY_OF_OBJECTS_CASE,
  OBJECT_UNION_NULL_CASE,
  ARRAY_UNION_NULL_CASE,
  ARRAY_UNKNOWN_ITEMS_UNDEFINED_CASE,
  ARRAY_UNKNOWN_ITEMS_BOOLEAN_CASE,
  OBJECT_SCALAR_UNION_CASE,
  ARRAY_SCALAR_UNION_CASE,
  MIXED_STRUCTURAL_UNION_CASE,
  OBJECT_BOOLEAN_PROPERTY_UNKNOWN_CASE,
  ONE_OF_OBJECT_CASE,
  ROOT_ARRAY_CASE,
  UNKNOWN_VALUE_CASE,
];
