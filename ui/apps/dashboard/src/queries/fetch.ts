export function fetchWithTimeout(timeout: number) {
  return (url: RequestInfo | URL, opts?: RequestInit) => {
    const controller = new AbortController();
    const id = setTimeout(() => controller.abort(), timeout);

    return fetch(url, {
      ...opts,
      signal: controller.signal,
    }).finally(() => {
      clearTimeout(id);
    });
  };
}
