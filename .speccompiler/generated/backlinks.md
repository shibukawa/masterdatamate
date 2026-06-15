# Backlinks

## Go CLI export runner (go-cli-export-runner)

- links_to from Export backend adapters
- depends_on from Export backend adapters
- links_to from Export execution flow
- uses_component from Export execution flow
- links_to from Export execution flow
- relates_to from Export execution flow
- depends_on from Export settings model
- links_to from Go embedded web server host
- relates_to from Go embedded web server host

## Go CLI generate runner (go-cli-generate-runner)

- relates_to from Go CLI export runner
- links_to from Export backend adapters
- depends_on from Export backend adapters
- depends_on from Pongo2 template generation adapter
- relates_to from Export execution flow
- depends_on from Export settings model
- depends_on from Template generation definition model
- depends_on from Template generation definition screen

## Agent tool contract (agent-tool-contract)

- links_to from AI assistant service
- links_to from AI assistant service
- links_to from AI assistant service
- links_to from AI assistant service
- links_to from AI assistant service
- depends_on from AI assistant service
- depends_on from Shared web editing frontend
- relates_to from AI provider configuration model
- relates_to from Product overview
- depends_on from Go embedded web server host
- depends_on from Wails desktop host
- links_to from Web service host
- links_to from Web service host
- depends_on from Web service host
- relates_to from Web service host
- invokes from In-app AI assistant panel
- uses_component from In-app AI assistant panel
- links_to from In-app AI assistant panel
- relates_to from In-app AI assistant panel

## AI assistant service (ai-assistant-service)

- depends_on from Agent tool contract
- depends_on from AI secret storage service
- depends_on from Shared web editing frontend
- reads from AI assistant session model
- relates_to from AI assistant session model
- relates_to from AI provider configuration model
- relates_to from Product overview
- links_to from Go embedded web server host
- depends_on from Go embedded web server host
- depends_on from Wails desktop host
- links_to from Web service host
- links_to from Web service host
- links_to from Web service host
- depends_on from Web service host
- relates_to from Web service host
- uses_component from AI settings screen
- invokes from In-app AI assistant panel
- invokes from In-app AI assistant panel
- uses_component from In-app AI assistant panel
- relates_to from In-app AI assistant panel

## AI secret storage service (ai-secret-storage-service)

- links_to from AI assistant service
- depends_on from AI assistant service
- relates_to from AI assistant session model
- links_to from AI provider configuration model
- reads from AI provider configuration model
- relates_to from AI provider configuration model
- relates_to from Product overview
- links_to from Go embedded web server host
- depends_on from Go embedded web server host
- depends_on from Wails desktop host
- links_to from Web service host
- depends_on from Web service host
- links_to from AI settings screen
- uses_component from AI settings screen

## Export backend adapters (export-backend-adapters)

- links_to from Go CLI export runner
- uses_component from Go CLI export runner
- relates_to from Go CLI export runner
- uses_component from Export execution flow
- relates_to from Export execution flow
- uses_component from Generation merge and export flow
- relates_to from Generation merge and export flow
- links_to from Export settings model
- depends_on from Export settings model
- relates_to from Table schema model
- relates_to from Product overview
- uses_component from Table editing workspace

## HTML editor plugin runtime (html-editor-plugin-runtime)

- relates_to from AI assistant service
- links_to from Shared web editing frontend
- depends_on from Shared web editing frontend
- relates_to from Binary asset model
- relates_to from Editor plugin model
- relates_to from Product overview
- depends_on from Web service host
- uses_component from Table editing workspace

## Pongo2 template generation adapter (pongo2-template-export-adapter)

- relates_to from Go CLI generate runner
- depends_on from Template generation definition model
- depends_on from Template generation definition screen

## Schema validation engine (schema-validation-engine)

- uses_component from Go CLI export runner
- relates_to from Go CLI export runner
- relates_to from Go CLI generate runner
- depends_on from Agent tool contract
- depends_on from AI assistant service
- links_to from Export backend adapters
- depends_on from Export backend adapters
- depends_on from HTML editor plugin runtime
- links_to from Shared web editing frontend
- depends_on from Shared web editing frontend
- uses_component from Export execution flow
- links_to from Export execution flow
- relates_to from Export execution flow
- uses_component from Generation analysis flow
- uses_component from Generation merge and export flow
- relates_to from Generation merge and export flow
- uses_component from Generation persistent merge flow
- relates_to from Binary asset model
- relates_to from Canonical YAML file layout
- links_to from Editor plugin model
- relates_to from Generic master data model
- relates_to from Table schema model
- relates_to from Product overview
- depends_on from Go embedded web server host
- depends_on from Wails desktop host
- depends_on from Web service host
- uses_component from Schema editing screen
- uses_component from Table editing workspace

