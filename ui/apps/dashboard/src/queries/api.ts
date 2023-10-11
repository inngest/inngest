export async function get(pathname: string): Promise<Response> {
  const url = new URL(pathname, process.env.NEXT_PUBLIC_API_URL);
  return await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
  });
}

export async function put(pathname: string, body: any): Promise<Response> {
  const url = new URL(pathname, process.env.NEXT_PUBLIC_API_URL);
  return await fetch(url, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
    credentials: 'include',
  });
}
