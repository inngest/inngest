import assert from 'node:assert';
import { describe, it } from 'vitest';

import { parseConnectionString } from './utils';

describe('parseConnectionString', (t) => {
  it('test basic connection string', () => {
    const connectionString = 'postgres://user:password@host/db';
    const parsed = parseConnectionString('supabase', connectionString);

    assert.notEqual(parsed, null);
    assert.deepStrictEqual(parsed, {
      name: 'supabase-host',
      engine: 'postgresql',
      adminConn: connectionString,
    });
  });

  it('test invalid connection string', () => {
    const connectionString = 'https://user:password@host.com/db';
    const parsed = parseConnectionString('Neon', connectionString);

    assert.equal(parsed, null);
  });

  it('test alternate protocol and domains', () => {
    const connectionString = 'postgresql://user:password@host.com/db';
    const parsed = parseConnectionString('Neon', connectionString);

    assert.notEqual(parsed, null);
    assert.deepStrictEqual(parsed, {
      name: 'Neon-host.com',
      engine: 'postgresql',
      adminConn: connectionString,
    });
  });

  it('test user name with delimiters', () => {
    const connectionString = 'postgres://user-name_underscore:password@host.with.domain/db';
    const parsed = parseConnectionString('Neon', connectionString);

    assert.notEqual(parsed, null);
    assert.deepStrictEqual(parsed, {
      name: 'Neon-host.with.domain',
      engine: 'postgresql',
      adminConn: connectionString,
    });
  });

  it('test neon connection string', () => {
    const connectionString =
      'postgresql://my-database_owner:038hvrd1d@ep-raspy-dust-a5l80fd3.us-east-2.aws.neon.tech/my-database?sslmode=require';
    const parsed = parseConnectionString('Neon', connectionString);

    assert.notEqual(parsed, null);
    assert.deepStrictEqual(parsed, {
      name: 'Neon-ep-raspy-dust-a5l80fd3.us-east-2.aws.neon.tech',
      engine: 'postgresql',
      adminConn: connectionString,
    });
  });
});
