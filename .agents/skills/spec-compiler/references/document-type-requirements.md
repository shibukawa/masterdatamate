# Extensible Document Type Framework — Product Requirements

## 1. Purpose

The specification system must support many kinds of documents without requiring the compiler to hard-code every future document type. A document type defines how a class of Markdown documents is authored, validated, summarized, indexed, linked, and visualized.

This requirement set defines a **document-type framework** in which:

- the skill ships with a read-only default baseline set of document type assets,
- each document type is described declaratively through a reusable type definition,
- user-defined document types use the same mechanism as built-in types,
- project-local assets provide mutable overrides and extensions,
- built-in and user-defined types participate in the same compile pipeline wherever the declarative framework provides enough information.

The framework should let teams start from a practical default set while preserving the ability to model project-specific document kinds later.

## 2. Goals

### 2.1 Primary goals

1. Make document types extensible without requiring product changes for every new project-specific type.
2. Keep compiler behavior consistent across built-in and user-defined types.
3. Provide a blessed starter kit that is immediately useful for common software-system specifications.
4. Preserve a coherent cross-document graph through shared families, facts, and relation rules.
5. Allow generated outputs such as summaries, indexes, backlinks, search records, and diagrams to be driven from type definitions rather than duplicated custom logic.
6. Support reusable common-detail patterns for document types whose facts are shared across many concrete documents.
7. Support retrieval and feedback hints that help AI agents load context incrementally and help feedback workflows improve document structure over time.

### 2.2 Non-goals for v1

1. A fully programmable extension runtime for arbitrary compiler behavior.
2. A requirement that every project use every built-in document type.
3. A fixed implementation file format for document type assets before the schema is designed.
4. A requirement that user-defined types support capabilities the declarative framework cannot describe.

## 3. Core Model

### 3.1 Document type family

A **document type family** is a broad semantic category used to keep validation, querying, and visualization coherent across related specialized types.

Examples include:

- system,
- requirement,
- decision,
- interface,
- UI,
- component,
- data,
- reference.

Families provide stable compiler semantics. Specialized document types provide authoring specificity.

### 3.2 Specialized document type

A **specialized document type** is the author-facing type selected when creating a document, such as `server component`, `batch component`, `UI screen`, or `error message`.

Each specialized type must declare its family and may add more specific:

- sections,
- metadata,
- facts,
- relation rules,
- indexes,
- summaries,
- validation behavior,
- diagram contributions.

### 3.3 Document type definition asset

A **document type definition asset** is a compiler-readable description of one specialized document type.

Document type assets must be discoverable from:

1. the skill-provided asset bundle, and
2. project-local extension locations.

The exact storage format is intentionally unspecified in these requirements, but the asset model must be expressive enough to declare the capabilities in section 4.

Skill-provided document type assets are immutable baseline defaults. Project-specific changes to type definitions or matching templates must be represented as project-local assets, either as new custom types or as project-local overrides of built-in types.

### 3.4 Built-in versus user-defined types

Built-in types and user-defined types must use the same conceptual model.

- **Built-in types** are shipped by the skill as the default baseline starter kit.
- **User-defined types** are added by a project using the same supported declaration mechanism.
- **Customized built-in types** are represented as project-local overrides rather than edits to installed skill assets.

The compiler must not assume that only built-in types can be validated, indexed, summarized, linked, or queried.

When a project-local type definition overrides a skill-provided type, the compiler must resolve and validate the effective type behavior before compiling dependent documents. Skill upgrades must not overwrite project-local type customizations.

### 3.5 Common detail document

A **common detail document** is a canonical document that stores reusable facts or sections shared by many documents of the same type or family.

For example, a common ERD detail document may define audit columns that apply to many data-model documents, while each concrete data-model document links to that common detail document instead of restating those columns.

Document type definitions must be able to describe whether a type:

- may link to common detail documents,
- may require one or more common detail links,
- should emit reuse/application relations for those links,
- should include referenced common detail in validation, summarization, indexing, or visualization behavior where supported.

## 4. Document Type Definition Requirements

Each document type definition must be able to declare the following capabilities.

### 4.1 Identity and classification

- stable type identifier,
- display name,
- family/category,
- optional aliases,
- compatibility or version metadata where needed.

### 4.2 Authoring shape

- authoring template,
- required sections,
- optional sections,
- expected compact canonical sections for retrieval,
- optional native-language-summary expectations where relevant.

### 4.3 Metadata and facts

- allowed metadata fields,
- required metadata fields,
- allowed declared facts,
- required declared facts,
- basic constraints such as cardinality, allowed values, and data shape where supported by the framework.

### 4.4 Relation extraction

- relation types that may be inferred from surrounding Markdown structure,
- section- or table-based extraction rules,
- rules for interpreting linked documents in context,
- constraints on valid source and target families or types where applicable.

### 4.5 Compiler behavior

- validation rules,
- summarization behavior,
- index-generation behavior,
- searchable section selection,
- backlink participation,
- relation emission,
- optional visualization hooks.

### 4.6 Common-detail behavior

