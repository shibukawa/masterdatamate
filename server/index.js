import { serve } from "@hono/node-server";
import { Hono } from "hono";
import { serveStatic } from "@hono/node-server/serve-static";
import { cp, readFile, writeFile, mkdir, readdir, rename, rm, realpath } from "node:fs/promises";
import { existsSync } from "node:fs";
import path from "node:path";
import YAML from "yaml";

const ROOT = process.cwd();
const SCHEMA_ROOT = path.join(ROOT, "masterdata", "schema");
const GENERATION_ROOT = path.join(ROOT, "masterdata", "generations");
const GENERATION_SETTINGS_FILE = path.join(GENERATION_ROOT, "_config.yaml");
const DEFAULT_GENERATION = "0000_initial";
const EXPORT_FORMATS = new Set(["csv_zip", "json_zip", "yaml_zip", "sql", "xlsx", "ndjson_zip", "sqlite"]);
const IMPLEMENTED_EXPORT_FORMATS = new Set(["csv_zip", "json_zip", "yaml_zip", "sql", "xlsx", "ndjson_zip"]);
const EXPORT_ARTIFACTS = new Map();

const app = new Hono();

function yamlPath(...parts) {
  return path.join(...parts);
}

function generationPath(generationId) {
  return yamlPath(GENERATION_ROOT, generationId);
}

async function assertGenerationPathInsideRoot(generationId) {
  const root = await realpath(GENERATION_ROOT);
  const targetPath = generationPath(generationId);
  const parent = existsSync(targetPath) ? await realpath(targetPath) : path.resolve(GENERATION_ROOT, generationId);
  const relative = path.relative(root, parent);
  if (relative === "" || relative.startsWith("..") || path.isAbsolute(relative)) {
    throw httpError(422, `Generation path is outside generation root: ${generationId}`);
  }
  return targetPath;
}

async function readYaml(filePath, fallback = null) {
  if (!existsSync(filePath)) return fallback;
  const text = await readFile(filePath, "utf8");
  return YAML.parse(text) ?? fallback;
}

async function writeYaml(filePath, value) {
  await mkdir(path.dirname(filePath), { recursive: true });
  const doc = new YAML.Document(value);
  doc.contents.spaceBefore = false;
  await writeFile(filePath, String(doc), "utf8");
}

function tableIdFromFile(file) {
  return path.basename(file, ".yaml");
}

async function loadSchemas() {
  const files = existsSync(SCHEMA_ROOT) ? await readdir(SCHEMA_ROOT) : [];
  const schemas = [];
  for (const file of files.filter((name) => name.endsWith(".yaml"))) {
    const table = tableIdFromFile(file);
    const schema = await readYaml(yamlPath(SCHEMA_ROOT, file));
    schemas.push(normalizeSchema(table, schema));
  }
  return schemas.sort((a, b) => a.table_id.localeCompare(b.table_id));
}

async function loadSchema(table) {
  const schema = await readYaml(yamlPath(SCHEMA_ROOT, `${table}.yaml`));
  if (!schema) throw new Error(`Schema not found: ${table}`);
  return normalizeSchema(table, schema);
}

function normalizeSchema(table, schema) {
  return {
    table_id: schema.table_id ?? schema.system_name ?? table,
    system_name: schema.system_name ?? table,
    business_name: schema.business_name ?? table,
    primary_key: schema.primary_key ?? [],
    export: schema.export ?? true,
    dependent_tables: schema.dependent_tables ?? [],
    comment: schema.comment ?? "",
    fields: schema.fields ?? []
  };
}

function schemaFile(table) {
  return yamlPath(SCHEMA_ROOT, `${table}.yaml`);
}

function schemaListRow(schema) {
  const references = [...new Set((schema.fields ?? [])
    .map((field) => field.reference?.table)
    .filter(Boolean))];
  return {
    selected: false,
    table_id: schema.table_id,
    system_name: schema.system_name,
    business_name: schema.business_name,
    export: Boolean(schema.export),
    primary_key: (schema.primary_key ?? []).join(", "),
    references: references.join(", "),
    comment: schema.comment ?? ""
  };
}

function fieldKind(field, schema) {
  if ((schema.primary_key ?? []).includes(field.system_name)) return "primary_key";
  if (field.formula) return "formula";
  if (field.reference?.table || field.type === "external_reference") return "reference";
  return "data";
}

function schemaFieldRows(schema) {
  return (schema.fields ?? []).map((field) => ({
    id: field.system_name,
    original_system_name: field.system_name,
    kind: fieldKind(field, schema),
    system_name: field.system_name,
    business_name: field.business_name ?? field.system_name,
    type: field.reference?.table ? "external_reference" : (field.type ?? "string"),
    formula: field.formula ?? "",
    reference_table: field.reference?.table ?? "",
    constants: Array.isArray(field.constants) ? field.constants.join(", ") : "",
    default_value: field.default_value ?? "",
    export: field.export !== false,
    required: Boolean(field.required),
    comment: field.comment ?? ""
  }));
}

function parseConstants(value) {
  if (Array.isArray(value)) return value.map(String).filter(Boolean);
  if (typeof value !== "string") return [];
  return value.split(",").map((item) => item.trim()).filter(Boolean);
}

function parseDefaultValue(value, type) {
  if (value === undefined || value === null || value === "") return undefined;
  if (type === "boolean") return value === true || String(value).toLowerCase() === "true";
  if (type === "integer") return Number.parseInt(value, 10);
  if (type === "decimal") return Number(value);
  return value;
}

function fieldRowsToSchema(baseSchema, rows) {
  const normalizedRows = (rows ?? []).filter((row) => row.system_name && row.kind !== "formula");
  const primaryKey = normalizedRows.filter((row) => row.kind === "primary_key").map((row) => row.system_name);
  const fields = normalizedRows.map((row) => {
    const type = row.kind === "reference" ? "external_reference" : (row.type || "string");
    const field = {
      system_name: row.system_name,
      business_name: row.business_name || row.system_name,
      type,
      required: Boolean(row.required),
      export: row.export !== false
    };
    const defaultValue = parseDefaultValue(row.default_value, type);
    if (defaultValue !== undefined) field.default_value = defaultValue;
    if (row.comment) field.comment = row.comment;
    if (type === "constant") field.constants = parseConstants(row.constants);
    if (row.kind === "reference" && row.reference_table) field.reference = { table: row.reference_table };
    return field;
  });
  return {
    system_name: baseSchema.system_name,
    business_name: baseSchema.business_name,
    primary_key: primaryKey,
    export: baseSchema.export !== false,
    dependent_tables: baseSchema.dependent_tables ?? [],
    comment: baseSchema.comment ?? "",
    fields
  };
}

function validateSchemaDraft(schema, allSchemas = []) {
  const diagnostics = [];
  if (!schema.system_name || !/^[A-Za-z0-9_][A-Za-z0-9_-]*$/.test(schema.system_name)) {
    diagnostics.push({ severity: "error", field: "system_name", message: "system_name is required and must contain only letters, numbers, underscore, and hyphen." });
  }
  const fields = schema.fields ?? [];
  const names = new Map();
  for (const [index, field] of fields.entries()) {
    if (!field.system_name) diagnostics.push({ severity: "error", rowIndex: index, field: "system_name", message: "Field system_name is required." });
    if (["key", "data", "children"].includes(field.system_name)) {
      diagnostics.push({ severity: "error", rowIndex: index, field: "system_name", message: `${field.system_name} is reserved.` });
    }
    names.set(field.system_name, (names.get(field.system_name) ?? 0) + 1);
    if (field.type === "constant" && (!Array.isArray(field.constants) || field.constants.length === 0)) {
      diagnostics.push({ severity: "error", rowIndex: index, field: "constants", message: "Constant fields require constants." });
    }
    if (field.type === "external_reference" && !field.reference?.table) {
      diagnostics.push({ severity: "error", rowIndex: index, field: "reference_table", message: "Reference fields require reference_table." });
    }
    if (field.reference?.table && allSchemas.length && !allSchemas.some((item) => item.table_id === field.reference.table)) {
      diagnostics.push({ severity: "error", rowIndex: index, field: "reference_table", message: `Referenced table does not exist: ${field.reference.table}.` });
    }
  }
  for (const [name, count] of names.entries()) {
    if (count > 1) diagnostics.push({ severity: "error", field: "system_name", message: `Field system_name is duplicated: ${name}.` });
  }
  if (!Array.isArray(schema.primary_key) || schema.primary_key.length === 0) {
    diagnostics.push({ severity: "error", field: "primary_key", message: "At least one primary key field is required." });
  }
  for (const key of schema.primary_key ?? []) {
    if (!fields.some((field) => field.system_name === key)) {
      diagnostics.push({ severity: "error", field: "primary_key", message: `Primary key field is missing: ${key}.` });
    }
  }
  return diagnostics;
}

function schemaBlockingDiagnostics(diagnostics) {
  return diagnostics.filter((diagnostic) => diagnostic.severity !== "warning");
}

async function requireGeneration(generationId) {
  const configPath = yamlPath(GENERATION_ROOT, generationId, "_config.yaml");
  const config = await readYaml(configPath);
  if (!config || typeof config !== "object") {
    const error = new Error(`Generation config is missing or invalid: ${generationId}`);
    error.status = 422;
    throw error;
  }
  return config;
}

function findGeneration(generations, generationId) {
  return generations.find((generation) => generation.id === generationId);
}

function requireUniqueIdList(value, fieldName, minLength = 1) {
  if (!Array.isArray(value) || value.length < minLength || value.some((id) => typeof id !== "string" || !id.trim())) {
    throw httpError(400, `${fieldName} must contain at least ${minLength} generation id(s).`);
  }
  const ids = value.map((id) => id.trim());
  if (new Set(ids).size !== ids.length) throw httpError(400, `${fieldName} contains duplicate generation ids.`);
  return ids;
}

