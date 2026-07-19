/**
 * Builds the executable, auto-mocked schema for the demo. The SDL is the
 * committed src/gql/schema.graphql artifact emitted by graphql-codegen (see
 * graphql.config.ts), so it tracks the real App API schema. `addMocksToSchema`
 * auto-mocks every type from deterministic scalar generators; buildResolvers()
 * overrides the bootstrap + hero fields with curated data. New schema fields
 * render an auto-mock value instead of breaking.
 */
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

import { addMocksToSchema } from '@graphql-tools/mock';
import { makeExecutableSchema } from '@graphql-tools/schema';
import { type GraphQLSchema, isScalarType } from 'graphql';

import { buildResolvers } from './resolvers';
import { fallbackScalarMock, scalarMocks } from './scalars';

const BUILTIN_SCALARS = new Set(['ID', 'String', 'Int', 'Float', 'Boolean']);

// The SDL lives next to the generated client types.
const SDL_URL = new URL('../../gql/schema.graphql', import.meta.url);

let cached: GraphQLSchema | undefined;

export function getMockSchema(): GraphQLSchema {
  if (cached) return cached;

  let typeDefs: string;
  try {
    typeDefs = readFileSync(fileURLToPath(SDL_URL), 'utf8');
  } catch (err) {
    throw new Error(
      `demo mock server: missing src/gql/schema.graphql. Run \`pnpm graphql-codegen\` against a reachable App API to emit it. (${String(
        err,
      )})`,
    );
  }

  const base = makeExecutableSchema({ typeDefs });

  // Guarantee a mock for every custom scalar the SDL declares. Without this,
  // any scalar we haven't listed (e.g. a newly-added one) throws "No mock
  // defined for type ..." the first time a field of that type is auto-mocked.
  // The explicit scalarMocks win; the rest get a harmless generic value so the
  // demo degrades gracefully as the schema evolves.
  const mocks: Record<string, () => unknown> = { ...scalarMocks };
  for (const type of Object.values(base.getTypeMap())) {
    if (
      isScalarType(type) &&
      !BUILTIN_SCALARS.has(type.name) &&
      !(type.name in mocks)
    ) {
      mocks[type.name] = fallbackScalarMock;
    }
  }

  cached = addMocksToSchema({
    schema: base,
    mocks,
    resolvers: buildResolvers,
    // Our resolvers win; auto-mocks fill everything else.
    preserveResolvers: true,
  });

  return cached;
}
