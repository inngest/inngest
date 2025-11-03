import type { JSONSchema, JSONSchemaDefinition, SchemaNode } from '../types';

export function buildObjectChildren(
  schema: JSONSchema,
  path: string,
  buildNode: (schema: JSONSchema, name: string, path: string) => SchemaNode
): SchemaNode[] {
  const properties = schema.properties ?? {};

  const children: SchemaNode[] = Object.entries(properties)
    .map(([propName, propDef]) =>
      buildNodeFromDef(propDef, propName, `${path}.${propName}`, buildNode)
    )
    .filter((node) => node !== undefined);

  // TODO: Handle processing of additional properties if this is used for authored JSON Schemas.
  // For now, it's unnecessary because schemas are derived from concrete instances.

  return children;
}

function buildNodeFromDef(
  def: JSONSchema | JSONSchemaDefinition,
  name: string,
  path: string,
  buildNode: (schema: JSONSchema, name: string, path: string) => SchemaNode
): SchemaNode | undefined {
  // TODO: Handle processing of boolean schemas.
  // For now, it's unnecessary because schemas are derived from concrete instances.
  if (typeof def === 'boolean') return undefined;

  return buildNode(def, name, path);
}