- whether the type supports common detail documents,
- allowed or required common-detail target types/families,
- relation semantics for links to common detail,
- whether referenced common facts participate in validation or derived outputs,
- whether summaries or indexes should surface inherited/reused common facts.

### 4.7 Retrieval and feedback behavior

- expected maximum size for canonical retrieval sections,
- sections that should be indexed as primary retrieval slices,
- sections that are supporting context rather than primary context,
- recommended relation types for context expansion,
- task-specific context roles, such as feature implementation, API change, data-model change, impact analysis, or drift audit,
- feedback diagnostics that are meaningful for the type, such as oversized sections, missing required relations, weak aliases, repeated co-retrieval, or common-detail extraction opportunities.

The compiler must consume document type definitions declaratively wherever possible rather than relying only on hard-coded knowledge of built-in document types.

## 5. Default Baseline Starter Kit

The skill must ship with a default baseline set of document type assets. These are standard starting types, not merely examples.

### 5.1 Baseline document types

The starter kit should include at least:

- system,
- requirement / feature,
- business rule,
- architectural decision,
- UI screen,
- UI flow,
- API,
- error message,
- server component,
- batch component,
- data model,
- data domain,
- data flow,
- glossary term.

The baseline should allow relevant starter types, especially data-oriented types such as `data model`, to use common detail documents for shared reusable facts.

### 5.2 Recommended family mapping

The baseline should use a two-layer model:

| Family | Specialized starter types |
| --- | --- |
| system | system |
| requirement | requirement / feature, business rule |
| decision | architectural decision |
| interface | API, error message |
| UI | UI screen, UI flow |
| component | server component, batch component |
| data | data model, data domain, data flow |
| reference | glossary term |

The exact family names may evolve, but the product must preserve the distinction between:

- broad families for compiler/query coherence, and
- specialized types for authoring clarity.

### 5.3 Required first-class operational types

The baseline must include the following first-class types because they carry cross-cutting system knowledge that should not be buried in unrelated documents:

- **API** — the contract boundary linking UI, components, batches, and external systems.
- **Error message** — a reusable operational artifact linking user-facing behavior, APIs, batches, and troubleshooting.
- **Business rule** — a durable cross-cutting rule that may apply across UI, server, batch, and data behavior.

## 6. Data Flow Requirements

### 6.1 Authored data-flow documents

The starter kit must include `data flow` as an authored document type for important end-to-end scenarios that need a concise canonical explanation.

An authored data-flow document should be able to describe:

- actors or initiating systems,
- processes or participating components,
- stores or data objects,
- exchanged data,
- linked requirements, APIs, screens, or batches,
- explicit flow relations needed for later visualization and query.

### 6.2 Generated DFDs

DFD views must also be generated from compiled relation data elsewhere in the corpus when sufficient facts exist.

The system must not require every useful DFD to originate from a standalone data-flow document. UI, API, component, batch, and data documents may jointly contribute relation records from which a DFD is generated.

### 6.3 Shared truth model

Authored data-flow documents and generated DFDs must use the same compiled relation model as the rest of the system. Diagrams must not become a separate hand-maintained source of truth.

## 7. User-Defined Document Types

### 7.1 Extension capability

Users must be able to add project-specific document types through the same document-type framework used by built-in types.

A user-defined type should be able to:

- define its authoring template and required structure,
- declare facts and relation rules,
- participate in deterministic validation,
- produce searchable slices and summaries,
- appear in generated indexes,
- participate in backlinks and graph queries,
- contribute to diagrams when the available visualization hooks are sufficient,
- declare retrieval hints and feedback diagnostics where supported,
- define supported common-detail reuse behavior when the declarative framework can express it.

### 7.2 Compile-pipeline parity

Where the declarative framework supports a capability, user-defined types must be treated as first-class participants in the compile pipeline rather than second-class opaque Markdown documents.

At minimum, supported custom types should be eligible for:

- document parsing,
- metadata validation,
- section validation,
- declared-fact extraction,
- relation extraction,
- backlink generation,
- search record generation,
- summary generation,
- index inclusion,
- graph-query participation,
- context-orchestration participation,
- feedback-diagnostic participation,
- common-detail participation where declared.

### 7.3 Extension boundaries

v1 must support declarative extension of document types, but it does not need to support arbitrary executable compiler plugins.

If a desired user-defined type requires:

- a new parser,
- a new relation-inference algorithm,
- a new diagram family,
- or validation logic that cannot be expressed through the supported declaration model,

then that capability may require compiler changes in v1.

The system should surface such unsupported behavior clearly rather than silently accepting definitions it cannot fully honor.

## 8. Compiler Requirements

### 8.1 Type asset loading

The compiler must:

- load built-in document type definitions from the skill asset bundle,
- discover project-local custom type definitions,
- discover project-local overrides for built-in type definitions and matching templates,
- resolve project-local assets before skill-provided defaults,
- validate type definitions before validating documents that depend on them,
- validate the effective resolved type behavior before compiling dependent documents,
- reject malformed or incomplete type definitions with clear errors.

The compiler and feedback workflows must treat installed skill-provided type assets as read-only. Any customization, feedback-driven improvement, or local template change must be written or proposed in the configured project-local asset locations.