async function loadGenerationSettings() {
  const settings = await readYaml(GENERATION_SETTINGS_FILE, {});
  return {
    ordering_mode: settings?.ordering_mode === "release_date" ? "release_date" : "numeric",
    numeric_digits: Number.isInteger(Number(settings?.numeric_digits)) ? Number(settings.numeric_digits) : 4
  };
}

function httpError(status, message) {
  const error = new Error(message);
  error.status = status;
  return error;
}

function validatePathName(pathName) {
  return typeof pathName === "string" && /^[a-zA-Z0-9][a-zA-Z0-9_-]*$/.test(pathName);
}

function normalizeGenerationConfig(config, settings) {
  const input = config ?? {};
  const output = typeof input.output === "boolean" ? input.output : true;
  const pathName = typeof input.path_name === "string" ? input.path_name.trim() : "";
  const description = typeof input.description === "string" ? input.description : "";
  if (!validatePathName(pathName)) throw httpError(422, "Generation path_name must start with an alphanumeric character and contain only letters, numbers, underscores, and hyphens.");

  if (settings.ordering_mode === "release_date") {
    const value = String(input.generation_index ?? "").trim();
    if (!/^\d{4}-\d{2}-\d{2}$/.test(value) || Number.isNaN(Date.parse(`${value}T00:00:00Z`))) {
      throw httpError(422, "Generation generation_index must be a YYYY-MM-DD date in release_date mode.");
    }
    return { generation_index: value, output, path_name: pathName, description };
  }

  const value = Number(input.generation_index);
  if (!Number.isFinite(value) || !Number.isInteger(value) || value < 0) {
    throw httpError(422, "Generation generation_index must be a non-negative integer in numeric mode.");
  }
  return { generation_index: value, output, path_name: pathName, description };
}

function generationFolderName(config, settings) {
  const prefix = settings.ordering_mode === "release_date"
    ? String(config.generation_index)
    : String(config.generation_index).padStart(settings.numeric_digits, "0");
  return `${prefix}_${config.path_name}`;
}

function generationSortValue(generation) {
  return typeof generation.generation_index === "number"
    ? generation.generation_index
    : Date.parse(`${generation.generation_index}T00:00:00Z`);
}

function sortGenerations(generations) {
  return generations.sort((a, b) => {
    const value = generationSortValue(a) - generationSortValue(b);
    return value || a.id.localeCompare(b.id);
  });
}

async function loadGenerations() {
  const settings = await loadGenerationSettings();
  const entries = existsSync(GENERATION_ROOT) ? await readdir(GENERATION_ROOT, { withFileTypes: true }) : [];
  const generations = [];
  for (const entry of entries.filter((item) => item.isDirectory())) {
    const id = entry.name;
    const configPath = yamlPath(GENERATION_ROOT, id, "_config.yaml");
    const rawConfig = await readYaml(configPath);
    if (!rawConfig || typeof rawConfig !== "object") throw httpError(422, `Generation config is missing or invalid: ${id}`);
    const config = normalizeGenerationConfig(rawConfig, settings);
    generations.push({ id, folder_name: id, derived_folder_name: generationFolderName(config, settings), ...config });
  }
  validateGenerationSet(generations);
  return { settings, generations: sortGenerations(generations) };
}

function validateGenerationSet(generations) {
  const indexes = new Set();
  const folders = new Set();
  for (const generation of generations) {
    const indexKey = String(generation.generation_index);
    if (indexes.has(indexKey)) throw httpError(422, `Generation index is duplicated: ${indexKey}`);
    indexes.add(indexKey);
    if (folders.has(generation.derived_folder_name)) throw httpError(422, `Derived generation folder is duplicated: ${generation.derived_folder_name}`);
    folders.add(generation.derived_folder_name);
  }
}

function nextPathName(baseName, usedNames) {
  let candidate = baseName;
  let index = 2;
  while (usedNames.has(candidate)) {
    candidate = `${baseName}_${index}`;
    index += 1;
  }
  return candidate;
}

function nextGenerationConfig(generations, settings) {
  const usedNames = new Set(generations.map((generation) => generation.path_name));
  if (settings.ordering_mode === "release_date") {
    let date = new Date().toISOString().slice(0, 10);
    const usedDates = new Set(generations.map((generation) => String(generation.generation_index)));
    while (usedDates.has(date)) {
      const next = new Date(`${date}T00:00:00Z`);
      next.setUTCDate(next.getUTCDate() + 1);
      date = next.toISOString().slice(0, 10);
    }
    return { generation_index: date, output: true, path_name: nextPathName("new_generation", usedNames), description: "" };
  }

  const maxIndex = generations.reduce((max, generation) => Math.max(max, Number(generation.generation_index)), -10);
  return { generation_index: maxIndex + 10, output: true, path_name: nextPathName("new_generation", usedNames), description: "" };
}

async function writeGenerationConfig(generationId, config) {
  await writeYaml(yamlPath(GENERATION_ROOT, generationId, "_config.yaml"), config);
}

function assertDestinationAvailable(configInput, current) {
  const config = normalizeGenerationConfig(configInput, current.settings);
  const folderName = generationFolderName(config, current.settings);
  const nextGenerations = current.generations.concat({ id: folderName, folder_name: folderName, derived_folder_name: folderName, ...config });
  validateGenerationSet(nextGenerations);
  if (existsSync(generationPath(folderName))) throw httpError(409, `Generation folder already exists: ${folderName}`);
  return { config, folderName };
}

async function createGeneration(configInput = null) {
  const current = await loadGenerations();
  const draft = configInput ?? nextGenerationConfig(current.generations, current.settings);
  const config = normalizeGenerationConfig(draft, current.settings);
  const folderName = generationFolderName(config, current.settings);
  if (existsSync(yamlPath(GENERATION_ROOT, folderName))) throw httpError(409, `Generation folder already exists: ${folderName}`);
  await writeGenerationConfig(folderName, config);
  return loadGenerations();
}

async function updateGeneration(generationId, configInput) {
  const current = await loadGenerations();
  const existing = current.generations.find((generation) => generation.id === generationId);
  if (!existing) throw httpError(404, `Generation not found: ${generationId}`);

  const config = normalizeGenerationConfig(configInput, current.settings);
  const nextId = generationFolderName(config, current.settings);
  const nextGenerations = current.generations
    .filter((generation) => generation.id !== generationId)
    .concat({ id: nextId, folder_name: nextId, derived_folder_name: nextId, ...config });
  validateGenerationSet(nextGenerations);

  const oldPath = yamlPath(GENERATION_ROOT, generationId);
  const nextPath = yamlPath(GENERATION_ROOT, nextId);
  if (nextId !== generationId) {
    if (existsSync(nextPath)) throw httpError(409, `Generation folder already exists: ${nextId}`);
    await rename(oldPath, nextPath);
  }
  await writeGenerationConfig(nextId, config);
  return loadGenerations();
}

async function deleteGenerations(generationIds, activeGenerationId = "") {
  const ids = requireUniqueIdList(generationIds, "generationIds", 1);
  const current = await loadGenerations();
  const existingIds = new Set(current.generations.map((generation) => generation.id));
  for (const id of ids) {
    if (!existingIds.has(id)) throw httpError(404, `Generation not found: ${id}`);
  }
  if (ids.length >= current.generations.length) throw httpError(409, "At least one generation must remain.");

  const deletedPaths = [];
  for (const id of ids) {
    const targetPath = await assertGenerationPathInsideRoot(id);
    await rm(targetPath, { recursive: true, force: false });
    deletedPaths.push(path.relative(ROOT, targetPath));
  }

  const latest = await loadGenerations();
  const deletedSet = new Set(ids);
  const fallbackBase = deletedSet.has(activeGenerationId) ? activeGenerationId : "";
  let resolvedActiveGenerationId = activeGenerationId && !deletedSet.has(activeGenerationId)
    ? activeGenerationId
    : "";
  if (!resolvedActiveGenerationId) {
    const previous = current.generations.filter((generation) => !deletedSet.has(generation.id));
    const active = current.generations.find((generation) => generation.id === fallbackBase);
    if (active) {
      const activeSort = generationSortValue(active);
      const older = previous.filter((generation) => generationSortValue(generation) < activeSort).at(-1);
      resolvedActiveGenerationId = older?.id ?? previous[0]?.id ?? "";
    } else {
      resolvedActiveGenerationId = latest.generations[0]?.id ?? "";
    }
  }

  return {
    ...latest,
    deletedGenerationIds: ids,
    deletedPaths,
    remainingGenerationIds: latest.generations.map((generation) => generation.id),
    resolvedActiveGenerationId,
    diagnostics: []
  };
}

async function duplicateGeneration(sourceGenerationId, destinationInput) {
  if (typeof sourceGenerationId !== "string" || !sourceGenerationId.trim()) throw httpError(400, "sourceGenerationId is required.");
  const current = await loadGenerations();
  const source = findGeneration(current.generations, sourceGenerationId);
  if (!source) throw httpError(404, `Generation not found: ${sourceGenerationId}`);
  const { config, folderName } = assertDestinationAvailable(destinationInput, current);
  const sourcePath = await assertGenerationPathInsideRoot(source.id);
  const destinationPath = generationPath(folderName);
  await cp(sourcePath, destinationPath, { recursive: true, errorOnExist: true, force: false, verbatimSymlinks: true });
  await writeGenerationConfig(folderName, config);
  const latest = await loadGenerations();
  return {
    ...latest,
    generationId: folderName,
    folderName,
    generation: config,
    sourceGenerationId: source.id,
    copiedPaths: [path.relative(ROOT, destinationPath)],
    diagnostics: []
  };
}

