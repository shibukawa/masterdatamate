package host

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"gopkg.in/yaml.v3"
)

const (
	aiSettingsFileName      = "ai_settings.yaml"
	aiCredentialService     = "MasterDataMate"
	maskedCredentialValue   = "********"
	appleFMServeProfileID   = "apple-fm-serve"
	codexCLIProfileID       = "local-codex"
	defaultAIToolRounds     = 50
	defaultAIToolLoopWindow = 10
)

type aiSettings struct {
	Version       int         `json:"version" yaml:"version"`
	Enabled       bool        `json:"enabled" yaml:"enabled"`
	ActiveProfile string      `json:"active_profile" yaml:"active_profile,omitempty"`
	Profiles      []aiProfile `json:"profiles" yaml:"profiles"`
}

type aiProfile struct {
	ID                  string   `json:"id" yaml:"id"`
	DisplayName         string   `json:"display_name" yaml:"display_name"`
	ProviderType        string   `json:"provider_type" yaml:"provider_type"`
	BaseURL             string   `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	APIKeyRef           string   `json:"api_key_ref,omitempty" yaml:"api_key_ref,omitempty"`
	Model               string   `json:"model,omitempty" yaml:"model,omitempty"`
	Command             string   `json:"command,omitempty" yaml:"command,omitempty"`
	Args                []string `json:"args,omitempty" yaml:"args,omitempty"`
	StdinMode           string   `json:"stdin_mode,omitempty" yaml:"stdin_mode,omitempty"`
	StdoutFormat        string   `json:"stdout_format,omitempty" yaml:"stdout_format,omitempty"`
	Sandbox             string   `json:"sandbox,omitempty" yaml:"sandbox,omitempty"`
	Temperature         *float64 `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	MaxOutputTokens     int      `json:"max_output_tokens,omitempty" yaml:"max_output_tokens,omitempty"`
	RequestTimeoutMS    int      `json:"request_timeout_ms,omitempty" yaml:"request_timeout_ms,omitempty"`
	MaxToolRounds       int      `json:"max_tool_rounds,omitempty" yaml:"max_tool_rounds,omitempty"`
	ToolLoopWindow      int      `json:"tool_loop_window,omitempty" yaml:"tool_loop_window,omitempty"`
	SupportsStreaming   bool     `json:"supports_streaming" yaml:"supports_streaming"`
	SupportsToolCalls   bool     `json:"supports_tool_calls" yaml:"supports_tool_calls"`
	LocalNetworkAllowed bool     `json:"local_network_allowed,omitempty" yaml:"local_network_allowed,omitempty"`
	RequiresAPIKey      bool     `json:"requires_api_key,omitempty" yaml:"requires_api_key,omitempty"`
	HasAPIKey           bool     `json:"has_api_key,omitempty" yaml:"-"`
	APIKey              string   `json:"api_key,omitempty" yaml:"-"`
	ClearAPIKey         bool     `json:"clear_api_key,omitempty" yaml:"-"`
	Notes               string   `json:"notes,omitempty" yaml:"notes,omitempty"`
	HealthStatus        string   `json:"health_status,omitempty" yaml:"-"`
	HealthMessage       string   `json:"health_message,omitempty" yaml:"-"`
}

type localProviderStatus struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Available bool   `json:"available"`
	Command   string `json:"command,omitempty"`
	Message   string `json:"message,omitempty"`
}

type aiModelInfo struct {
	ID string `json:"id"`
}

type credentialStore interface {
	Set(profileID string, secret string) error
	Get(profileID string) (string, error)
	Delete(profileID string) error
	Has(profileID string) (bool, error)
}

type keyringCredentialStore struct {
	ring keyring.Keyring
}

