export async function api(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: { "Content-Type": "application/json", ...(options.headers ?? {}) }
  });
  const payload = await response.json();
  if (!response.ok && response.status !== 409) throw new Error(payload.error ?? "API request failed");
  return { status: response.status, payload };
}