function nextCopyPathName(sourcePathName, usedNames) {
  const base = `${sourcePathName}_copy`;
  let candidate = base;
  let index = 2;
  while (usedNames.has(candidate)) {
    candidate = `${base}${index}`;
    index += 1;
  }
  usedNames.add(candidate);
  return candidate;
}

function nextDuplicateIndexFactory(generations, settings) {
  if (settings.ordering_mode === "release_date") {
    let latest = generations.reduce((max, generation) => {
      const value = Date.parse(`${generation.generation_index}T00:00:00Z`);
      return Number.isFinite(value) && value > max ? value : max;
    }, Date.parse("1970-01-01T00:00:00Z"));
    return () => {
      const next = new Date(latest);
      next.setUTCDate(next.getUTCDate() + 1);
      latest = next.getTime();
      return next.toISOString().slice(0, 10);
    };
  }

  let latest = generations.reduce((max, generation) => {
    const value = Number(generation.generation_index);
    return Number.isFinite(value) ? Math.max(max, value) : max;
  }, -10);
  return () => {
    latest += 10;
    return latest;
  };
}

async function duplicateGenerations(sourceGenerationIds) {
  const ids = requireUniqueIdList(sourceGenerationIds, "sourceGenerationIds", 1);
  const current = await loadGenerations();
  const sourceGenerations = ids.map((id) => {
    const source = findGeneration(current.generations, id);
    if (!source) throw httpError(404, `Generation not found: ${id}`);
    return source;
  });
  const orderedSources = sortGenerations([...sourceGenerations]);
  const usedNames = new Set(current.generations.map((generation) => generation.path_name));
  const nextIndex = nextDuplicateIndexFactory(current.generations, current.settings);
  const planned = [];
  const planningGenerations = [...current.generations];

  for (const source of orderedSources) {
    const config = normalizeGenerationConfig({
      generation_index: nextIndex(),
      path_name: nextCopyPathName(source.path_name, usedNames),
      output: source.output,
      description: source.description ?? ""
    }, current.settings);
    const folderName = generationFolderName(config, current.settings);
    const nextGenerations = planningGenerations.concat({ id: folderName, folder_name: folderName, derived_folder_name: folderName, ...config });
    validateGenerationSet(nextGenerations);
    if (existsSync(generationPath(folderName))) throw httpError(409, `Generation folder already exists: ${folderName}`);
    planningGenerations.push({ id: folderName, folder_name: folderName, derived_folder_name: folderName, ...config });
    planned.push({ source, config, folderName });
  }

  const copiedPaths = [];
  const createdGenerationIds = [];
  try {
    for (const item of planned) {
      const sourcePath = await assertGenerationPathInsideRoot(item.source.id);
      const destinationPath = generationPath(item.folderName);
      await cp(sourcePath, destinationPath, { recursive: true, errorOnExist: true, force: false, verbatimSymlinks: true });
      await writeGenerationConfig(item.folderName, item.config);
      copiedPaths.push(path.relative(ROOT, destinationPath));
      createdGenerationIds.push(item.folderName);
    }
  } catch (error) {
    await Promise.all(createdGenerationIds.map((id) => rm(generationPath(id), { recursive: true, force: true }).catch(() => {})));
    throw error;
  }

  const latest = await loadGenerations();
  return {
    ...latest,
    generationId: createdGenerationIds[0],
    generationIds: createdGenerationIds,
    createdGenerationIds,
    sourceGenerationIds: orderedSources.map((generation) => generation.id),
    copiedPaths,
    diagnostics: []
  };
}

function recordKeyComparable(record) {
  return normalizeComparable(record?.key ?? "");
}

async function loadTableRecordsIfPresent(table, generationId) {
  await requireGeneration(generationId);
  const value = await readYaml(tableFile(table, generationId), null);
  if (!value) return [];
  return Array.isArray(value?.[table]) ? value[table] : [];
}

async function mergeGenerationRecords(sourceGenerations, schemas) {
  const tables = {};
  for (const schema of schemas) {
    const table = schema.table_id;
    const rowsByKey = new Map();
    const order = [];
    const overrodeByKey = new Map();
    for (const generation of sourceGenerations) {
      const records = await loadTableRecordsIfPresent(table, generation.id);
      for (const record of records) {
        const key = recordKeyComparable(record);
        if (!rowsByKey.has(key)) order.push(key);
        else {
          const list = overrodeByKey.get(key) ?? [];
          list.push(rowsByKey.get(key).sourceGenerationId);
          overrodeByKey.set(key, list);
        }
        rowsByKey.set(key, { record, sourceGenerationId: generation.id });
      }
    }
    const records = order.map((key) => rowsByKey.get(key)?.record).filter(Boolean);
    tables[table] = {
      records,
      recordCount: records.length,
      overriddenRecordCount: [...overrodeByKey.values()].reduce((total, list) => total + list.length, 0)
    };
  }
  return tables;
}

async function persistentMergeGenerations(sourceGenerationIds, destinationInput) {
  const ids = requireUniqueIdList(sourceGenerationIds, "sourceGenerationIds", 2);
  const current = await loadGenerations();
  const sourceGenerations = ids.map((id) => {
    const generation = findGeneration(current.generations, id);
    if (!generation) throw httpError(404, `Generation not found: ${id}`);
    return generation;
  });
  const ordered = sortGenerations([...sourceGenerations]);
  const { config, folderName } = assertDestinationAvailable(destinationInput, current);
  const schemas = await loadSchemas();
  const tables = await mergeGenerationRecords(ordered, schemas);

  const destinationPath = generationPath(folderName);
  await mkdir(destinationPath, { recursive: false });
  try {
    await writeGenerationConfig(folderName, config);
    const writtenFiles = [path.relative(ROOT, yamlPath(destinationPath, "_config.yaml"))];
    for (const [table, result] of Object.entries(tables)) {
      if (!result.records.length) continue;
      const filePath = tableFile(table, folderName);
      await writeYaml(filePath, { [table]: result.records });
      writtenFiles.push(path.relative(ROOT, filePath));
    }
    const latest = await loadGenerations();
    return {
      ...latest,
      generationId: folderName,
      folderName,
      generation: config,
      sourceGenerationIds: ids,
      orderedSourceGenerationIds: ordered.map((generation) => generation.id),
      tables: Object.fromEntries(Object.entries(tables).map(([table, result]) => [table, {
        file: path.relative(ROOT, tableFile(table, folderName)),
        recordCount: result.recordCount,
        overriddenRecordCount: result.overriddenRecordCount
      }])),
      writtenFiles,
      diagnostics: []
    };
  } catch (error) {
    await rm(destinationPath, { recursive: true, force: true }).catch(() => {});
    throw error;
  }
}

async function analyzeGenerations(generationIds, includeMergeImpact = true) {
  const ids = requireUniqueIdList(generationIds, "generationIds", 1);
  const current = await loadGenerations();
  const selected = ids.map((id) => {
    const generation = findGeneration(current.generations, id);
    if (!generation) throw httpError(404, `Generation not found: ${id}`);
    return generation;
  });
  const ordered = sortGenerations([...selected]);
  const schemas = await loadSchemas();
  const diagnostics = [];
  const generationSummaries = new Map(selected.map((generation) => [generation.id, {
    generationId: generation.id,
    folderPath: path.relative(ROOT, generationPath(generation.id)),
    output: generation.output,
    tableCount: 0,
    recordCount: 0
  }]));
  const tableSummary = {};
  let totalRecordCount = 0;
  let totalOverridden = 0;

  for (const schema of schemas) {
    const table = schema.table_id;
    const generationRecordCounts = {};
    const winningByKey = new Map();
    const overrodeByKey = new Map();
    let recordCount = 0;
    for (const generation of ordered) {
      let records = [];
      try {
        records = await loadTableRecordsIfPresent(table, generation.id);
      } catch (error) {
        diagnostics.push({ severity: "error", generationId: generation.id, table, message: error.message });
      }
      generationRecordCounts[generation.id] = records.length;
      if (records.length) {
        const summary = generationSummaries.get(generation.id);
        summary.tableCount += 1;
        summary.recordCount += records.length;
      }
      recordCount += records.length;
      const seenInGeneration = new Set();
      for (const [rowIndex, record] of records.entries()) {
        const key = recordKeyComparable(record);
        if (seenInGeneration.has(key)) {
          diagnostics.push({ severity: "error", generationId: generation.id, table, rowIndex, message: "Primary key is duplicated within this generation." });
        }
        seenInGeneration.add(key);
        if (includeMergeImpact && winningByKey.has(key)) {
          const list = overrodeByKey.get(key) ?? [];
          list.push(winningByKey.get(key));
          overrodeByKey.set(key, list);
        }
        winningByKey.set(key, generation.id);
      }
    }
    const overriddenRecordCount = [...overrodeByKey.values()].reduce((total, list) => total + list.length, 0);
    totalRecordCount += recordCount;
    totalOverridden += overriddenRecordCount;
    tableSummary[table] = { recordCount, generationRecordCounts, overriddenRecordCount };
  }

  return {
    generationIds: ids,
    orderedGenerationIds: ordered.map((generation) => generation.id),
    summary: {
      generationCount: ids.length,
      tableCount: schemas.length,
      recordCount: totalRecordCount,
      overriddenRecordCount: totalOverridden
    },
    generations: ids.map((id) => generationSummaries.get(id)),
    tables: tableSummary,
    diagnostics
  };
}

