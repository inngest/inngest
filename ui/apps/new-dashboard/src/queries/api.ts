export async function get(pathname: string): Promise<Response> {
  const url = new URL(pathname, import.meta.env.VITE_API_URL);
  return await fetch(url, {
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
  });
}

export async function put(pathname: string, body: any): Promise<Response> {
  const url = new URL(pathname, import.meta.env.VITE_API_URL);
  return await fetch(url, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
    credentials: "include",
  });
}