func newKeyringCredentialStore() (credentialStore, error) {
	allowed := []keyring.BackendType{
		keyring.KeychainBackend,
		keyring.WinCredBackend,
		keyring.SecretServiceBackend,
		keyring.KWalletBackend,
		keyring.KeyCtlBackend,
		keyring.PassBackend,
	}
	ring, err := keyring.Open(keyring.Config{
		ServiceName:              aiCredentialService,
		KeychainName:             "login",
		KeychainTrustApplication: true,
		KeychainSynchronizable:   false,
		KWalletAppID:             aiCredentialService,
		KWalletFolder:            aiCredentialService,
		LibSecretCollectionName:  "default",
		WinCredPrefix:            aiCredentialService,
		AllowedBackends:          allowed,
	})
	if err != nil {
		return nil, err
	}
	return &keyringCredentialStore{ring: ring}, nil
}

func credentialKey(profileID string) string {
	return "ai-provider:" + profileID + ":api-key"
}

func (s *keyringCredentialStore) Set(profileID string, secret string) error {
	return s.ring.Set(keyring.Item{
		Key:         credentialKey(profileID),
		Data:        []byte(secret),
		Label:       "MasterDataMate AI provider API key",
		Description: "API key for MasterDataMate AI provider profile " + profileID,
	})
}

func (s *keyringCredentialStore) Get(profileID string) (string, error) {
	item, err := s.ring.Get(credentialKey(profileID))
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

func (s *keyringCredentialStore) Delete(profileID string) error {
	err := s.ring.Remove(credentialKey(profileID))
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return nil
	}
	return err
}

func (s *keyringCredentialStore) Has(profileID string) (bool, error) {
	_, err := s.ring.GetMetadata(credentialKey(profileID))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, keyring.ErrMetadataNotSupported) || errors.Is(err, keyring.ErrMetadataNeedsCredentials) {
		_, getErr := s.ring.Get(credentialKey(profileID))
		if getErr == nil {
			return true, nil
		}
		if errors.Is(getErr, keyring.ErrKeyNotFound) {
			return false, nil
		}
		return false, getErr
	}
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return false, nil
	}
	return false, err
}

func (s *server) dispatchAIAPI(r *http.Request, parts []string) (int, any, string, []byte, error) {
	if len(parts) >= 3 && parts[2] == "sessions" {
		return s.dispatchAISessionAPI(r, parts)
	}
	if len(parts) == 3 && parts[2] == "runs" && r.Method == http.MethodPost {
		var body aiRunRequest
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.runManagedAIChat(r.Context(), body)
		return 200, payload, "", nil, err
	}
	if len(parts) == 4 && parts[2] == "tools" && r.Method == http.MethodPost {
		var body map[string]any
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.executeAITool(parts[3], body)
		return 200, payload, "", nil, err
	}
	if len(parts) == 3 && parts[2] == "settings" {
		switch r.Method {
		case http.MethodGet:
			payload, err := s.loadAISettingsResponse()
			return 200, payload, "", nil, err
		case http.MethodPut:
			var body aiSettings
			if err := readJSON(r, &body); err != nil {
				return 0, nil, "", nil, err
			}
			payload, err := s.saveAISettings(body)
			return 200, payload, "", nil, err
		default:
			return 405, nil, "", nil, appError{405, "Method not allowed"}
		}
	}
	if len(parts) == 3 && parts[2] == "local-providers" && r.Method == http.MethodGet {
		return 200, map[string]any{"providers": detectLocalAIProviders()}, "", nil, nil
	}
	if len(parts) == 3 && parts[2] == "profiles" && r.Method == http.MethodGet {
		payload, err := s.loadAISettingsResponse()
		return 200, map[string]any{"profiles": payload.Profiles, "active_profile": payload.ActiveProfile}, "", nil, err
	}
	if len(parts) == 5 && parts[2] == "profiles" && parts[4] == "health" && r.Method == http.MethodPost {
		payload, err := s.checkAIProfileHealth(parts[3])
		return 200, payload, "", nil, err
	}
	if len(parts) == 5 && parts[2] == "profiles" && parts[4] == "models" && r.Method == http.MethodGet {
		payload, err := s.listAIProfileModels(parts[3])
		return 200, payload, "", nil, err
	}
	if len(parts) == 5 && parts[2] == "profiles" && parts[4] == "credential" && r.Method == http.MethodDelete {
		store, err := newKeyringCredentialStore()
		if err != nil {
			return 0, nil, "", nil, appError{503, "Credential storage is unavailable: " + err.Error()}
		}
		if err := store.Delete(parts[3]); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.loadAISettingsResponse()
		return 200, payload, "", nil, err
	}
	return 404, nil, "", nil, appError{404, "API route not found"}
}

