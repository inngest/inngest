const regex = /postgresq?l?:\/\/([\w-]+):([^@]+)@([^/]+)/;

export function parseConnectionString(connectionString: string) {
  const match = connectionString.match(regex);

  if (match) {
    const [, username, password, host] = match;
    return {
      name: `Neon-${host}`,
      engine: 'postgresql',
      adminConn: connectionString,
    };
  }

  return null;
}
