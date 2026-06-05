export const EDIT_GENERATION_KEY = "masterdatamate.editGenerationId";

export function readJsonStorage(key, fallback) {
  try {
    const value = window.localStorage.getItem(key);
    return value ? JSON.parse(value) : fallback;
  } catch {
    return fallback;
  }
}

export function writeJsonStorage(key, value) {
  window.localStorage.setItem(key, JSON.stringify(value));
}
