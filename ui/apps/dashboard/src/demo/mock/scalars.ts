/**
 * Deterministic generators for the custom scalars used by the App API schema
 * (enumerated in graphql.config.ts) plus id helpers used by the resolvers. All
 * derived from seeds so output is stable across requests.
 */
import { hashString, mulberry32, Rng } from './prng';

const HEX = '0123456789abcdef';
const CROCKFORD = '0123456789ABCDEFGHJKMNPQRSTVWXYZ';

/** Deterministic RFC-4122-shaped UUID (v4 layout) from a seed. */
export function seededUUID(seed: string): string {
  const rand = mulberry32(hashString('uuid:' + seed));
  const bytes: number[] = [];
  for (let i = 0; i < 16; i++) bytes.push(Math.floor(rand() * 256));
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = bytes.map((b) => HEX[b >> 4] + HEX[b & 0x0f]).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(
    12,
    16,
  )}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

/**
 * Deterministic ULID from a seed and a timestamp (ms). 26 Crockford-base32
 * chars: 10 for the timestamp, 16 for the seeded "randomness".
 */
export function seededULID(seed: string, timeMs: number): string {
  let ts = Math.floor(timeMs);
  let time = '';
  for (let i = 0; i < 10; i++) {
    time = CROCKFORD[ts % 32] + time;
    ts = Math.floor(ts / 32);
  }
  const rand = mulberry32(hashString('ulid:' + seed));
  let rest = '';
  for (let i = 0; i < 16; i++) rest += CROCKFORD[Math.floor(rand() * 32)];
  return time + rest;
}

/** Base64-ish deterministic bytes payload (schema `Bytes` maps to string). */
export function seededBytes(seed: string, obj: unknown): string {
  return Buffer.from(JSON.stringify(obj ?? { seed })).toString('base64');
}

/**
 * Scalar defaults for addMocksToSchema. Only used to fill auto-mocked
 * long-tail fields; resolvers supply concrete values for everything on the
 * hero screens. A monotonically-advancing counter keeps output deterministic
 * for a given query shape.
 */
let counter = 0;
export function resetScalarCounter() {
  counter = 0;
}
const nextSeed = () => `scalar:${counter++}`;

export const scalarMocks = {
  ID: () => seededUUID(nextSeed()),
  UUID: () => seededUUID(nextSeed()),
  ULID: () => seededULID(nextSeed(), Date.now()),
  Time: () => new Date().toISOString(),
  NullTime: () => null,
  NullString: () => 'demo',
  IP: () => '203.0.113.10',
  Bytes: () => Buffer.from('{}').toString('base64'),
  Int64: () => new Rng(nextSeed()).int(1, 1_000_000),
  // Positive by default: the graphql-tools built-in Int mock can be negative,
  // which corrupts derived values (e.g. an entitlement limit that becomes a
  // "-42d" duration string, or "-8 seats"). Keep auto-mocked ints sane.
  Int: () => new Rng(nextSeed()).int(1, 200),
  Float: () => new Rng(nextSeed()).float(1, 200),
  JSON: () => ({}),
  Map: () => ({}),
  DSN: () => 'demo-dsn',
  Period: () => 'day',
  Role: () => 'admin',
  Runtime: () => 'node',
  Timerange: () => ({}),
  Upload: () => null,
  Unknown: () => null,
  Boolean: () => true,
  // Header maps and other object-shaped scalars (mappings mirror
  // graphql.config.ts). Empty containers/strings render harmlessly.
  HTTPHeaders: () => ({}),
  SpanMetadataValues: () => ({}),
  SearchObject: () => ({}),
  SchemaSource: () => ({}),
  // String-like scalars.
  BillingPeriod: () => 'month',
  EdgeType: () => 'default',
  FilterType: () => 'string',
  IngestSource: () => 'sdk',
  SegmentType: () => 'string',
  SpanMetadataKind: () => 'string',
  SpanMetadataScope: () => 'string',
  InsightsDiagnosticCode: () => 'UNKNOWN',
  InsightsDiagnosticSeverity: () => 'INFO',
} as const;

/** Generic fallback for any scalar not explicitly listed above. */
export const fallbackScalarMock = () => 'demo';