func (s *server) handleAIRunStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method not allowed"})
		return
	}
	var body aiRunRequest
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	writeEvent := func(event map[string]any) {
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		_, _ = w.Write(append(data, '\n'))
		if flusher != nil {
			flusher.Flush()
		}
	}
	body.DebugSink = func(event map[string]any) {
		writeEvent(map[string]any{"kind": "debug_event", "event": event})
	}
	body.FrontendToolSink = func(ctx context.Context, call aiFrontendToolCall) (map[string]any, error) {
		resultChan := s.registerAIFrontendToolRequest(call.RequestID)
		defer s.unregisterAIFrontendToolRequest(call.RequestID)
		writeEvent(map[string]any{"kind": "frontend_tool_request", "request": call})
		select {
		case result := <-resultChan:
			if result.Error != "" {
				return map[string]any{"success": false, "status": "error", "error": result.Error}, nil
			}
			if result.Result == nil {
				return map[string]any{"success": false, "status": "error", "error": "Frontend tool returned no result."}, nil
			}
			return result.Result, nil
		case <-ctx.Done():
			if call.Name == "get_current_context" && len(body.Context) > 0 {
				return map[string]any{"success": true, "status": "fallback", "summary": "Used request-time frontend context fallback.", "context": body.Context}, nil
			}
			if call.Name == "stage_table_changes" {
				return map[string]any{"success": false, "status": "pending_frontend_staging", "error": ctx.Err().Error(), "change_set": call.Arguments}, nil
			}
			return map[string]any{"success": false, "status": "blocked", "error": ctx.Err().Error()}, nil
		case <-time.After(5 * time.Second):
			if call.Name == "get_current_context" && len(body.Context) > 0 {
				return map[string]any{"success": true, "status": "fallback", "summary": "Used request-time frontend context fallback.", "context": body.Context}, nil
			}
			if call.Name == "stage_table_changes" {
				return map[string]any{"success": false, "status": "pending_frontend_staging", "error": "Timed out waiting for frontend tool result.", "change_set": call.Arguments}, nil
			}
			return map[string]any{"success": false, "status": "blocked", "error": "Timed out waiting for frontend tool result."}, nil
		}
	}
	payload, err := s.runManagedAIChat(r.Context(), body)
	if err != nil {
		writeEvent(map[string]any{"kind": "error", "error": err.Error(), "payload": payload})
		return
	}
	writeEvent(map[string]any{"kind": "final", "payload": payload})
}