### 8.2 Type-aware document compilation

For each document, the compiler must:

- resolve the declared specialized type,
- resolve its family,
- apply the relevant type-definition rules,
- emit compiled artifacts using the same common pipeline as other supported types.

### 8.3 Type-definition validation

The compiler must detect:

- missing stable identifiers,
- missing family declarations,
- missing required template or section configuration,
- malformed fact declarations,
- malformed relation extraction rules,
- incompatible or unsupported visualization hooks,
- duplicate type identifiers,
- invalid compatibility/version declarations where applicable.

### 8.4 Generated outputs

Type definitions must be able to influence:

- document templates,
- generated indexes,
- summaries,
- searchable section records,
- backlinks,
- relation records,
- Mermaid diagram participation where supported,
- context orchestration and expansion hints,
- feedback diagnostics and recommendations,
- handling of reusable common detail documents where supported.

## 9. Conceptual Interface

The eventual type-definition asset schema should expose concepts equivalent to:

```yaml
id: "server-component"
name: "Server Component"
family: "component"
template: "..."
sections:
  required: []
  optional: []
metadata:
  required: []
  optional: []
facts:
  required: []
  optional: []
relations: []
validation: []
indexing: {}
summarization: {}
retrieval: {}
feedback: {}
visualization: {}
common_detail: {}
compatibility: {}
```

This example is conceptual, not a final schema commitment. The implementation may choose another representation if it preserves equivalent capabilities.

## 10. Acceptance Scenarios

### 10.1 Built-in type validation

Given a built-in `server component` type asset and a matching document, when compilation runs, then the compiler must load the type definition and validate the document against that definition.

### 10.2 Custom type without compiler changes

Given a user-added custom document type definition that uses supported declarative features, when a document of that type is compiled, then the compiler must validate and process it without requiring compiler code changes.

### 10.3 Custom type participation

Given a valid user-defined type, when matching documents are compiled, then they must be able to contribute search slices, summaries, backlinks, relation records, graph-query results, and generated index entries in the same manner as built-in types where configured.

### 10.4 Data-flow dual role

Given an authored `data flow` document and sufficiently linked related documents, when compilation and visualization run, then the system must emit relation records from the authored document and be able to generate a DFD from the shared compiled relation model.

### 10.5 Invalid type definition

Given a malformed document type definition, such as one missing required identity metadata or containing invalid relation rules, when the compiler loads type assets, then it must reject that type definition with a clear deterministic error.

### 10.6 Built-in and custom coexistence

Given a corpus containing both built-in and user-defined document types, when compilation and graph querying run, then documents from both sources must coexist in one compiled corpus and participate in backlinks and relation queries according to their definitions.

### 10.7 Common detail reuse

Given a common ERD detail document that defines reusable audit columns and several data-model documents that link to it, when compilation runs, then the type framework must allow the compiler to validate those links, emit reuse relations, and make the referenced common facts available to downstream summaries, indexes, and queries according to the data-model type definition.

### 10.8 Project-local type override

Given a project-local override for a built-in `API` type definition or template, when compilation and document creation run, then the compiler must use the project-local asset in preference to the skill-provided default while leaving the installed skill asset unchanged.

### 10.9 Feedback-driven type improvement

Given corpus feedback that recommends adding a retrieval hint or maximum section-size expectation to a document type, when the recommendation is accepted, then the change must be written to a project-local type asset or override and validated before it affects compilation.

### 10.10 Retrieval and feedback participation

Given a customized project-local type with retrieval and feedback hints, when context orchestration and corpus feedback run, then matching documents must use those hints for slice selection, relation expansion, and type-specific diagnostics.

## 11. Success Metrics

The document-type framework is successful when:

- new project-specific document types can be introduced without routine compiler changes,
- built-in and user-defined types behave consistently in the compile pipeline,
- the default baseline is useful enough for teams to begin authoring immediately,
- cross-document relations remain queryable across specialized types,
- generated indexes and diagrams stay aligned with declarative source definitions,
- retrieval and feedback behavior can be shaped declaratively rather than hard-coded for every type,
- project-local overrides can evolve without mutating skill-provided baseline assets,
- the compiler rejects unsupported or malformed type behavior before it causes silent drift.

## 12. Assumptions and Defaults

1. This document specifies product requirements, not the final serialized schema for type assets.
2. Built-in document types are a blessed default baseline shipped with the skill assets.
3. The product preserves a two-layer model of broad families and specialized authoring types.
4. `Data flow` is both an authored starter type and a contributor to generated DFD views.
5. User-defined types are first-class participants wherever the declarative framework can describe the needed behavior.
6. Declarative extension is required in v1; arbitrary executable extensions are not.
7. Some document types may rely on linked common detail documents for reusable shared facts instead of duplicating those facts in every concrete document.
8. Skill-provided document type and template assets are immutable defaults.
9. Project-local type and template assets are the mutable layer for custom types, customized built-in types, feedback improvements, and project-specific retrieval behavior.
10. Project-local overrides take precedence over skill-provided assets during asset resolution.
