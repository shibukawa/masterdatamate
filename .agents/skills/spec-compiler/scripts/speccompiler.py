#!/usr/bin/env python3
"""Starter CLI for AI-friendly Markdown spec corpora."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import sys
from collections import defaultdict, deque
from pathlib import Path
from urllib.parse import unquote


CONFIG_NAME = "speccompiler.json"
DEFAULT_GENERATED = ".speccompiler/generated"
KANBAN = ["backlog", "ready", "in progress", "blocked", "review", "done"]
FRONTMATTER_RE = re.compile(r"^---\n(.*?)\n---\n?", re.DOTALL)
HEADING_RE = re.compile(r"^(#{1,6})\s+(.+?)\s*$")
LINK_RE = re.compile(r"\[([^\]]+)\]\(([^)]+)\)")
BARE_ID_RE = re.compile(r"^[a-z0-9][a-z0-9-]*$")


def skill_dir() -> Path:
    return Path(__file__).resolve().parents[1]


def project_path(args) -> Path:
    return Path(getattr(args, "project", ".")).resolve()


def config_path(project: Path) -> Path:
    return project / CONFIG_NAME


def load_baseline() -> dict:
    path = skill_dir() / "assets/type-definitions/baseline.json"
    return json.loads(path.read_text())


def default_config() -> dict:
    return {
        "version": 1,
        "document_sets": [{"name": "main", "path": "specs"}],
        "shared_folders": ["specs/shared"],
        "generated_path": DEFAULT_GENERATED,
        "backlog_path": "BACKLOG.md",
        "project_assets": {
            "templates": "spec-assets/templates",
            "type_definitions": "spec-assets/type-definitions",
            "retrieval_recipes": "spec-assets/retrieval-recipes",
            "feedback_rules": "spec-assets/feedback-rules",
        },
        "skill_assets": {
            "templates": str(skill_dir() / "assets/templates"),
            "type_definitions": str(skill_dir() / "assets/type-definitions"),
        },
        "implementation_roots": [],
    }


def load_config(project: Path) -> dict:
    path = config_path(project)
    if not path.exists():
        raise SystemExit(f"Missing {CONFIG_NAME}. Run: speccompiler.py init --project {project}")
    return json.loads(path.read_text())


def write_json(path: Path, data: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n")


def slugify(value: str) -> str:
    slug = re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-")
    return re.sub(r"-{2,}", "-", slug) or "untitled"


def read_markdown(path: Path) -> tuple[dict, str]:
    text = path.read_text()
    match = FRONTMATTER_RE.match(text)
    if not match:
        return {}, text
    body = text[match.end() :]
    return parse_simple_yaml(match.group(1)), body


def parse_simple_yaml(raw: str) -> dict:
    """Parse the small frontmatter subset used by the starter templates."""
    try:
        import yaml  # type: ignore

        parsed = yaml.safe_load(raw)
        return parsed or {}
    except Exception:
        pass
    data: dict = {}
    stack: list[tuple[int, object]] = [(-1, data)]
    current_key_at_indent: dict[int, str] = {}
    for line in raw.splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        indent = len(line) - len(line.lstrip(" "))
        stripped = line.strip()
        while stack and indent <= stack[-1][0]:
            stack.pop()
        parent = stack[-1][1]
        if stripped.startswith("- "):
            item = clean_scalar(stripped[2:])
            if isinstance(parent, list):
                parent.append(item)
            continue
        if ":" not in stripped:
            continue
        key, value = stripped.split(":", 1)
        key = key.strip()
        value = value.strip()
        if value == "":
            container: object = {}
            if isinstance(parent, dict):
                parent[key] = container
            stack.append((indent, container))
            current_key_at_indent[indent] = key
        elif value == "[]":
            if isinstance(parent, dict):
                parent[key] = []
        else:
            if isinstance(parent, dict):
                parent[key] = clean_scalar(value)
    return data


def clean_scalar(value: str):
    value = value.strip()
    if len(value) >= 2 and value[0] == value[-1] and value[0] in {"'", '"'}:
        return value[1:-1]
    if value in {"true", "false"}:
        return value == "true"
    return value


def document_roots(project: Path, cfg: dict) -> list[Path]:
    roots: list[Path] = []
    for docset in cfg.get("document_sets", []):
        roots.append(project / docset["path"])
    for folder in cfg.get("shared_folders", []):
        roots.append(project / folder)
    seen = set()
    result = []
    for root in roots:
        resolved = root.resolve()
        if resolved not in seen:
            seen.add(resolved)
            result.append(root)
    return result


def iter_docs(project: Path, cfg: dict):
    generated = (project / cfg.get("generated_path", DEFAULT_GENERATED)).resolve()
    backlog = (project / cfg.get("backlog_path", "BACKLOG.md")).resolve()
    for root in document_roots(project, cfg):
        if not root.exists():
            continue
        for path in sorted(root.rglob("*.md")):
            resolved = path.resolve()
            if generated in resolved.parents or resolved == backlog:
                continue
            yield path


def docset_name(project: Path, cfg: dict, path: Path) -> str:
    resolved = path.resolve()
    for docset in cfg.get("document_sets", []):
        root = (project / docset["path"]).resolve()
        if resolved == root or root in resolved.parents:
            return docset["name"]
    return "shared"


def resolve_template(project: Path, cfg: dict, type_id: str) -> Path:
    baseline = load_baseline()
    type_map = {item["id"]: item for item in baseline["types"]}
    template_name = type_map.get(type_id, {}).get("template", "generic.md")
    local = project / cfg["project_assets"]["templates"] / template_name
    if local.exists():
        return local
    skill_template = skill_dir() / "assets/templates" / template_name
    if skill_template.exists():
        return skill_template
    return skill_dir() / "assets/templates/generic.md"


def render_template(template: str, values: dict) -> str:
    for key, value in values.items():
        template = template.replace("{{" + key + "}}", str(value))
    return template


def collect_records(project: Path, cfg: dict):
    docs = []
    sections = []
    links = []
    ids = {}
    for path in iter_docs(project, cfg):
        fm, body = read_markdown(path)
        rel = path.relative_to(project).as_posix()
        doc_id = str(fm.get("id") or "")
        record = {
            "id": doc_id,
            "path": rel,
            "type": fm.get("type", ""),
            "title": fm.get("title", path.stem),
            "aliases": fm.get("aliases", []),
            "tags": fm.get("tags", []),
            "facts": fm.get("facts", {}),
            "document_set": docset_name(project, cfg, path),
        }
        docs.append(record)
        if doc_id:
            ids[doc_id] = record
        sections.extend(extract_sections(record, body))
        links.extend(extract_links(project, path, record, body))
    return docs, sections, links, ids


def extract_sections(doc: dict, body: str) -> list[dict]:
    sections = []
    current = {"heading": "Document", "level": 0, "lines": []}
    def flush() -> None:
        if "\n".join(current["lines"]).strip():
            sections.append(section_record(doc, current))

    for line in body.splitlines():
        match = HEADING_RE.match(line)
        if match:
            flush()
            current = {"heading": match.group(2).strip(), "level": len(match.group(1)), "lines": []}
        else:
            current["lines"].append(line)
    flush()
    return sections


def section_record(doc: dict, section: dict) -> dict:
    text = "\n".join(section["lines"]).strip()
    return {
        "document_id": doc["id"],
        "document_title": doc["title"],
        "document_type": doc["type"],
        "section": section["heading"],
        "level": section["level"],
        "text": text,
        "estimated_tokens": max(1, len(text.split())),
    }


def extract_links(project: Path, path: Path, doc: dict, body: str) -> list[dict]:
    links = []
    current_heading = "Document"
    for line in body.splitlines():
        match = HEADING_RE.match(line)
        if match:
            current_heading = match.group(2).strip()
            continue
        for caption, target in LINK_RE.findall(line):
            if is_external_link(target):
                continue
            clean_target = target.split("#", 1)[0]
            resolved = (path.parent / unquote(clean_target)).resolve()
            try:
                target_path = resolved.relative_to(project).as_posix()
            except ValueError:
                target_path = str(resolved)
            links.append(
                {
                    "source_id": doc["id"],
                    "source_path": doc["path"],
                    "caption": caption,
                    "target": target,
                    "target_path": target_path,
                    "section": current_heading,
                }
            )
    return links


def is_external_link(target: str) -> bool:
    return bool(re.match(r"^[a-z][a-z0-9+.-]*:", target)) or target.startswith("#")


def infer_relations(links: list[dict], path_to_id: dict[str, str]) -> list[dict]:
    relation_map = load_baseline().get("relation_headings", {})
    relations = []
    for link in links:
        target_id = path_to_id.get(link["target_path"], "")
        relation = relation_map.get(link["section"].lower(), "links_to")
        relations.append(
            {
                "source_id": link["source_id"],
                "target_id": target_id,
                "relation": relation,
                "section": link["section"],
                "target_path": link["target_path"],
            }
        )
    return relations


def write_jsonl(path: Path, records: list[dict]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w") as f:
        for record in records:
            f.write(json.dumps(record, ensure_ascii=False) + "\n")


def read_jsonl(path: Path) -> list[dict]:
    if not path.exists():
        return []
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def command_init(args) -> int:
    project = project_path(args)
    cfg = default_config()
    path = config_path(project)
    if path.exists() and not args.force:
        print(f"Exists: {path}")
    else:
        write_json(path, cfg)
        print(f"Wrote: {path}")
    for docset in cfg["document_sets"]:
        (project / docset["path"]).mkdir(parents=True, exist_ok=True)
    for folder in cfg["shared_folders"]:
        (project / folder).mkdir(parents=True, exist_ok=True)
    for folder in cfg["project_assets"].values():
        (project / folder).mkdir(parents=True, exist_ok=True)
    backlog = project / cfg["backlog_path"]
    if not backlog.exists():
        backlog.write_text("# Project Backlog\n\n## Items\n\n")
        print(f"Wrote: {backlog}")
    validate_asset_resolution(project, cfg)
    return 0


def validate_asset_resolution(project: Path, cfg: dict) -> None:
    missing = []
    for key, folder in cfg["skill_assets"].items():
        if not Path(folder).exists():
            missing.append(f"skill_assets.{key}: {folder}")
    if missing:
        raise SystemExit("Missing skill assets:\n" + "\n".join(missing))


def command_create_doc(args) -> int:
    project = project_path(args)
    cfg = load_config(project)
    type_id = slugify(args.type)
    doc_id = args.id or slugify(args.title)
    docset = args.set
    docset_cfg = next((item for item in cfg["document_sets"] if item["name"] == docset), None)
    if not docset_cfg:
        raise SystemExit(f"Unknown document set: {docset}")
    out_dir = project / (args.dir or docset_cfg["path"]) / type_id
    out = Path(args.output) if args.output else out_dir / f"{doc_id}.md"
    if not out.is_absolute():
        out = project / out
    if out.exists() and not args.force:
        raise SystemExit(f"Refusing to overwrite existing document: {out}")
    template = resolve_template(project, cfg, type_id).read_text()
    text = render_template(
        template,
        {
            "id": doc_id,
            "type": type_id,
            "title": args.title,
            "status": args.status,
            "summary": args.summary or "TODO",
        },
    )
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(text)
    print(f"Wrote: {out}")
    return 0


def validate(project: Path, cfg: dict) -> tuple[list[str], list[str]]:
    errors = []
    warnings = []
    docs, _sections, links, _ids = collect_records(project, cfg)
    ids_seen: dict[str, str] = {}
    path_to_id = {doc["path"]: doc["id"] for doc in docs if doc["id"]}
    known_types = {item["id"] for item in load_baseline()["types"]}
    for doc in docs:
        path = doc["path"]
        if not doc["id"]:
            errors.append(f"{path}: missing required frontmatter id")
        elif doc["id"] in ids_seen:
            errors.append(f"{path}: duplicate id {doc['id']} also used by {ids_seen[doc['id']]}")
        else:
            ids_seen[doc["id"]] = path
        if not doc["type"]:
            errors.append(f"{path}: missing required frontmatter type")
        elif doc["type"] not in known_types:
            warnings.append(f"{path}: type '{doc['type']}' is not in baseline type definitions")
        if not doc["title"]:
            errors.append(f"{path}: missing required frontmatter title")
        facts = doc.get("facts")
        if facts and not isinstance(facts, dict):
            errors.append(f"{path}: facts must be a mapping")
    for link in links:
        if link["target_path"] not in path_to_id:
            errors.append(
                f"{link['source_path']}: dangling link target '{link['target']}' from section '{link['section']}'"
            )
    generated = project / cfg.get("generated_path", DEFAULT_GENERATED)
    if generated.exists():
        gen_mtime = max((p.stat().st_mtime for p in generated.rglob("*") if p.is_file()), default=0)
        source_mtime = max((p.stat().st_mtime for p in iter_docs(project, cfg)), default=0)
        if source_mtime > gen_mtime:
            warnings.append("generated outputs appear stale; run compile")
    return errors, warnings


def command_validate(args) -> int:
    project = project_path(args)
    cfg = load_config(project)
    errors, warnings = validate(project, cfg)
    for item in warnings:
        print(f"WARN: {item}")
    for item in errors:
        print(f"ERROR: {item}")
    print(f"Validation: {len(errors)} error(s), {len(warnings)} warning(s)")
    return 1 if errors else 0


def command_compile(args) -> int:
    project = project_path(args)
    cfg = load_config(project)
    errors, warnings = validate(project, cfg)
    if errors and not args.allow_errors:
        for item in errors:
            print(f"ERROR: {item}")
        print("Compile stopped because validation has errors. Pass --allow-errors to emit partial artifacts.")
        return 1
    docs, sections, links, _ids = collect_records(project, cfg)
    path_to_id = {doc["path"]: doc["id"] for doc in docs if doc["id"]}
    relations = infer_relations(links, path_to_id)
    out = project / cfg.get("generated_path", DEFAULT_GENERATED)
    write_jsonl(out / "documents.jsonl", docs)
    write_jsonl(out / "sections.jsonl", sections)
    write_jsonl(out / "links.jsonl", links)
    write_jsonl(out / "relations.jsonl", relations)
    write_backlinks(out / "backlinks.md", docs, relations)
    write_indexes(out / "indexes", docs)
    for item in warnings:
        print(f"WARN: {item}")
    print(f"Compiled {len(docs)} document(s), {len(sections)} section(s), {len(relations)} relation(s) into {out}")
    return 0


def write_backlinks(path: Path, docs: list[dict], relations: list[dict]) -> None:
    titles = {doc["id"]: doc["title"] for doc in docs}
    inbound: dict[str, list[dict]] = defaultdict(list)
    for rel in relations:
        if rel["target_id"]:
            inbound[rel["target_id"]].append(rel)
    lines = ["# Backlinks", ""]
    for doc in docs:
        lines.extend([f"## {doc['title']} ({doc['id']})", ""])
        for rel in inbound.get(doc["id"], []):
            lines.append(f"- {rel['relation']} from {titles.get(rel['source_id'], rel['source_id'])}")
        if not inbound.get(doc["id"]):
            lines.append("- No inbound links")
        lines.append("")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("\n".join(lines))


def write_indexes(path: Path, docs: list[dict]) -> None:
    by_type: dict[str, list[dict]] = defaultdict(list)
    for doc in docs:
        by_type[doc["type"]].append(doc)
    path.mkdir(parents=True, exist_ok=True)
    for type_id, records in sorted(by_type.items()):
        lines = [f"# {type_id or 'untyped'} index", ""]
        for doc in sorted(records, key=lambda item: item["title"]):
            lines.append(f"- [{doc['title']}](../../{doc['path']}) `{doc['id']}`")
        (path / f"{type_id or 'untyped'}.md").write_text("\n".join(lines) + "\n")


def command_search(args) -> int:
    project = project_path(args)
    cfg = load_config(project)
    out = project / cfg.get("generated_path", DEFAULT_GENERATED)
    sections = read_jsonl(out / "sections.jsonl")
    if not sections:
        raise SystemExit("No compiled sections found. Run compile first.")
    terms = [term.lower() for term in re.findall(r"[a-zA-Z0-9_-]+", args.query)]
    scored = []
    for section in sections:
        haystack = " ".join(
            [
                str(section.get("document_title", "")),
                str(section.get("document_type", "")),
                str(section.get("section", "")),
                str(section.get("text", "")),
            ]
        ).lower()
        score = sum(haystack.count(term) for term in terms)
        if score:
            scored.append((score, section))
    for score, section in sorted(scored, key=lambda item: item[0], reverse=True)[: args.limit]:
        print(f"{score}\t{section['document_id']}\t{section['section']}\t{section['document_title']}")
        print(section["text"][: args.excerpt].replace("\n", " "))
    return 0


def command_impact(args) -> int:
    project = project_path(args)
    cfg = load_config(project)
    out = project / cfg.get("generated_path", DEFAULT_GENERATED)
    relations = read_jsonl(out / "relations.jsonl")
    docs = {doc["id"]: doc for doc in read_jsonl(out / "documents.jsonl")}
    if not relations:
        raise SystemExit("No compiled relations found. Run compile first.")
    start = args.entity
    forward = defaultdict(list)
    backward = defaultdict(list)
    for rel in relations:
        if rel["source_id"] and rel["target_id"]:
            forward[rel["source_id"]].append(rel)
            backward[rel["target_id"]].append(rel)
    queue = deque([(start, 0)])
    seen = {start}
    rows = []
    while queue:
        node, distance = queue.popleft()
        if distance >= args.depth:
            continue
        for rel in forward[node] + backward[node]:
            other = rel["target_id"] if rel["source_id"] == node else rel["source_id"]
            if not other or other in seen:
                continue
            seen.add(other)
            rows.append((distance + 1, rel, other))
            queue.append((other, distance + 1))
    for distance, rel, other in rows:
        doc = docs.get(other, {})
        path = f"{rel['source_id']} -> {rel['target_id']}"
        print(f"{distance}\t{rel['relation']}\t{other}\t{doc.get('title', '')}\t{path}")
    print(json.dumps({"start": start, "affected_ids": [row[2] for row in rows]}, ensure_ascii=False))
    return 0


def command_context(args) -> int:
    project = project_path(args)
    cfg = load_config(project)
    out = project / cfg.get("generated_path", DEFAULT_GENERATED)
    sections = read_jsonl(out / "sections.jsonl")
    if not sections:
        raise SystemExit("No compiled sections found. Run compile first.")
    terms = [term.lower() for term in re.findall(r"[a-zA-Z0-9_-]+", args.task)]
    selected = []
    used = 0
    for section in sorted(sections, key=lambda s: score_section(s, terms), reverse=True):
        score = score_section(section, terms)
        if score <= 0:
            continue
        cost = int(section.get("estimated_tokens", 1))
        if used + cost > args.budget:
            continue
        selected.append(section)
        used += cost
        if len(selected) >= args.limit:
            break
    bundle = {
        "task": args.task,
        "budget": args.budget,
        "estimated_tokens": used,
        "sufficient": bool(selected),
        "sections": selected,
        "omitted_hint": "Increase --budget or compile more relations if the selected context is insufficient.",
    }
    print(json.dumps(bundle, indent=2, ensure_ascii=False))
    return 0


def score_section(section: dict, terms: list[str]) -> int:
    haystack = " ".join(
        [
            str(section.get("document_title", "")),
            str(section.get("document_type", "")),
            str(section.get("section", "")),
            str(section.get("text", "")),
        ]
    ).lower()
    return sum(haystack.count(term) for term in terms)


def command_wizard(args) -> int:
    project = project_path(args)
    evidence = inspect_project(project)
    classification = classify_project(evidence)
    result = {
        "entry_point": classification,
        "architecture_recommendation": "modular monolith with explicit module boundaries",
        "first_runnable_slice": infer_first_slice(evidence),
        "evidence": evidence,
        "open_questions": [
            "What immediate user-visible behavior must the first slice prove?",
            "Which source areas should be inherited or ignored?",
            "What local command should prove the first slice works?",
        ],
    }
    print(json.dumps(result, indent=2, ensure_ascii=False))
    if args.write:
        cfg = default_config()
        if not config_path(project).exists():
            write_json(config_path(project), cfg)
        else:
            cfg = load_config(project)
        overview = project / cfg["document_sets"][0]["path"] / "system" / "project-overview.md"
        if not overview.exists():
            command_create_doc(
                argparse.Namespace(
                    project=str(project),
                    type="system",
                    title="Project overview",
                    id="project-overview",
                    set=cfg["document_sets"][0]["name"],
                    dir=None,
                    output=str(overview),
                    status="blueprint",
                    summary=f"Entry point: {classification}. First slice: {result['first_runnable_slice']}.",
                    force=False,
                )
            )
        add_backlog_item(project, cfg, "Select and validate first runnable slice", "implementation", "ready")
    return 0


def inspect_project(project: Path) -> dict:
    files = [p.relative_to(project).as_posix() for p in project.rglob("*") if p.is_file() and ".git" not in p.parts]
    manifests = [f for f in files if Path(f).name in {"package.json", "pyproject.toml", "Cargo.toml", "go.mod", "Package.swift"}]
    docs = [f for f in files if f.lower().endswith((".md", ".mdx"))]
    tests = [
        f
        for f in files
        if re.search(r"(^|/)(test|tests|__tests__)(/|$)", f.lower())
        or re.search(r"(\.test|\.spec)\.[^.]+$", f.lower())
    ]
    source = [f for f in files if f.lower().endswith((".py", ".js", ".ts", ".tsx", ".go", ".rs", ".swift", ".java", ".rb"))]
    return {
        "file_count": len(files),
        "manifests": manifests[:20],
        "markdown_docs": docs[:50],
        "test_files": tests[:50],
        "source_files": source[:50],
    }


def classify_project(evidence: dict) -> str:
    if evidence["source_files"] and evidence["markdown_docs"]:
        return "feature or change request in an active codebase"
    if evidence["source_files"]:
        return "legacy source code" if len(evidence["source_files"]) > 20 else "incomplete POC"
    if evidence["markdown_docs"]:
        return "docs-only or spec-only project"
    return "new project from scratch"


def infer_first_slice(evidence: dict) -> str:
    if evidence["manifests"]:
        return f"small locally runnable slice using {evidence['manifests'][0]}"
    if evidence["markdown_docs"]:
        return "first spec-backed blueprint and validation loop"
    return "minimal executable proof of the core idea"


def add_backlog_item(project: Path, cfg: dict, title: str, task_type: str, state: str) -> str:
    backlog = project / cfg.get("backlog_path", "BACKLOG.md")
    backlog.parent.mkdir(parents=True, exist_ok=True)
    if not backlog.exists():
        backlog.write_text("# Project Backlog\n\n## Items\n\n")
    existing = backlog.read_text()
    item_id = f"BLG-{dt.datetime.now().strftime('%Y%m%d%H%M%S')}"
    block = (
        f"- id: {item_id}\n"
        f"  state: {state}\n"
        f"  type: {task_type}\n"
        f"  title: {title}\n"
        f"  docs: []\n"
        f"  sources: []\n"
        f"  acceptance: TODO\n"
        f"  blockers: []\n"
        f"  updated: {dt.date.today().isoformat()}\n"
    )
    backlog.write_text(existing.rstrip() + "\n\n" + block)
    return item_id


def command_backlog(args) -> int:
    project = project_path(args)
    cfg = load_config(project)
    if args.backlog_cmd == "add":
        item_id = add_backlog_item(project, cfg, args.title, args.type, args.state)
        print(f"Added: {item_id}")
        return 0
    if args.backlog_cmd == "list":
        backlog = project / cfg.get("backlog_path", "BACKLOG.md")
        if not backlog.exists():
            print("No backlog found")
            return 0
        print(backlog.read_text())
        return 0
    if args.backlog_cmd == "move":
        backlog = project / cfg.get("backlog_path", "BACKLOG.md")
        text = backlog.read_text()
        pattern = re.compile(rf"(- id: {re.escape(args.id)}\n(?:  .+\n)*?  state: )(.+)")
        if pattern.search(text):
            text = pattern.sub(rf"\g<1>{args.state}", text, count=1)
        else:
            pattern = re.compile(rf"(- id: {re.escape(args.id)}\n)")
            text = pattern.sub(rf"\g<1>  state: {args.state}\n", text, count=1)
        backlog.write_text(text)
        print(f"Moved: {args.id} -> {args.state}")
        return 0
    raise SystemExit(f"Unknown backlog command: {args.backlog_cmd}")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Starter spec compiler for Markdown spec corpora")
    parser.add_argument("--project", default=".", help="Project root")
    sub = parser.add_subparsers(dest="command", required=True)

    init = sub.add_parser("init")
    init.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    init.add_argument("--force", action="store_true")
    init.set_defaults(func=command_init)

    create = sub.add_parser("create-doc")
    create.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    create.add_argument("--type", required=True)
    create.add_argument("--title", required=True)
    create.add_argument("--id")
    create.add_argument("--set", default="main")
    create.add_argument("--dir")
    create.add_argument("--output")
    create.add_argument("--status", default="blueprint")
    create.add_argument("--summary")
    create.add_argument("--force", action="store_true")
    create.set_defaults(func=command_create_doc)

    validate_cmd = sub.add_parser("validate")
    validate_cmd.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    validate_cmd.set_defaults(func=command_validate)

    compile_cmd = sub.add_parser("compile")
    compile_cmd.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    compile_cmd.add_argument("--allow-errors", action="store_true")
    compile_cmd.set_defaults(func=command_compile)

    search = sub.add_parser("search")
    search.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    search.add_argument("query")
    search.add_argument("--limit", type=int, default=10)
    search.add_argument("--excerpt", type=int, default=400)
    search.set_defaults(func=command_search)

    impact = sub.add_parser("impact")
    impact.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    impact.add_argument("entity")
    impact.add_argument("--depth", type=int, default=2)
    impact.set_defaults(func=command_impact)

    context = sub.add_parser("context")
    context.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    context.add_argument("task")
    context.add_argument("--budget", type=int, default=1200)
    context.add_argument("--limit", type=int, default=8)
    context.set_defaults(func=command_context)

    wizard = sub.add_parser("wizard")
    wizard.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    wizard.add_argument("--write", action="store_true")
    wizard.set_defaults(func=command_wizard)

    backlog = sub.add_parser("backlog")
    backlog.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    backlog_sub = backlog.add_subparsers(dest="backlog_cmd", required=True)
    add = backlog_sub.add_parser("add")
    add.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    add.add_argument("--title", required=True)
    add.add_argument("--type", default="documentation")
    add.add_argument("--state", choices=KANBAN, default="backlog")
    list_cmd = backlog_sub.add_parser("list")
    list_cmd.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    move = backlog_sub.add_parser("move")
    move.add_argument("--project", default=argparse.SUPPRESS, help="Project root")
    move.add_argument("id")
    move.add_argument("state", choices=KANBAN)
    backlog.set_defaults(func=command_backlog)

    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    try:
        return args.func(args)
    except BrokenPipeError:
        return 1


if __name__ == "__main__":
    sys.exit(main())