function normalizeExportFormat(format) {
  const value = String(format ?? "").trim();
  if (!value) throw httpError(400, "format is required.");
  if (!EXPORT_FORMATS.has(value)) throw httpError(400, `Unknown export format: ${value}`);
  if (!IMPLEMENTED_EXPORT_FORMATS.has(value)) throw httpError(501, `Export format is not implemented yet: ${value}`);
  return value;
}

function exportContentType(format) {
  if (format.endsWith("_zip")) return "application/zip";
  if (format === "sql") return "application/sql; charset=utf-8";
  if (format === "xlsx") return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";
  return "application/octet-stream";
}

function exportFilename(format) {
  const suffix = {
    csv_zip: "csv.zip",
    json_zip: "json.zip",
    yaml_zip: "yaml.zip",
    ndjson_zip: "ndjson.zip",
    sql: "sql",
    xlsx: "xlsx"
  }[format] ?? "dat";
  return `masterdata-export.${suffix}`;
}

function normalizeReferenceValue(value) {
  if (value && typeof value === "object" && !Array.isArray(value) && "value" in value) return value.value;
  return value;
}

function exportFields(schema) {
  const primary = new Set(schema.primary_key ?? []);
  const fields = [];
  for (const key of schema.primary_key ?? []) {
    const field = schema.fields.find((item) => item.system_name === key);
    fields.push(field ?? { system_name: key, business_name: key, type: "string", export: true });
  }
  for (const field of schema.fields ?? []) {
    if (primary.has(field.system_name)) continue;
    if (field.export === false) continue;
    fields.push(field);
  }
  return fields;
}

function exportedRow(record, schema) {
  const source = recordToRow(record, schema);
  const row = {};
  for (const field of exportFields(schema)) {
    row[field.system_name] = normalizeReferenceValue(source[field.system_name]);
  }
  return row;
}

function validateExportValue(value, field) {
  const missing = value === undefined || value === null || value === "";
  if (field.required && missing) return `${field.business_name ?? field.system_name} is required.`;
  if (missing) return "";
  if (field.type === "integer" && !Number.isInteger(Number(value))) return `${field.business_name ?? field.system_name} must be an integer.`;
  if (field.type === "decimal" && Number.isNaN(Number(value))) return `${field.business_name ?? field.system_name} must be a number.`;
  if (field.type === "boolean" && !(typeof value === "boolean" || value === "true" || value === "false")) return `${field.business_name ?? field.system_name} must be true or false.`;
  if (field.type === "constant" && field.constants && !field.constants.includes(value)) return `${field.business_name ?? field.system_name} must be one of: ${field.constants.join(", ")}.`;
  return "";
}

async function buildExportDataset(generationIds, format) {
  const normalizedFormat = normalizeExportFormat(format);
  const ids = requireUniqueIdList(generationIds, "generationIds", 1);
  const current = await loadGenerations();
  const sourceGenerations = ids.map((id) => {
    const generation = findGeneration(current.generations, id);
    if (!generation) throw httpError(404, `Generation not found: ${id}`);
    return generation;
  });
  const ordered = sortGenerations([...sourceGenerations]);
  const schemas = await loadSchemas();
  const merged = await mergeGenerationRecords(ordered, schemas);
  const exportSchemas = schemas.filter((schema) => schema.export !== false);
  const schemaByTable = new Map(schemas.map((schema) => [schema.table_id, schema]));
  const exportableKeys = new Map();
  const tables = {};
  const diagnostics = [];
  let recordCount = 0;

  for (const schema of exportSchemas) {
    const rows = (merged[schema.table_id]?.records ?? []).map((record) => exportedRow(record, schema));
    const keys = new Set();
    for (const [rowIndex, row] of rows.entries()) {
      const key = keyFromRow(row, schema);
      const keyLabel = normalizeComparable(key);
      if (keys.has(keyLabel)) {
        diagnostics.push({ severity: "error", table: schema.table_id, rowIndex, recordKey: keyLabel, field: schema.primary_key[0], message: "Primary key is duplicated in the effective export dataset." });
      }
      keys.add(keyLabel);
      for (const field of exportFields(schema)) {
        const value = normalizeReferenceValue(row[field.system_name]);
        const message = validateExportValue(value, field);
        if (message) diagnostics.push({ severity: "error", table: schema.table_id, rowIndex, recordKey: keyLabel, field: field.system_name, message });
      }
    }
    exportableKeys.set(schema.table_id, keys);
    tables[schema.table_id] = { schema, rows };
    recordCount += rows.length;
  }

  for (const schema of exportSchemas) {
    const table = tables[schema.table_id];
    for (const [rowIndex, row] of table.rows.entries()) {
      const recordKey = normalizeComparable(keyFromRow(row, schema));
      for (const field of schema.fields ?? []) {
        if (field.type !== "external_reference" || !field.reference?.table) continue;
        const target = field.reference.table;
        const targetSchema = schemaByTable.get(target);
        const value = normalizeReferenceValue(row[field.system_name]);
        if (value === undefined || value === null || value === "") continue;
        if (!targetSchema || targetSchema.export === false || !exportableKeys.get(target)?.has(normalizeComparable(value))) {
          diagnostics.push({
            severity: "error",
            table: schema.table_id,
            rowIndex,
            recordKey,
            field: field.system_name,
            message: `Referenced ${target} record ${value} is not present in the selected export generation set.`
          });
        }
      }
    }
  }

  return {
    exportable: diagnostics.every((diagnostic) => diagnostic.severity !== "error"),
    generationIds: ids,
    orderedGenerationIds: ordered.map((generation) => generation.id),
    format: normalizedFormat,
    tables,
    summary: {
      tableCount: Object.keys(tables).length,
      recordCount,
      diagnosticCount: diagnostics.length
    },
    diagnostics
  };
}

function textCell(value) {
  if (value === undefined || value === null) return "";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function csvEscape(value) {
  const text = textCell(value);
  return /[",\n\r]/.test(text) ? `"${text.replaceAll("\"", "\"\"")}"` : text;
}

function buildCsvZip(dataset) {
  const files = Object.values(dataset.tables).map(({ schema, rows }) => {
    const fields = exportFields(schema).map((field) => field.system_name);
    const lines = [fields.map(csvEscape).join(",")];
    for (const row of rows) lines.push(fields.map((field) => csvEscape(row[field])).join(","));
    return { name: `${schema.table_id}.csv`, data: Buffer.from(`${lines.join("\n")}\n`, "utf8") };
  });
  files.push({ name: "manifest.json", data: Buffer.from(JSON.stringify(exportManifest(dataset), null, 2), "utf8") });
  return createZip(files);
}

function buildJsonZip(dataset) {
  const files = Object.values(dataset.tables).map(({ schema, rows }) => ({
    name: `${schema.table_id}.json`,
    data: Buffer.from(JSON.stringify(rows, null, 2), "utf8")
  }));
  files.push({ name: "manifest.json", data: Buffer.from(JSON.stringify(exportManifest(dataset), null, 2), "utf8") });
  return createZip(files);
}

function buildNdjsonZip(dataset) {
  const files = Object.values(dataset.tables).map(({ schema, rows }) => ({
    name: `${schema.table_id}.ndjson`,
    data: Buffer.from(`${rows.map((row) => JSON.stringify(row)).join("\n")}\n`, "utf8")
  }));
  files.push({ name: "manifest.json", data: Buffer.from(JSON.stringify(exportManifest(dataset), null, 2), "utf8") });
  return createZip(files);
}

function buildYamlZip(dataset) {
  const files = Object.values(dataset.tables).map(({ schema, rows }) => ({
    name: `${schema.table_id}.yaml`,
    data: Buffer.from(String(new YAML.Document({ [schema.table_id]: rows })), "utf8")
  }));
  files.push({ name: "manifest.yaml", data: Buffer.from(String(new YAML.Document(exportManifest(dataset))), "utf8") });
  return createZip(files);
}

function exportManifest(dataset) {
  return {
    generationIds: dataset.generationIds,
    orderedGenerationIds: dataset.orderedGenerationIds,
    format: dataset.format,
    summary: dataset.summary,
    tables: Object.fromEntries(Object.entries(dataset.tables).map(([table, value]) => [table, {
      recordCount: value.rows.length,
      fields: exportFields(value.schema).map((field) => field.system_name)
    }]))
  };
}

function sqlType(field) {
  if (field.type === "integer") return "INTEGER";
  if (field.type === "decimal") return "REAL";
  if (field.type === "boolean") return "BOOLEAN";
  return "TEXT";
}

function sqlIdent(name) {
  return `"${String(name).replaceAll("\"", "\"\"")}"`;
}

function sqlLiteral(value) {
  if (value === undefined || value === null || value === "") return "NULL";
  if (typeof value === "number") return Number.isFinite(value) ? String(value) : "NULL";
  if (typeof value === "boolean") return value ? "TRUE" : "FALSE";
  return `'${textCell(value).replaceAll("'", "''")}'`;
}

function buildSql(dataset) {
  const lines = [
    "-- MasterDataMate export",
    `-- Generations: ${dataset.orderedGenerationIds.join(", ")}`,
    ""
  ];
  for (const { schema, rows } of Object.values(dataset.tables)) {
    const fields = exportFields(schema);
    const primaryKeys = schema.primary_key.map(sqlIdent).join(", ");
    lines.push(`CREATE TABLE IF NOT EXISTS ${sqlIdent(schema.table_id)} (`);
    lines.push(`  ${fields.map((field) => `${sqlIdent(field.system_name)} ${sqlType(field)}`).join(",\n  ")}${primaryKeys ? `,\n  PRIMARY KEY (${primaryKeys})` : ""}`);
    lines.push(");");
    lines.push(`TRUNCATE TABLE ${sqlIdent(schema.table_id)};`);
    for (const row of rows) {
      lines.push(`INSERT INTO ${sqlIdent(schema.table_id)} (${fields.map((field) => sqlIdent(field.system_name)).join(", ")}) VALUES (${fields.map((field) => sqlLiteral(row[field.system_name])).join(", ")});`);
    }
    lines.push("");
  }
  return Buffer.from(lines.join("\n"), "utf8");
}

function xmlEscape(value) {
  return textCell(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll("\"", "&quot;");
}

function sheetName(name, used) {
  let base = String(name).replace(/[\\/?*:[\]]/g, "_").slice(0, 31) || "Sheet";
  let candidate = base;
  let index = 2;
  while (used.has(candidate)) {
    const suffix = `_${index}`;
    candidate = `${base.slice(0, 31 - suffix.length)}${suffix}`;
    index += 1;
  }
  used.add(candidate);
  return candidate;
}

function buildSheetXml(fields, rows) {
  const rowXml = [];
  const allRows = [fields.map((field) => field.system_name), ...rows.map((row) => fields.map((field) => row[field.system_name]))];
  for (const [rowIndex, row] of allRows.entries()) {
    const cells = row.map((value, columnIndex) => {
      const ref = `${String.fromCharCode(65 + columnIndex)}${rowIndex + 1}`;
      return `<c r="${ref}" t="inlineStr"><is><t>${xmlEscape(value)}</t></is></c>`;
    }).join("");
    rowXml.push(`<row r="${rowIndex + 1}">${cells}</row>`);
  }
  return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>${rowXml.join("")}</sheetData></worksheet>`;
}

function buildXlsx(dataset) {
  const used = new Set();
  const sheets = Object.values(dataset.tables).map((table, index) => ({
    id: index + 1,
    name: sheetName(table.schema.table_id, used),
    fields: exportFields(table.schema),
    rows: table.rows
  }));
  const files = [
    {
      name: "[Content_Types].xml",
      data: Buffer.from(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
${sheets.map((sheet) => `<Override PartName="/xl/worksheets/sheet${sheet.id}.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`).join("")}
</Types>`, "utf8")
    },
    {
      name: "_rels/.rels",
      data: Buffer.from(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`, "utf8")
    },
    {
      name: "xl/workbook.xml",
      data: Buffer.from(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>${sheets.map((sheet) => `<sheet name="${xmlEscape(sheet.name)}" sheetId="${sheet.id}" r:id="rId${sheet.id}"/>`).join("")}</sheets></workbook>`, "utf8")
    },
    {
      name: "xl/_rels/workbook.xml.rels",
      data: Buffer.from(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">${sheets.map((sheet) => `<Relationship Id="rId${sheet.id}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet${sheet.id}.xml"/>`).join("")}</Relationships>`, "utf8")
    },
    ...sheets.map((sheet) => ({
      name: `xl/worksheets/sheet${sheet.id}.xml`,
      data: Buffer.from(buildSheetXml(sheet.fields, sheet.rows), "utf8")
    }))
  ];
  return createZip(files);
}