func (s *server) handleAIFrontendToolResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method not allowed"})
		return
	}
	var body struct {
		RequestID string         `json:"request_id"`
		Result    map[string]any `json:"result"`
		Error     string         `json:"error"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.RequestID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "request_id is required"})
		return
	}
	if ok := s.deliverAIFrontendToolResult(body.RequestID, aiFrontendToolResult{Result: body.Result, Error: body.Error}); !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"delivered": false,
			"status":    "expired",
			"summary":   "Frontend tool request was already completed or timed out.",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"delivered": true})
}

func (s *server) registerAIFrontendToolRequest(requestID string) <-chan aiFrontendToolResult {
	resultChan := make(chan aiFrontendToolResult, 1)
	s.aiFrontendMu.Lock()
	if s.aiFrontendWait == nil {
		s.aiFrontendWait = map[string]chan aiFrontendToolResult{}
	}
	s.aiFrontendWait[requestID] = resultChan
	s.aiFrontendMu.Unlock()
	return resultChan
}

func (s *server) unregisterAIFrontendToolRequest(requestID string) {
	s.aiFrontendMu.Lock()
	delete(s.aiFrontendWait, requestID)
	s.aiFrontendMu.Unlock()
}

func (s *server) deliverAIFrontendToolResult(requestID string, result aiFrontendToolResult) bool {
	s.aiFrontendMu.Lock()
	resultChan := s.aiFrontendWait[requestID]
	s.aiFrontendMu.Unlock()
	if resultChan == nil {
		return false
	}
	select {
	case resultChan <- result:
		return true
	default:
		return false
	}
}

func (s *server) aiSettingsPath() string {
	return filepath.Join(s.root, "masterdata", aiSettingsFileName)
}

func (s *server) loadAISettingsResponse() (aiSettings, error) {
	settings, err := s.loadAISettings()
	if err != nil {
		return aiSettings{}, err
	}
	settings = applyAIDefaults(settings)
	settings = normalizeAISettings(settings)
	_ = s.attachCredentialPresence(&settings)
	settings = attachProfileHealthHints(settings)
	return settings, nil
}

func (s *server) loadAISettings() (aiSettings, error) {
	var settings aiSettings
	data, err := osReadFileIfExists(s.aiSettingsPath())
	if err != nil {
		return aiSettings{}, err
	}
	if len(data) == 0 {
		return applyAIDefaults(aiSettings{Version: 1}), nil
	}
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return aiSettings{}, err
	}
	if settings.Version == 0 {
		settings.Version = 1
	}
	return settings, nil
}

func osReadFileIfExists(file string) ([]byte, error) {
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

func (s *server) saveAISettings(settings aiSettings) (aiSettings, error) {
	if settings.Version == 0 {
		settings.Version = 1
	}
	settings = normalizeAISettings(settings)
	store, storeErr := newKeyringCredentialStore()
	for index := range settings.Profiles {
		profile := &settings.Profiles[index]
		if profile.APIKey != "" && profile.APIKey != maskedCredentialValue {
			if storeErr != nil {
				return aiSettings{}, appError{503, "Credential storage is unavailable: " + storeErr.Error()}
			}
			if err := store.Set(profile.ID, profile.APIKey); err != nil {
				return aiSettings{}, err
			}
			profile.APIKeyRef = credentialKey(profile.ID)
		}
		if profile.ClearAPIKey {
			if storeErr != nil {
				return aiSettings{}, appError{503, "Credential storage is unavailable: " + storeErr.Error()}
			}
			if err := store.Delete(profile.ID); err != nil {
				return aiSettings{}, err
			}
			profile.APIKeyRef = ""
		}
		profile.APIKey = ""
		profile.ClearAPIKey = false
		profile.HasAPIKey = false
	}
	if settings.ActiveProfile == "" && len(settings.Profiles) > 0 {
		settings.ActiveProfile = settings.Profiles[0].ID
	}
	if err := writeYAMLFile(s.aiSettingsPath(), settings); err != nil {
		return aiSettings{}, err
	}
	return s.loadAISettingsResponse()
}

func normalizeAISettings(settings aiSettings) aiSettings {
	seen := map[string]bool{}
	profiles := make([]aiProfile, 0, len(settings.Profiles))
	for _, profile := range settings.Profiles {
		profile.ID = strings.TrimSpace(profile.ID)
		if profile.ID == "" || seen[profile.ID] {
			continue
		}
		seen[profile.ID] = true
		profile.DisplayName = strings.TrimSpace(profile.DisplayName)
		if profile.DisplayName == "" {
			profile.DisplayName = profile.ID
		}
		profile.ProviderType = strings.TrimSpace(profile.ProviderType)
		profile.APIKey = strings.TrimSpace(profile.APIKey)
		if profile.APIKeyRef == "" && profile.RequiresAPIKey {
			profile.APIKeyRef = credentialKey(profile.ID)
		}
		profile = normalizeAIToolLimits(profile)
		profile = normalizeBuiltinAIProfile(profile)
		if !isAIProfileVisible(profile) {
			continue
		}
		profiles = append(profiles, profile)
	}
	settings.Profiles = profiles
	if settings.ActiveProfile != "" {
		found := false
		for _, profile := range profiles {
			if profile.ID == settings.ActiveProfile {
				found = true
				break
			}
		}
		if !found {
			settings.ActiveProfile = ""
		}
	}
	if settings.ActiveProfile == "" && len(settings.Profiles) > 0 {
		settings.ActiveProfile = settings.Profiles[0].ID
	}
	return settings
}

func normalizeAIToolLimits(profile aiProfile) aiProfile {
	if profile.MaxToolRounds <= 0 {
		profile.MaxToolRounds = defaultAIToolRounds
	}
	if profile.MaxToolRounds < 1 {
		profile.MaxToolRounds = 1
	}
	if profile.MaxToolRounds > 500 {
		profile.MaxToolRounds = 500
	}
	if profile.ToolLoopWindow <= 0 {
		profile.ToolLoopWindow = defaultAIToolLoopWindow
	}
	if profile.ToolLoopWindow < 1 {
		profile.ToolLoopWindow = 1
	}
	if profile.ToolLoopWindow > profile.MaxToolRounds {
		profile.ToolLoopWindow = profile.MaxToolRounds
	}
	if profile.ToolLoopWindow > 100 {
		profile.ToolLoopWindow = 100
	}
	return profile
}

func applyAIDefaults(settings aiSettings) aiSettings {
	if settings.Version == 0 {
		settings.Version = 1
	}
	if len(settings.Profiles) == 0 {
		settings.Profiles = defaultAIProfiles()
	}
	if settings.ActiveProfile == "" {
		if fmAvailable() {
			settings.Enabled = true
			settings.ActiveProfile = appleFMServeProfileID
		}
	}
	return settings
}

func defaultAIProfiles() []aiProfile {
	profiles := []aiProfile{}
	if fmCommandAvailable() {
		profiles = append(profiles,
			aiProfile{
				ID:                appleFMServeProfileID,
				DisplayName:       "Apple Foundation Models server",
				ProviderType:      "openai_compatible",
				BaseURL:           "http://127.0.0.1:1976",
				Model:             "system",
				RequiresAPIKey:    false,
				SupportsStreaming: true,
				SupportsToolCalls: true,
				Notes:             "Backed by fm serve. Start fm serve before using this profile.",
			},
		)
	}
	profiles = append(profiles,
		aiProfile{
			ID:                "openai-default",
			DisplayName:       "OpenAI",
			ProviderType:      "openai",
			BaseURL:           "https://api.openai.com/v1",
			Model:             "gpt-4.1-mini",
			RequiresAPIKey:    true,
			APIKeyRef:         credentialKey("openai-default"),
			SupportsStreaming: true,
			SupportsToolCalls: true,
		},
		aiProfile{
			ID:                "local-ollama",
			DisplayName:       "Ollama",
			ProviderType:      "ollama",
			BaseURL:           "http://127.0.0.1:11434",
			Model:             "llama3.1",
			SupportsStreaming: true,
			SupportsToolCalls: false,
		},
		aiProfile{
			ID:                "lmstudio-local",
			DisplayName:       "LM Studio",
			ProviderType:      "lmstudio",
			BaseURL:           "http://127.0.0.1:1234",
			Model:             "local-model",
			SupportsStreaming: true,
			SupportsToolCalls: true,
		},
		aiProfile{
			ID:                codexCLIProfileID,
			DisplayName:       "Codex CLI",
			ProviderType:      "codex_cli",
			Command:           "codex",
			Args:              []string{"exec", "--json"},
			StdinMode:         "prompt",
			StdoutFormat:      "jsonl",
			Sandbox:           "read_only",
			SupportsStreaming: true,
			SupportsToolCalls: true,
		},
	)
	return profiles
}

func normalizeBuiltinAIProfile(profile aiProfile) aiProfile {
	switch profile.ID {
	case appleFMServeProfileID:
		profile.DisplayName = "Apple Foundation Models server"
		profile.ProviderType = "openai_compatible"
		if profile.BaseURL == "" {
			profile.BaseURL = "http://127.0.0.1:1976"
		}
		if profile.Model == "" {
			profile.Model = "system"
		}
		profile.Command = ""
		profile.Args = nil
		profile.StdinMode = ""
		profile.StdoutFormat = ""
		profile.Sandbox = ""
		profile.RequiresAPIKey = false
		profile.APIKeyRef = ""
		profile.APIKey = ""
		profile.ClearAPIKey = false
		profile.SupportsStreaming = true
		profile.SupportsToolCalls = true
	case codexCLIProfileID:
		profile.DisplayName = "Codex CLI"
		profile.ProviderType = "codex_cli"
		profile.BaseURL = ""
		profile.Model = ""
		profile.Command = "codex"
		profile.Args = []string{"exec", "--json"}
		profile.StdinMode = "prompt"
		profile.StdoutFormat = "jsonl"
		profile.Sandbox = "read_only"
		profile.RequiresAPIKey = false
		profile.APIKeyRef = ""
		profile.APIKey = ""
		profile.ClearAPIKey = false
		profile.SupportsStreaming = true
		profile.SupportsToolCalls = true
	}
	return profile
}

func isAIProfileVisible(profile aiProfile) bool {
	return profile.ID != appleFMServeProfileID || fmCommandAvailable()
}

func (s *server) attachCredentialPresence(settings *aiSettings) error {
	store, err := newKeyringCredentialStore()
	for index := range settings.Profiles {
		profile := &settings.Profiles[index]
		profile.APIKey = ""
		profile.ClearAPIKey = false
		if profile.RequiresAPIKey && profile.APIKeyRef == "" {
			profile.APIKeyRef = credentialKey(profile.ID)
		}
		if err != nil {
			profile.HasAPIKey = false
			continue
		}
		has, hasErr := store.Has(profile.ID)
		if hasErr != nil {
			profile.HasAPIKey = false
			continue
		}
		profile.HasAPIKey = has
	}
	return nil
}

func attachProfileHealthHints(settings aiSettings) aiSettings {
	local := detectLocalAIProviders()
	available := map[string]localProviderStatus{}
	for _, item := range local {
		available[item.ID] = item
	}
	for index := range settings.Profiles {
		profile := &settings.Profiles[index]
		switch profile.ID {
		case appleFMServeProfileID:
			if available["fm"].Available {
				profile.HealthStatus = "available"
				profile.HealthMessage = "fm command is available. Start fm serve before running chat."
			} else {
				profile.HealthStatus = "unavailable"
				profile.HealthMessage = available["fm"].Message
			}
		case codexCLIProfileID:
			if available["codex"].Available {
				profile.HealthStatus = "available"
			} else {
				profile.HealthStatus = "unavailable"
				profile.HealthMessage = available["codex"].Message
			}
		}
	}
	return settings
}

func detectLocalAIProviders() []localProviderStatus {
	providers := []localProviderStatus{}
	if fmCommandAvailable() {
		providers = append(providers, detectCommandProvider("fm", "foundation_models_cli"))
	}
	providers = append(providers,
		detectCommandProvider("ollama", "ollama"),
		detectCommandProvider("lmstudio", "lmstudio"),
		detectCommandProvider("codex", "codex_cli"),
	)
	return providers
}

func detectCommandProvider(command string, kind string) localProviderStatus {
	path, err := exec.LookPath(command)
	status := localProviderStatus{ID: command, Kind: kind, Command: command}
	if err != nil {
		status.Message = command + " command not found"
		return status
	}
	status.Available = true
	status.Command = path
	if command == "fm" && runtime.GOOS == "darwin" {
		if !fmAvailable() {
			status.Available = false
			status.Message = "fm command found but system model is not available"
		}
	}
	return status
}

func fmAvailable() bool {
	if !fmCommandAvailable() {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "fm", "available", "--model", "system")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func fmCommandAvailable() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := exec.LookPath("fm")
	return err == nil
}

func (s *server) checkAIProfileHealth(profileID string) (map[string]any, error) {
	profile, err := s.findAIProfile(profileID)
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"profileId": profile.ID,
		"ok":        false,
		"checks":    []map[string]any{},
	}
	checks := []map[string]any{}
	addCheck := func(name string, ok bool, message string) {
		checks = append(checks, map[string]any{"name": name, "ok": ok, "message": message})
	}
	if profile.RequiresAPIKey && !profile.HasAPIKey {
		addCheck("credential", false, "API key is not saved.")
		result["checks"] = checks
		return result, nil
	}
	switch profile.ProviderType {
	case "openai_compatible", "openai", "ollama", "lmstudio":
		baseURL := profile.BaseURL
		if baseURL == "" && profile.ProviderType == "openai" {
			baseURL = "https://api.openai.com/v1"
		}
		ok, message := checkHTTPProvider(baseURL)
		addCheck("endpoint", ok, message)
		result["ok"] = ok
	case "codex_cli", "foundation_models_cli":
		command := profile.Command
		if command == "" {
			if profile.ProviderType == "codex_cli" {
				command = "codex"
			} else {
				command = "fm"
			}
		}
		_, err := exec.LookPath(command)
		ok := err == nil
		message := "command is available"
		if err != nil {
			message = "command not found"
		}
		addCheck("command", ok, message)
		result["ok"] = ok
	default:
		addCheck("provider", false, "Unsupported provider type.")
	}
	result["checks"] = checks
	return result, nil
}

func (s *server) findAIProfile(profileID string) (*aiProfile, error) {
	settings, err := s.loadAISettingsResponse()
	if err != nil {
		return nil, err
	}
	for index := range settings.Profiles {
		if settings.Profiles[index].ID == profileID {
			return &settings.Profiles[index], nil
		}
	}
	return nil, appError{404, "AI provider profile not found."}
}

func (s *server) listAIProfileModels(profileID string) (map[string]any, error) {
	profile, err := s.findAIProfile(profileID)
	if err != nil {
		return nil, err
	}
	switch profile.ProviderType {
	case "openai_compatible", "openai", "lmstudio":
		models, err := fetchOpenAICompatibleModels(profile.BaseURL)
		if err != nil {
			return nil, err
		}
		return map[string]any{"models": models}, nil
	default:
		return map[string]any{"models": []aiModelInfo{}}, nil
	}
}

func checkHTTPProvider(baseURL string) (bool, string) {
	if baseURL == "" {
		return false, "Base URL is required."
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false, "Base URL is invalid."
	}
	healthURL := strings.TrimRight(baseURL, "/") + "/health"
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(healthURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return true, "Health endpoint is reachable."
		}
	}
	modelsURL := openAICompatibleModelsURL(baseURL)
	resp, err = client.Get(modelsURL)
	if err != nil {
		return false, "Endpoint is not reachable: " + err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return true, "Model endpoint is reachable."
	}
	return false, "Endpoint returned status " + resp.Status
}

func fetchOpenAICompatibleModels(baseURL string) ([]aiModelInfo, error) {
	if baseURL == "" {
		return nil, appError{400, "Base URL is required."}
	}
	modelsURL := openAICompatibleModelsURL(baseURL)
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(modelsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, appError{resp.StatusCode, "Model endpoint returned status " + resp.Status}
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	models := make([]aiModelInfo, 0, len(body.Data))
	for _, item := range body.Data {
		if item.ID != "" {
			models = append(models, aiModelInfo{ID: item.ID})
		}
	}
	return models, nil
}

func openAICompatibleModelsURL(baseURL string) string {
	trimmed := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/models"
	}
	return trimmed + "/v1/models"
}

func writeYAMLFile(file string, value any) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o600)
}
