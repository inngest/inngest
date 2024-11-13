const regex = /postgresq?l?:\/\/([\w-]+):([^@]+)@([^/]+)/;

export function parseConnectionString(integration: string, connectionString: string) {
  const match = connectionString.match(regex);

  if (match) {
    const [, username, password, host] = match;
    return {
      name: `${integration}-${host}`,
      engine: 'postgresql',
      adminConn: connectionString,
    };
  }

  return null;
}