function crc32(buffer) {
  let crc = -1;
  for (const byte of buffer) {
    crc = (crc >>> 8) ^ CRC32_TABLE[(crc ^ byte) & 0xff];
  }
  return (crc ^ -1) >>> 0;
}

const CRC32_TABLE = Array.from({ length: 256 }, (_, index) => {
  let c = index;
  for (let bit = 0; bit < 8; bit += 1) c = (c & 1) ? (0xedb88320 ^ (c >>> 1)) : (c >>> 1);
  return c >>> 0;
});

function dosDateTime(date = new Date()) {
  const time = (date.getHours() << 11) | (date.getMinutes() << 5) | Math.floor(date.getSeconds() / 2);
  const dosDate = ((date.getFullYear() - 1980) << 9) | ((date.getMonth() + 1) << 5) | date.getDate();
  return { time, date: dosDate };
}

function createZip(files) {
  const localParts = [];
  const centralParts = [];
  let offset = 0;
  const stamp = dosDateTime();
  for (const file of files) {
    const name = Buffer.from(file.name, "utf8");
    const data = Buffer.isBuffer(file.data) ? file.data : Buffer.from(file.data);
    const crc = crc32(data);
    const local = Buffer.alloc(30);
    local.writeUInt32LE(0x04034b50, 0);
    local.writeUInt16LE(20, 4);
    local.writeUInt16LE(0x0800, 6);
    local.writeUInt16LE(0, 8);
    local.writeUInt16LE(stamp.time, 10);
    local.writeUInt16LE(stamp.date, 12);
    local.writeUInt32LE(crc, 14);
    local.writeUInt32LE(data.length, 18);
    local.writeUInt32LE(data.length, 22);
    local.writeUInt16LE(name.length, 26);
    local.writeUInt16LE(0, 28);
    localParts.push(local, name, data);

    const central = Buffer.alloc(46);
    central.writeUInt32LE(0x02014b50, 0);
    central.writeUInt16LE(20, 4);
    central.writeUInt16LE(20, 6);
    central.writeUInt16LE(0x0800, 8);
    central.writeUInt16LE(0, 10);
    central.writeUInt16LE(stamp.time, 12);
    central.writeUInt16LE(stamp.date, 14);
    central.writeUInt32LE(crc, 16);
    central.writeUInt32LE(data.length, 20);
    central.writeUInt32LE(data.length, 24);
    central.writeUInt16LE(name.length, 28);
    central.writeUInt16LE(0, 30);
    central.writeUInt16LE(0, 32);
    central.writeUInt16LE(0, 34);
    central.writeUInt16LE(0, 36);
    central.writeUInt32LE(0, 38);
    central.writeUInt32LE(offset, 42);
    centralParts.push(central, name);
    offset += local.length + name.length + data.length;
  }
  const centralSize = centralParts.reduce((total, part) => total + part.length, 0);
  const end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50, 0);
  end.writeUInt16LE(0, 4);
  end.writeUInt16LE(0, 6);
  end.writeUInt16LE(files.length, 8);
  end.writeUInt16LE(files.length, 10);
  end.writeUInt32LE(centralSize, 12);
  end.writeUInt32LE(offset, 16);
  end.writeUInt16LE(0, 20);
  return Buffer.concat([...localParts, ...centralParts, end]);
}

function buildExportArtifact(dataset) {
  if (dataset.format === "csv_zip") return buildCsvZip(dataset);
  if (dataset.format === "json_zip") return buildJsonZip(dataset);
  if (dataset.format === "yaml_zip") return buildYamlZip(dataset);
  if (dataset.format === "ndjson_zip") return buildNdjsonZip(dataset);
  if (dataset.format === "sql") return buildSql(dataset);
  if (dataset.format === "xlsx") return buildXlsx(dataset);
  throw httpError(501, `Export format is not implemented yet: ${dataset.format}`);
}

function tableFile(table, generationId) {
  return yamlPath(GENERATION_ROOT, generationId, `${table}.yaml`);
}

async function generationIds() {
  const entries = existsSync(GENERATION_ROOT) ? await readdir(GENERATION_ROOT, { withFileTypes: true }) : [];
  return entries.filter((entry) => entry.isDirectory()).map((entry) => entry.name);
}

async function renameTableDataFiles(oldTable, newTable) {
  const renamed = [];
  for (const generationId of await generationIds()) {
    const oldPath = tableFile(oldTable, generationId);
    const nextPath = tableFile(newTable, generationId);
    if (!existsSync(oldPath)) continue;
    if (existsSync(nextPath)) throw httpError(409, `Table data file already exists: ${path.relative(ROOT, nextPath)}`);
    await rename(oldPath, nextPath);
    renamed.push({ from: path.relative(ROOT, oldPath), to: path.relative(ROOT, nextPath) });
  }
  return renamed;
}

async function updateSchemaReferences(oldTable, newTable) {
  const schemas = await loadSchemas();
  const changed = [];
  for (const schema of schemas) {
    let dirty = false;
    for (const field of schema.fields ?? []) {
      if (field.reference?.table === oldTable) {
        field.reference.table = newTable;
        dirty = true;
      }
    }
    for (const dependent of schema.dependent_tables ?? []) {
      if (dependent.table === oldTable) {
        dependent.table = newTable;
        dirty = true;
      }
    }
    if (dirty) {
      await writeYaml(schemaFile(schema.table_id), schemaToYaml(schema));
      changed.push(schema.table_id);
    }
  }
  return changed;
}

function schemaToYaml(schema) {
  const next = {
    system_name: schema.system_name,
    business_name: schema.business_name ?? schema.system_name,
    primary_key: schema.primary_key ?? [],
    export: schema.export !== false
  };
  if (schema.dependent_tables?.length) next.dependent_tables = schema.dependent_tables;
  if (schema.comment) next.comment = schema.comment;
  next.fields = schema.fields ?? [];
  return next;
}

function materializedDefault(field) {
  if (field.default_value !== undefined) return field.default_value;
  if (field.type === "boolean") return false;
  if (field.type === "integer" || field.type === "decimal") return 0;
  if (field.required && field.type === "string") return "";
  return "";
}

function materializeRowDefaults(row, schema) {
  const next = { ...row };
  for (const field of schema.fields ?? []) {
    if (field.formula) continue;
    if (next[field.system_name] === undefined || next[field.system_name] === null || next[field.system_name] === "") {
      next[field.system_name] = materializedDefault(field);
    }
  }
  return next;
}

function renameRecordField(record, oldName, newName, schema, fieldKindValue) {
  const next = { ...record };
  if (fieldKindValue === "primary_key") {
    if (schema.primary_key.length > 1 && next.key && typeof next.key === "object" && oldName in next.key) {
      const key = { ...next.key };
      key[newName] = key[oldName];
      delete key[oldName];
      next.key = key;
    }
    return next;
  }
  const data = { ...(next.data ?? {}) };
  if (oldName in data) {
    data[newName] = data[oldName];
    delete data[oldName];
  }
  next.data = data;
  return next;
}