## Shared web editing frontend (shared-web-editing-frontend)

- relates_to from AI assistant service
- depends_on from HTML editor plugin runtime
- relates_to from Editor plugin model
- relates_to from Product overview
- depends_on from Go embedded web server host
- depends_on from Wails desktop host
- depends_on from Web service host
- relates_to from Single page application shell
- uses_component from AI settings screen
- uses_component from In-app AI assistant panel
- uses_component from Table editing workspace

## Export execution flow (export-execution-flow)

- links_to from Go CLI export runner
- uses_component from Go CLI export runner
- links_to from Go CLI export runner
- relates_to from Go CLI export runner
- links_to from Export backend adapters
- depends_on from Export backend adapters
- links_to from Generation merge and export flow
- invokes from Generation merge and export flow
- relates_to from Generation merge and export flow
- depends_on from Export settings model
- depends_on from Template generation definition model
- relates_to from Go embedded web server host
- links_to from Single page application shell

## Generation analysis flow (generation-analysis-flow)

- depends_on from Shared web editing frontend
- reads from Canonical YAML file layout
- relates_to from Canonical YAML file layout
- relates_to from Generation model
- relates_to from Web service host
- uses_component from Generation editing screen
- relates_to from Generation editing screen

## Generation deletion flow (generation-deletion-flow)

- depends_on from Shared web editing frontend
- uses_component from Generation analysis flow
- relates_to from Generation analysis flow
- reads from Canonical YAML file layout
- relates_to from Canonical YAML file layout
- relates_to from Generation model
- relates_to from Web service host
- uses_component from Generation editing screen
- relates_to from Generation editing screen

## Generation duplication flow (generation-duplication-flow)

- depends_on from Shared web editing frontend
- reads from Canonical YAML file layout
- relates_to from Canonical YAML file layout
- relates_to from Generation model
- relates_to from Web service host
- uses_component from Generation editing screen
- relates_to from Generation editing screen

## Generation merge and export flow (generation-merge-and-export-flow)

- uses_component from Go CLI export runner
- relates_to from Go CLI export runner
- relates_to from Go CLI generate runner
- links_to from Export backend adapters
- relates_to from HTML editor plugin runtime
- links_to from Pongo2 template generation adapter
- depends_on from Schema validation engine
- depends_on from Shared web editing frontend
- uses_component from Export execution flow
- links_to from Export execution flow
- relates_to from Export execution flow
- uses_component from Generation persistent merge flow
- relates_to from Generation persistent merge flow
- reads from Canonical YAML file layout
- reads from Generation model
- relates_to from Generic master data model
- depends_on from Template generation definition model
- relates_to from Product overview
- relates_to from Go embedded web server host
- relates_to from Web service host
- uses_component from Generation editing screen
- uses_component from Generation selection screen
- relates_to from Table editing workspace

## Generation persistent merge flow (generation-persistent-merge-flow)

- depends_on from Shared web editing frontend
- uses_component from Generation analysis flow
- relates_to from Generation analysis flow
- links_to from Generation merge and export flow
- relates_to from Generation merge and export flow
- reads from Canonical YAML file layout
- relates_to from Canonical YAML file layout
- relates_to from Generation model
- relates_to from Web service host
- uses_component from Generation editing screen
- relates_to from Generation editing screen

## AI assistant session model (ai-assistant-session-model)

- links_to from AI assistant service
- links_to from AI assistant service
- depends_on from AI assistant service
- links_to from AI provider configuration model
- links_to from Web service host
- links_to from In-app AI assistant panel
- uses_component from In-app AI assistant panel
- relates_to from In-app AI assistant panel

## AI provider configuration model (ai-provider-configuration-model)

- depends_on from Agent tool contract
- links_to from AI assistant service
- depends_on from AI assistant service
- depends_on from AI secret storage service
- reads from AI assistant session model
- relates_to from AI assistant session model
- relates_to from Product overview
- links_to from Web service host
- depends_on from Web service host
- uses_component from AI settings screen
- uses_component from In-app AI assistant panel
- relates_to from In-app AI assistant panel

## Binary asset model (binary-asset-model)

- links_to from HTML editor plugin runtime
- depends_on from HTML editor plugin runtime
- links_to from Schema validation engine
- depends_on from Schema validation engine
- depends_on from Shared web editing frontend
- links_to from Canonical YAML file layout
- reads from Canonical YAML file layout
- links_to from Table schema model
- links_to from Table schema model
- reads from Table schema model
- links_to from Web service host
- depends_on from Web service host
- relates_to from Table editing workspace

