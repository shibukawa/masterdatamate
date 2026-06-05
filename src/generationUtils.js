import { EDIT_GENERATION_KEY, readJsonStorage } from "./storage.js";

export function generationSortValue(generation) {
  return typeof generation.generation_index === "number"
    ? generation.generation_index
    : Date.parse(`${generation.generation_index}T00:00:00Z`);
}

export function sortGenerations(generations) {
  return [...generations].sort((a, b) => generationSortValue(a) - generationSortValue(b) || a.id.localeCompare(b.id));
}

export function deriveGenerationId(config, settings) {
  const prefix = settings.ordering_mode === "release_date"
    ? String(config.generation_index)
    : String(config.generation_index).padStart(settings.numeric_digits ?? 4, "0");
  return `${prefix}_${config.path_name}`;
}

export function displayGenerationName(generation, settings) {
  const prefix = settings.ordering_mode === "release_date"
    ? String(generation.generation_index)
    : String(generation.generation_index).padStart(settings.numeric_digits ?? 4, "0");
  return `(${prefix}) ${generation.path_name}`;
}

export function nextEditGeneration(generations, previousId) {
  if (generations.some((generation) => generation.id === previousId)) return previousId;
  const selected = readJsonStorage(EDIT_GENERATION_KEY, "");
  if (generations.some((generation) => generation.id === selected)) return selected;
  return generations[generations.length - 1]?.id ?? "";
}

export function defaultOutputGenerationIds(generations) {
  return generations.filter((generation) => generation.output).map((generation) => generation.id);
}