function deleteRecordField(record, fieldName) {
  const next = { ...record, data: { ...(record.data ?? {}) } };
  delete next.data[fieldName];
  return next;
}

async function rewriteTableRecords(table, transform) {
  const written = [];
  for (const generationId of await generationIds()) {
    const generationDir = generationPath(generationId);
    const files = existsSync(generationDir) ? await readdir(generationDir) : [];
    for (const file of files.filter((name) => name.endsWith(".yaml") && name !== "_config.yaml")) {
      const filePath = yamlPath(generationDir, file);
      const value = await readYaml(filePath, null);
      if (!value || typeof value !== "object") continue;
      const { next, changed } = rewriteRecordsInYaml(value, table, transform);
      if (!changed) continue;
      await writeYaml(filePath, next);
      written.push(path.relative(ROOT, filePath));
    }
  }
  return [...new Set(written)];
}

function rewriteRecordsInYaml(value, targetTable, transform) {
  let changed = false;
  const next = Array.isArray(value) ? [...value] : { ...value };
  for (const [key, records] of Object.entries(value)) {
    if (!Array.isArray(records)) continue;
    const rewritten = records.map((record) => {
      const rootRecord = key === targetTable ? transform(record) : { ...record };
      const { nextRecord, changed: childChanged } = rewriteEmbeddedChildren(rootRecord, targetTable, transform);
      changed = changed || key === targetTable || childChanged;
      return nextRecord;
    });
    next[key] = rewritten;
  }
  return { next, changed };
}

function rewriteEmbeddedChildren(record, targetTable, transform) {
  if (!Array.isArray(record.children)) return { nextRecord: record, changed: false };
  let changed = false;
  const children = record.children.map((child) => {
    const transformed = child.table === targetTable ? transform(child) : { ...child };
    const nested = rewriteEmbeddedChildren(transformed, targetTable, transform);
    changed = changed || child.table === targetTable || nested.changed;
    return nested.nextRecord;
  });
  return { nextRecord: { ...record, children }, changed };
}

function validateExistingValuesForSchemaChange(rows, currentRows, nextRows) {
  const diagnostics = [];
  const currentByName = new Map(currentRows.map((row) => [row.system_name, row]));
  for (const row of nextRows) {
    const previous = currentByName.get(row.original_system_name || row.system_name);
    if (!previous) continue;
    if (previous.type !== row.type) {
      diagnostics.push({
        severity: "warning",
        field: row.system_name,
        message: `Field type changes from ${previous.type} to ${row.type}; existing values will be validated under the new type when table data is saved or exported.`
      });
    }
  }
  return diagnostics;
}

async function saveSchemaDetail(table, body) {
  const current = existsSync(schemaFile(table))
    ? await loadSchema(table)
    : normalizeSchema(table, { system_name: table, business_name: table, primary_key: [], fields: [] });
  const metadata = body.schema ?? {};
  const baseSchema = {
    ...current,
    system_name: metadata.system_name ?? current.system_name,
    business_name: metadata.business_name ?? current.business_name,
    export: metadata.export ?? current.export,
    comment: metadata.comment ?? current.comment
  };
  const previousRows = schemaFieldRows(current);
  const nextSchema = fieldRowsToSchema(baseSchema, body.fields ?? previousRows);
  const allSchemas = await loadSchemas().catch(() => []);
  const nextRows = body.fields ?? previousRows;
  const diagnostics = [
    ...validateSchemaDraft(nextSchema, allSchemas.filter((schema) => schema.table_id !== table || schema.table_id === nextSchema.system_name)),
    ...validateExistingValuesForSchemaChange([], previousRows, nextRows)
  ];
  const blocking = schemaBlockingDiagnostics(diagnostics);
  if (blocking.length) return { status: 422, payload: { saved: false, diagnostics, requiresForce: false } };
  const hasWarnings = diagnostics.some((diagnostic) => diagnostic.severity === "warning");
  if (hasWarnings && !body.force) return { status: 409, payload: { saved: false, diagnostics, requiresForce: true } };

  const nextByOriginal = new Map(nextRows.filter((row) => row.original_system_name).map((row) => [row.original_system_name, row]));
  const changedFiles = [];
  for (const previous of previousRows) {
    const next = nextByOriginal.get(previous.system_name);
    if (next && next.system_name && next.system_name !== previous.system_name) {
      changedFiles.push(...await rewriteTableRecords(table, (record) => renameRecordField(record, previous.system_name, next.system_name, current, previous.kind)));
    }
  }
  const nextNames = new Set(nextRows.map((row) => row.system_name));
  for (const previous of previousRows) {
    if (!nextNames.has(previous.system_name) && previous.kind !== "primary_key") {
      changedFiles.push(...await rewriteTableRecords(table, (record) => deleteRecordField(record, previous.system_name)));
    }
  }
  await writeYaml(schemaFile(nextSchema.system_name), schemaToYaml(nextSchema));
  if (nextSchema.system_name !== table && existsSync(schemaFile(table))) await rm(schemaFile(table));
  return {
    status: 200,
    payload: {
      saved: true,
      schema: normalizeSchema(nextSchema.system_name, nextSchema),
      fieldRows: schemaFieldRows(normalizeSchema(nextSchema.system_name, nextSchema)),
      changedFiles: [...new Set(changedFiles)],
      diagnostics
    }
  };
}

async function renameSchemaFields(table, renames, force = false) {
  if (!Array.isArray(renames) || !renames.length) throw httpError(400, "renames is required.");
  const detail = await loadSchema(table);
  const rows = schemaFieldRows(detail);
  for (const renameItem of renames) {
    const from = String(renameItem.from ?? renameItem.oldName ?? "").trim();
    const to = String(renameItem.to ?? renameItem.newName ?? "").trim();
    if (!from || !to) throw httpError(400, "Each rename requires from and to.");
    const row = rows.find((item) => item.system_name === from);
    if (!row) throw httpError(404, `Field not found: ${from}`);
    row.system_name = to;
  }
  return saveSchemaDetail(table, { schema: detail, fields: rows, force });
}

async function deleteSchemaFields(table, fieldNames, force = false) {
  if (!Array.isArray(fieldNames) || !fieldNames.length) throw httpError(400, "fieldNames is required.");
  const detail = await loadSchema(table);
  const names = new Set(fieldNames.map((name) => String(name).trim()).filter(Boolean));
  const rows = schemaFieldRows(detail);
  for (const name of names) {
    if (!rows.some((row) => row.system_name === name)) throw httpError(404, `Field not found: ${name}`);
  }
  const kept = rows.filter((row) => !names.has(row.system_name));
  return saveSchemaDetail(table, { schema: detail, fields: kept, force });
}

async function loadCanonicalRecords(table, generationId) {
  await requireGeneration(generationId);
  const value = await readYaml(tableFile(table, generationId), { [table]: [] });
  return Array.isArray(value?.[table]) ? value[table] : [];
}

function keyFromRow(row, schema) {
  if (schema.primary_key.length === 1) return row[schema.primary_key[0]] ?? "";
  return Object.fromEntries(schema.primary_key.map((field) => [field, row[field] ?? ""]));
}

function keyToRow(key, schema) {
  if (schema.primary_key.length === 1) return { [schema.primary_key[0]]: key };
  return typeof key === "object" && key ? key : {};
}

function recordToRow(record, schema) {
  return materializeRowDefaults({
    ...keyToRow(record.key, schema),
    name: record.name ?? "",
    ...(record.data ?? {})
  }, schema);
}

function rowToRecord(row, schema) {
  const data = {};
  for (const field of schema.fields) {
    if (schema.primary_key.includes(field.system_name) || field.formula) continue;
    const value = row[field.system_name];
    if (value !== undefined && value !== null && value !== "") data[field.system_name] = value;
  }
  const record = { key: keyFromRow(row, schema), data };
  if (row.name) record.name = row.name;
  return record;
}

async function loadRows(table, generationId) {
  const schema = await loadSchema(table);
  const records = await loadCanonicalRecords(table, generationId);
  return { schema, records, rows: records.map((record) => recordToRow(record, schema)) };
}

function normalizeComparable(value) {
  return JSON.stringify(value ?? "");
}

function rowLabel(row, schema) {
  const key = keyFromRow(row, schema);
  return row.name || row.display_name || row.title || String(typeof key === "object" ? JSON.stringify(key) : key);
}

function generationDisplayName(generation) {
  return generation.description ? `${generation.id}: ${generation.description}` : generation.id;
}

function visibleGenerations(allGenerations, activeGenerationId, mode) {
  const active = allGenerations.find((generation) => generation.id === activeGenerationId);
  if (!active) throw httpError(404, `Generation not found: ${activeGenerationId}`);
  if (mode === "active_only") return [active];
  if (mode !== "include_previous") throw httpError(400, `Unknown generation view mode: ${mode}`);

  const activeSortValue = generationSortValue(active);
  return allGenerations.filter((generation) => {
    if (generation.id === active.id) return true;
    return generation.output && generationSortValue(generation) <= activeSortValue;
  });
}

function referenceGenerations(allGenerations, activeGenerationId, mode) {
  const active = allGenerations.find((generation) => generation.id === activeGenerationId);
  if (!active) throw httpError(404, `Generation not found: ${activeGenerationId}`);
  if (mode === "active_only") return active.output ? [active] : [];
  if (mode !== "include_previous") throw httpError(400, `Unknown generation view mode: ${mode}`);

  const activeSortValue = generationSortValue(active);
  return allGenerations.filter((generation) => generation.output && generationSortValue(generation) <= activeSortValue);
}