## Canonical YAML file layout (canonical-yaml-file-layout)

- depends_on from Schema validation engine
- uses_component from Generation analysis flow
- relates_to from Generation analysis flow
- uses_component from Generation deletion flow
- relates_to from Generation deletion flow
- uses_component from Generation duplication flow
- relates_to from Generation duplication flow
- relates_to from Generation merge and export flow
- uses_component from Generation persistent merge flow
- relates_to from Generation persistent merge flow
- reads from Binary asset model
- depends_on from Export settings model
- reads from Generation model
- reads from Generic master data model
- relates_to from Generic master data model
- reads from Table schema model
- depends_on from Template generation definition model
- depends_on from Go embedded web server host
- depends_on from Wails desktop host
- relates_to from Web service host
- relates_to from Generation editing screen
- relates_to from Generation selection screen
- relates_to from Schema editing screen

## Editor plugin model (editor-plugin-model)

- links_to from HTML editor plugin runtime
- links_to from HTML editor plugin runtime
- depends_on from HTML editor plugin runtime
- links_to from Shared web editing frontend
- relates_to from Product overview
- links_to from Web service host
- relates_to from Table editing workspace

## Export settings model (export-settings-model)

- uses_component from Go CLI export runner
- relates_to from Go CLI export runner
- depends_on from Export backend adapters
- uses_component from Export execution flow
- invokes from Export execution flow
- relates_to from Export execution flow
- reads from Canonical YAML file layout
- relates_to from Table editing workspace

## Generation model (generation-model)

- uses_component from Export execution flow
- relates_to from Export execution flow
- uses_component from Generation analysis flow
- relates_to from Generation analysis flow
- uses_component from Generation deletion flow
- relates_to from Generation deletion flow
- uses_component from Generation duplication flow
- relates_to from Generation duplication flow
- uses_component from Generation merge and export flow
- relates_to from Generation merge and export flow
- uses_component from Generation persistent merge flow
- relates_to from Generation persistent merge flow
- reads from Canonical YAML file layout
- uses_component from Generation editing screen
- uses_component from Generation selection screen

## Generic master data model (generic-master-data-model)

- depends_on from Export backend adapters
- relates_to from HTML editor plugin runtime
- depends_on from Schema validation engine
- depends_on from Shared web editing frontend
- relates_to from Export execution flow
- uses_component from Generation merge and export flow
- relates_to from Generation merge and export flow
- reads from Binary asset model
- reads from Canonical YAML file layout
- reads from Editor plugin model
- reads from Table schema model
- relates_to from Product overview
- depends_on from Web service host
- relates_to from Schema editing screen
- relates_to from Table editing workspace

## Table schema model (table-schema-model)

- depends_on from Export backend adapters
- relates_to from HTML editor plugin runtime
- depends_on from Schema validation engine
- uses_component from Export execution flow
- relates_to from Export execution flow
- reads from Binary asset model
- reads from Editor plugin model
- reads from Generic master data model
- relates_to from Generic master data model
- depends_on from Template generation definition model
- relates_to from Product overview
- uses_component from Schema editing screen
- relates_to from Table editing workspace
- depends_on from Template generation definition screen

## Template generation definition model (template-export-definition-model)

- relates_to from Go CLI generate runner
- links_to from Pongo2 template generation adapter
- links_to from Pongo2 template generation adapter
- depends_on from Pongo2 template generation adapter
- reads from Canonical YAML file layout
- relates_to from Canonical YAML file layout
- depends_on from Export settings model
- depends_on from Template generation definition screen

## Product overview (product-overview)

- relates_to from Go CLI export runner
- relates_to from AI assistant service
- relates_to from Export backend adapters
- relates_to from Schema validation engine
- relates_to from Export execution flow
- relates_to from Generation merge and export flow
- relates_to from Generic master data model
- depends_on from Go embedded web server host
- links_to from Wails desktop host
- depends_on from Wails desktop host
- depends_on from Web service host
- relates_to from Generation editing screen
- relates_to from Generation selection screen

## Go embedded web server host (go-embedded-web-server-host)

- uses_component from Go CLI export runner
- links_to from Go CLI export runner
- relates_to from Go CLI export runner
- links_to from Go CLI generate runner
- relates_to from Go CLI generate runner
- depends_on from AI assistant service
- relates_to from AI secret storage service
- depends_on from Shared web editing frontend
- relates_to from Product overview
- links_to from Wails desktop host
- depends_on from Wails desktop host
- links_to from Web service host
- links_to from Web service host
- relates_to from Web service host

## Wails desktop host (wails-desktop-host)

