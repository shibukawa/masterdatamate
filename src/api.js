export async function api(path, options = {}) {
  const isFormData = options.body instanceof FormData;
  const response = await fetch(path, {
    ...options,
    headers: isFormData
      ? { ...(options.headers ?? {}) }
      : { "Content-Type": "application/json", ...(options.headers ?? {}) }
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : await response.text();
  if (!response.ok && response.status !== 409) throw new Error(payload.error ?? "API request failed");
  return { status: response.status, payload };
}

export function binaryAssetUrl(table, key) {
  const encodedKey = typeof key === "object" ? JSON.stringify(key) : String(key);
  return `/api/binaries/${encodeURIComponent(table)}/${encodeURIComponent(encodedKey)}`;
}

export async function uploadBinaryAsset({ table, key, field, generationId, file }) {
  const formData = new FormData();
  formData.append("file", file);
  if (field) formData.append("field", field);
  if (generationId) formData.append("generationId", generationId);
  const { payload } = await api(binaryAssetUrl(table, key), {
    method: "POST",
    body: formData
  });
  return payload;
}

export async function deleteBinaryAsset({ table, key }) {
  const { payload } = await api(binaryAssetUrl(table, key), { method: "DELETE" });
  return payload;
}

export function editorPluginAssetUrl(pluginId, assetPath = "index.html") {
  return `/api/editor-plugins/${encodeURIComponent(pluginId)}/assets/${assetPath.split("/").map(encodeURIComponent).join("/")}`;
}