async function loadGenerationView(table, activeGenerationId, mode) {
  const schema = await loadSchema(table);
  const { generations } = await loadGenerations();
  const visible = visibleGenerations(generations, activeGenerationId, mode);
  const rows = [];

  for (const generation of visible) {
    const records = await loadCanonicalRecords(table, generation.id);
    records.forEach((record, rowIndex) => {
      const row = recordToRow(record, schema);
      const key = keyFromRow(row, schema);
      rows.push({
        ...row,
        sourceGenerationId: generation.id,
        sourceGenerationLabel: generationDisplayName(generation),
        isActiveGeneration: generation.id === activeGenerationId,
        isReadOnly: generation.id !== activeGenerationId,
        isOverridden: false,
        overriddenByGenerationId: "",
        _sourceRowIndex: rowIndex,
        _keyComparable: normalizeComparable(key)
      });
    });
  }

  const winningGenerationByKey = new Map();
  for (const row of rows) {
    winningGenerationByKey.set(row._keyComparable, row.sourceGenerationId);
  }

  const viewRows = rows.map(({ _sourceRowIndex, _keyComparable, ...row }) => {
    const winner = winningGenerationByKey.get(_keyComparable);
    const isOverridden = winner !== row.sourceGenerationId;
    return {
      ...row,
      isOverridden,
      overriddenByGenerationId: isOverridden ? winner : ""
    };
  });

  return {
    schema,
    activeGenerationId,
    mode,
    orderedGenerationIds: visible.map((generation) => generation.id),
    rows: viewRows,
    diagnostics: await validateRows(table, activeGenerationId, viewRows.filter((row) => row.sourceGenerationId === activeGenerationId), mode)
  };
}

async function effectiveReferenceRows(table, activeGenerationId, mode) {
  const schema = await loadSchema(table);
  const { generations } = await loadGenerations();
  const visible = referenceGenerations(generations, activeGenerationId, mode);
  const rows = [];

  for (const generation of visible) {
    const records = await loadCanonicalRecords(table, generation.id);
    records.forEach((record) => {
      const row = recordToRow(record, schema);
      rows.push({
        ...row,
        sourceGenerationId: generation.id,
        sourceGenerationLabel: generationDisplayName(generation),
        _keyComparable: normalizeComparable(keyFromRow(row, schema))
      });
    });
  }

  const winningGenerationByKey = new Map();
  for (const row of rows) {
    winningGenerationByKey.set(row._keyComparable, row.sourceGenerationId);
  }

  const rowsByKey = new Map();
  const overrodeByKey = new Map();
  for (const row of rows) {
    const winner = winningGenerationByKey.get(row._keyComparable);
    if (winner === row.sourceGenerationId) continue;
    const key = normalizeComparable(keyFromRow(row, schema));
    const list = overrodeByKey.get(key) ?? [];
    list.push(row.sourceGenerationId);
    overrodeByKey.set(key, list);
  }
  for (const { _keyComparable, ...row } of rows) {
    const winner = winningGenerationByKey.get(_keyComparable);
    if (winner !== row.sourceGenerationId) continue;
    const key = normalizeComparable(keyFromRow(row, schema));
    rowsByKey.set(key, { ...row, overrodeGenerationIds: overrodeByKey.get(key) ?? [] });
  }
  return { schema, rows: [...rowsByKey.values()] };
}

async function referenceKeys(table, generationId, mode = "active_only") {
  const { schema, rows } = mode === "active_only"
    ? await loadRows(table, generationId)
    : await effectiveReferenceRows(table, generationId, mode);
  return new Set(rows.map((row) => normalizeComparable(keyFromRow(row, schema))));
}

async function validateRows(table, generationId, rows, mode = "active_only") {
  const schema = await loadSchema(table);
  const schemas = await loadSchemas();
  const diagnostics = [];
  const seen = new Map();
  const referenceCache = new Map();

  for (const [rowIndex, row] of rows.entries()) {
    const key = keyFromRow(row, schema);
    const keyLabel = normalizeComparable(key);
    if (seen.has(keyLabel)) {
      diagnostics.push({ severity: "error", table, generationId, rowIndex, field: schema.primary_key[0], message: "Primary key is duplicated." });
    }
    seen.set(keyLabel, true);

    for (const field of schema.fields) {
      const fieldName = field.system_name;
      const value = row[fieldName];
      const missing = value === undefined || value === null || value === "";
      if (field.required && missing) {
        diagnostics.push({ severity: "error", table, generationId, rowIndex, field: fieldName, message: `${field.business_name} is required.` });
        continue;
      }
      if (missing) continue;
      if (field.type === "integer" && !Number.isInteger(Number(value))) {
        diagnostics.push({ severity: "error", table, generationId, rowIndex, field: fieldName, message: `${field.business_name} must be an integer.` });
      }
      if (field.type === "decimal" && Number.isNaN(Number(value))) {
        diagnostics.push({ severity: "error", table, generationId, rowIndex, field: fieldName, message: `${field.business_name} must be a number.` });
      }
      if (field.type === "boolean" && typeof value !== "boolean") {
        diagnostics.push({ severity: "error", table, generationId, rowIndex, field: fieldName, message: `${field.business_name} must be true or false.` });
      }
      if (field.type === "constant" && field.constants && !field.constants.includes(value)) {
        diagnostics.push({ severity: "error", table, generationId, rowIndex, field: fieldName, message: `${field.business_name} must be one of: ${field.constants.join(", ")}.` });
      }
      if (field.type === "external_reference" && field.reference?.table) {
        const target = field.reference.table;
        if (!referenceCache.has(target)) referenceCache.set(target, await referenceKeys(target, generationId, mode));
        if (!referenceCache.get(target).has(normalizeComparable(value))) {
          diagnostics.push({ severity: "error", table, generationId, rowIndex, field: fieldName, message: `${field.business_name} references missing ${target} key: ${value}.` });
        }
      }
    }
  }

  const schemaNames = new Set(schemas.map((item) => item.table_id));
  if (!schemaNames.has(table)) {
    diagnostics.push({ severity: "error", table, generationId, message: "Table schema is not registered." });
  }
  return diagnostics;
}

app.get("/api/tables", async (c) => {
  const schemas = await loadSchemas();
  return c.json({ generationId: DEFAULT_GENERATION, tables: schemas.map(({ table_id, system_name, business_name, comment }) => ({ table_id, system_name, business_name, comment })) });
});

app.get("/api/tables/:table/schema", async (c) => {
  return c.json({ schema: await loadSchema(c.req.param("table")) });
});

app.get("/api/tables/:table/generations/:generationId/records", async (c) => {
  const table = c.req.param("table");
  const generationId = c.req.param("generationId");
  const { schema, records, rows } = await loadRows(table, generationId);
  const diagnostics = await validateRows(table, generationId, rows);
  return c.json({ schema, records, rows, diagnostics });
});

app.get("/api/tables/:table/generation-view", async (c) => {
  const table = c.req.param("table");
  const activeGenerationId = c.req.query("activeGenerationId");
  const mode = c.req.query("mode") ?? "active_only";
  if (!activeGenerationId) throw httpError(400, "activeGenerationId is required.");
  return c.json(await loadGenerationView(table, activeGenerationId, mode));
});

app.post("/api/tables/:table/generations/:generationId/validate", async (c) => {
  const table = c.req.param("table");
  const generationId = c.req.param("generationId");
  const body = await c.req.json();
  return c.json({ diagnostics: await validateRows(table, generationId, body.rows ?? [], body.mode ?? "active_only") });
});

app.post("/api/tables/:table/generations/:generationId/records/commit", async (c) => {
  const table = c.req.param("table");
  const generationId = c.req.param("generationId");
  const body = await c.req.json();
  const mode = body.mode ?? "active_only";
  const schema = await loadSchema(table);
  await requireGeneration(generationId);

  let rows = body.rows;
  if (!Array.isArray(rows)) {
    const current = await loadRows(table, generationId);
    rows = current.rows;
    for (const operation of body.operations ?? []) {
      if (operation.type === "insert") rows.splice(operation.index, 0, operation.row);
      if (operation.type === "update") rows.splice(operation.index, 1, operation.row);
      if (operation.type === "delete") rows.splice(operation.index, 1);
      if (operation.type === "move") rows.splice(operation.toIndex, 0, rows.splice(operation.fromIndex, 1)[0]);
    }
  }

  rows = rows
    .filter((row) => !row.sourceGenerationId || row.sourceGenerationId === generationId)
    .map(({ sourceGenerationId, sourceGenerationLabel, isActiveGeneration, isReadOnly, isOverridden, overriddenByGenerationId, status, ...row }) => row);

  const diagnostics = await validateRows(table, generationId, rows, mode);
  const hasErrors = diagnostics.some((diagnostic) => diagnostic.severity === "error");
  if (hasErrors && !body.force) return c.json({ saved: false, diagnostics, requiresForce: true }, 409);

  const records = rows.map((row) => rowToRecord(row, schema));
  await writeYaml(tableFile(table, generationId), { [table]: records });
  return c.json({ saved: true, diagnostics, records, rows });
});

app.get("/api/tables/:table/references", async (c) => {
  const table = c.req.param("table");
  const activeGenerationId = c.req.query("activeGenerationId");
  const mode = c.req.query("mode");
  const generationId = activeGenerationId ?? c.req.query("generationId") ?? DEFAULT_GENERATION;
  const { schema, rows } = activeGenerationId && mode
    ? await effectiveReferenceRows(table, activeGenerationId, mode)
    : await loadRows(table, generationId);
  return c.json({
    candidates: rows.map((row) => ({
      key: keyFromRow(row, schema),
      label: rowLabel(row, schema),
      sourceGenerationId: row.sourceGenerationId,
      sourceGenerationLabel: row.sourceGenerationLabel,
      overrodeGenerationIds: row.overrodeGenerationIds ?? []
    }))
  });
});