- relates_to from Go CLI export runner
- relates_to from Agent tool contract
- depends_on from AI assistant service
- depends_on from AI secret storage service
- depends_on from Shared web editing frontend
- relates_to from Product overview
- relates_to from Go embedded web server host
- relates_to from Web service host
- relates_to from AI settings screen

## Web service host (web-service-host)

- relates_to from Agent tool contract
- depends_on from AI assistant service
- depends_on from AI secret storage service
- depends_on from HTML editor plugin runtime
- depends_on from Shared web editing frontend
- uses_component from Generation analysis flow
- relates_to from Generation analysis flow
- uses_component from Generation deletion flow
- relates_to from Generation deletion flow
- uses_component from Generation duplication flow
- relates_to from Generation duplication flow
- uses_component from Generation persistent merge flow
- relates_to from Generation persistent merge flow
- relates_to from AI assistant session model
- relates_to from Binary asset model
- relates_to from Canonical YAML file layout
- relates_to from Editor plugin model
- relates_to from Generation model
- relates_to from Product overview
- links_to from Go embedded web server host
- links_to from Go embedded web server host
- links_to from Go embedded web server host
- depends_on from Go embedded web server host
- links_to from Wails desktop host
- depends_on from Wails desktop host
- relates_to from Single page application shell
- relates_to from AI settings screen
- uses_component from Generation editing screen
- uses_component from Generation selection screen
- uses_component from Schema editing screen
- uses_component from Table editing workspace

## Single page application shell (single-page-application-shell)

- depends_on from Shared web editing frontend
- relates_to from Product overview
- relates_to from Go embedded web server host
- links_to from Wails desktop host
- relates_to from Wails desktop host
- relates_to from Web service host
- uses_component from AI settings screen
- uses_component from Generation editing screen
- uses_component from Generation selection screen
- uses_component from Schema editing screen
- uses_component from Table editing workspace
- depends_on from Template generation definition screen

## AI settings screen (ai-settings-screen)

- depends_on from AI assistant service
- depends_on from AI secret storage service
- relates_to from Shared web editing frontend
- links_to from AI provider configuration model
- writes from AI provider configuration model
- relates_to from AI provider configuration model
- relates_to from Product overview
- links_to from Go embedded web server host
- links_to from Web service host
- relates_to from Web service host
- links_to from Single page application shell
- relates_to from Single page application shell

## Generation editing screen (generation-editing-screen)

- relates_to from Shared web editing frontend
- uses_component from Generation analysis flow
- relates_to from Generation analysis flow
- uses_component from Generation deletion flow
- relates_to from Generation deletion flow
- uses_component from Generation duplication flow
- relates_to from Generation duplication flow
- uses_component from Generation persistent merge flow
- relates_to from Generation persistent merge flow
- relates_to from Generation model
- relates_to from Wails desktop host
- relates_to from Web service host
- links_to from Single page application shell
- relates_to from Single page application shell
- links_to from Generation selection screen
- relates_to from In-app AI assistant panel
- links_to from Table editing workspace

## Generation selection screen (generation-selection-screen)

- relates_to from Shared web editing frontend
- relates_to from Generation model
- relates_to from Web service host
- links_to from Single page application shell
- relates_to from Single page application shell
- uses_component from Generation editing screen

## In-app AI assistant panel (in-app-ai-assistant-panel)

- relates_to from Agent tool contract
- links_to from AI assistant service
- links_to from AI assistant service
- depends_on from AI assistant service
- relates_to from AI secret storage service
- links_to from Shared web editing frontend
- relates_to from Shared web editing frontend
- reads from AI assistant session model
- relates_to from AI assistant session model
- relates_to from AI provider configuration model
- relates_to from Product overview
- links_to from Web service host
- relates_to from Web service host
- relates_to from AI settings screen

## Schema editing screen (schema-editing-screen)

- relates_to from Shared web editing frontend
- relates_to from Wails desktop host
- relates_to from Web service host
- links_to from Single page application shell
- relates_to from Single page application shell
- relates_to from In-app AI assistant panel
- links_to from Table editing workspace

## Table editing workspace (table-editing-workspace)

- depends_on from Agent tool contract
- depends_on from HTML editor plugin runtime
- relates_to from Schema validation engine
- relates_to from Shared web editing frontend
- relates_to from Binary asset model
- links_to from Editor plugin model
- relates_to from Editor plugin model
- relates_to from Table schema model
- links_to from Product overview
- relates_to from Wails desktop host
- relates_to from Web service host
- links_to from Single page application shell
- relates_to from Single page application shell
- relates_to from In-app AI assistant panel
- relates_to from Schema editing screen

## Template generation definition screen (template-export-definition-screen)

- depends_on from Template generation definition model
- links_to from Single page application shell
- relates_to from Single page application shell