app.get("/api/schemas", async (c) => {
  const schemas = await loadSchemas();
  return c.json({ schemas, rows: schemas.map(schemaListRow) });
});

app.put("/api/schemas", async (c) => {
  const body = await c.req.json();
  const rows = Array.isArray(body.rows) ? body.rows : [];
  const currentSchemas = await loadSchemas();
  const byId = new Map(currentSchemas.map((schema) => [schema.table_id, schema]));
  const diagnostics = [];
  const names = new Set();
  for (const row of rows) {
    if (!row.system_name) diagnostics.push({ severity: "error", table: row.table_id, field: "system_name", message: "system_name is required." });
    if (names.has(row.system_name)) diagnostics.push({ severity: "error", table: row.table_id, field: "system_name", message: `Duplicate table system_name: ${row.system_name}.` });
    names.add(row.system_name);
  }
  if (schemaBlockingDiagnostics(diagnostics).length) return c.json({ saved: false, diagnostics, requiresForce: false }, 422);

  const renamed = [];
  const changedSchemas = [];
  const renamedDataFiles = [];
  for (const row of rows) {
    const oldId = row.table_id || row.original_system_name || row.system_name;
    const current = byId.get(oldId);
    if (!current) {
      const systemName = row.system_name?.trim();
      if (!systemName) continue;
      const schema = normalizeSchema(systemName, {
        system_name: systemName,
        business_name: row.business_name || systemName,
        primary_key: [],
        export: row.export !== false,
        comment: row.comment ?? "",
        fields: []
      });
      await writeYaml(schemaFile(systemName), schemaToYaml(schema));
      changedSchemas.push(systemName);
      continue;
    }
    const nextId = row.system_name.trim();
    const latestCurrent = existsSync(schemaFile(oldId)) ? await loadSchema(oldId) : current;
    const nextSchema = {
      ...latestCurrent,
      table_id: nextId,
      system_name: nextId,
      business_name: row.business_name || latestCurrent.business_name || nextId,
      export: row.export !== false,
      comment: row.comment ?? ""
    };
    if (nextId !== oldId) {
      if (existsSync(schemaFile(nextId))) throw httpError(409, `Schema already exists: ${nextId}`);
      const dataFiles = await renameTableDataFiles(oldId, nextId);
      renamedDataFiles.push(...dataFiles);
      const referenceSchemas = await updateSchemaReferences(oldId, nextId);
      await writeYaml(schemaFile(nextId), schemaToYaml(nextSchema));
      await rm(schemaFile(oldId));
      renamed.push({ from: oldId, to: nextId, referenceSchemas, dataFiles });
      changedSchemas.push(nextId, ...referenceSchemas);
    } else {
      await writeYaml(schemaFile(oldId), schemaToYaml(nextSchema));
      changedSchemas.push(oldId);
    }
  }
  const latest = await loadSchemas();
  return c.json({ saved: true, schemas: latest, rows: latest.map(schemaListRow), renamed, renamedDataFiles, changedSchemas: [...new Set(changedSchemas)], diagnostics });
});

app.get("/api/schemas/:table", async (c) => {
  const schema = await loadSchema(c.req.param("table"));
  return c.json({ schema, fieldRows: schemaFieldRows(schema), tables: (await loadSchemas()).map((item) => item.table_id) });
});

app.put("/api/schemas/:table", async (c) => {
  const table = c.req.param("table");
  const result = await saveSchemaDetail(table, await c.req.json());
  return c.json(result.payload, result.status);
});

app.post("/api/schemas/:table/fields/rename", async (c) => {
  const table = c.req.param("table");
  const body = await c.req.json();
  const result = await renameSchemaFields(table, body.renames ?? [{ from: body.from, to: body.to }], body.force ?? false);
  return c.json(result.payload, result.status);
});

app.post("/api/schemas/:table/fields/delete", async (c) => {
  const table = c.req.param("table");
  const body = await c.req.json();
  const result = await deleteSchemaFields(table, body.fieldNames ?? body.fields ?? [], body.force ?? false);
  return c.json(result.payload, result.status);
});

app.post("/api/schemas/:table/rename", async (c) => {
  const table = c.req.param("table");
  const body = await c.req.json();
  const nextTable = String(body.system_name ?? "").trim();
  if (!nextTable) throw httpError(400, "system_name is required.");
  const current = await loadSchema(table);
  if (existsSync(schemaFile(nextTable))) throw httpError(409, `Schema already exists: ${nextTable}`);
  const dataFiles = await renameTableDataFiles(table, nextTable);
  const referenceSchemas = await updateSchemaReferences(table, nextTable);
  const nextSchema = { ...current, table_id: nextTable, system_name: nextTable };
  await writeYaml(schemaFile(nextTable), schemaToYaml(nextSchema));
  await rm(schemaFile(table));
  const latest = await loadSchemas();
  return c.json({ saved: true, schema: normalizeSchema(nextTable, nextSchema), schemas: latest, rows: latest.map(schemaListRow), renamed: { from: table, to: nextTable, referenceSchemas, dataFiles } });
});

app.post("/api/schemas/delete", async (c) => {
  const body = await c.req.json();
  const ids = Array.isArray(body.tableIds) ? body.tableIds : [];
  if (!ids.length) throw httpError(400, "tableIds is required.");
  const deletedPaths = [];
  for (const id of ids) {
    const filePath = schemaFile(id);
    if (!existsSync(filePath)) throw httpError(404, `Schema not found: ${id}`);
    await rm(filePath);
    deletedPaths.push(path.relative(ROOT, filePath));
  }
  const latest = await loadSchemas();
  return c.json({ deletedTableIds: ids, deletedPaths, schemas: latest, rows: latest.map(schemaListRow), diagnostics: [] });
});

app.get("/api/generations", async (c) => {
  return c.json(await loadGenerations());
});

app.post("/api/generations", async (c) => {
  const body = await c.req.json().catch(() => ({}));
  return c.json(await createGeneration(body.config ?? null), 201);
});

app.put("/api/generations/:generationId/config", async (c) => {
  const generationId = c.req.param("generationId");
  const body = await c.req.json();
  return c.json(await updateGeneration(generationId, body.config ?? body));
});

app.post("/api/generations/persistent-merge", async (c) => {
  const body = await c.req.json();
  return c.json(await persistentMergeGenerations(body.sourceGenerationIds, body.destination));
});

app.post("/api/generations/delete", async (c) => {
  const body = await c.req.json();
  return c.json(await deleteGenerations(body.generationIds, body.activeGenerationId ?? ""));
});

app.post("/api/generations/duplicate", async (c) => {
  const body = await c.req.json();
  if (Array.isArray(body.sourceGenerationIds)) {
    return c.json(await duplicateGenerations(body.sourceGenerationIds), 201);
  }
  if (body.destination) {
    return c.json(await duplicateGeneration(body.sourceGenerationId, body.destination), 201);
  }
  return c.json(await duplicateGenerations([body.sourceGenerationId]), 201);
});

app.post("/api/generations/analyze", async (c) => {
  const body = await c.req.json();
  return c.json(await analyzeGenerations(body.generationIds, body.includeMergeImpact ?? true));
});

app.post("/api/exports/check", async (c) => {
  const body = await c.req.json();
  const dataset = await buildExportDataset(body.generationIds, body.format);
  return c.json({
    exportable: dataset.exportable,
    generationIds: dataset.generationIds,
    orderedGenerationIds: dataset.orderedGenerationIds,
    format: dataset.format,
    summary: dataset.summary,
    diagnostics: dataset.diagnostics
  });
});

app.post("/api/exports", async (c) => {
  const body = await c.req.json();
  const dataset = await buildExportDataset(body.generationIds, body.format);
  if (!dataset.exportable) {
    return c.json({
      exportable: false,
      generationIds: dataset.generationIds,
      orderedGenerationIds: dataset.orderedGenerationIds,
      format: dataset.format,
      summary: dataset.summary,
      diagnostics: dataset.diagnostics
    }, 422);
  }
  const buffer = buildExportArtifact(dataset);
  const exportId = `${new Date().toISOString().replace(/[:.]/g, "-")}_${dataset.format}`;
  const filename = exportFilename(dataset.format);
  const contentType = exportContentType(dataset.format);
  EXPORT_ARTIFACTS.set(exportId, { buffer, filename, contentType, createdAt: Date.now() });
  return c.json({
    exportId,
    format: dataset.format,
    filename,
    contentType,
    downloadUrl: `/api/exports/${encodeURIComponent(exportId)}/download`,
    generationIds: dataset.generationIds,
    orderedGenerationIds: dataset.orderedGenerationIds,
    summary: dataset.summary,
    diagnostics: []
  }, 201);
});

app.get("/api/exports/:exportId/download", async (c) => {
  const artifact = EXPORT_ARTIFACTS.get(c.req.param("exportId"));
  if (!artifact) throw httpError(404, "Export artifact not found or expired.");
  return c.body(artifact.buffer, 200, {
    "Content-Type": artifact.contentType,
    "Content-Disposition": `attachment; filename="${artifact.filename}"`,
    "Cache-Control": "no-store"
  });
});

app.use("/*", serveStatic({ root: "./dist" }));
app.get("*", serveStatic({ path: "./dist/index.html" }));

app.onError((error, c) => {
  return c.json({ error: error.message }, error.status ?? 500);
});

serve({ fetch: app.fetch, port: Number(process.env.PORT ?? 8787), hostname: "127.0.0.1" }, (info) => {
  console.log(`MasterDataMate server listening on http://${info.address}:${info.port}`);
});
