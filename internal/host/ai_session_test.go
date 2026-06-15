package host

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAISessionCRUDUsesUserConfigStore(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\ndescription: Initial\n")

	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	session, err := s.createAISession("Test chat", "managed_chat_agent", appleFMServeProfileID)
	if err != nil {
		t.Fatalf("createAISession() error = %v", err)
	}
	if session.WorkspaceID == "" {
		t.Fatal("workspace id is empty")
	}
	sessionFile := s.aiSessionFile(session.SessionID)
	if _, err := os.Stat(sessionFile); err != nil {
		t.Fatalf("expected session file under user config: %v", err)
	}
	if filepath.Clean(sessionFile) == filepath.Clean(root) || !filepath.IsAbs(sessionFile) {
		t.Fatalf("unexpected session path %q", sessionFile)
	}
	if rel, err := filepath.Rel(root, sessionFile); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("session file should not be inside workspace: %s", sessionFile)
	}
	loaded, err := s.loadAISession(session.SessionID)
	if err != nil {
		t.Fatalf("loadAISession() error = %v", err)
	}
	if loaded.Title != "Test chat" {
		t.Fatalf("loaded title = %q", loaded.Title)
	}
	items, err := s.listAISessions()
	if err != nil {
		t.Fatalf("listAISessions() error = %v", err)
	}
	if len(items) != 1 || items[0].SessionID != session.SessionID {
		t.Fatalf("sessions = %#v", items)
	}
	if err := s.deleteAISession(session.SessionID); err != nil {
		t.Fatalf("deleteAISession() error = %v", err)
	}
	items, err = s.listAISessions()
	if err != nil {
		t.Fatalf("listAISessions() after delete error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("deleted session should be hidden, got %#v", items)
	}
}

func TestRunManagedAIChatCallsOpenAICompatibleProviderAndPersistsMessages(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	var received openAIChatRequest
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected provider path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode provider request: %v", err)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "検証エラーはありません。"}},
			},
			"usage": map[string]any{"prompt_tokens": 12, "completion_tokens": 6, "total_tokens": 18},
		})
	}))
	defer provider.Close()

	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\ndescription: Initial\n")
	writeFile(t, filepath.Join(root, "masterdata", aiSettingsFileName), `version: 1
enabled: true
active_profile: test-provider
profiles:
  - id: test-provider
    display_name: Test Provider
    provider_type: openai_compatible
    base_url: `+provider.URL+`
    model: system
    supports_streaming: false
    supports_tool_calls: false
`)

	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := s.runManagedAIChat(t.Context(), aiRunRequest{
		Message:   "このテーブルを確認して",
		ProfileID: "test-provider",
		Context: map[string]any{
			"selectedTable": "product",
			"visibleRows":   []map[string]any{{"product_id": "prod-core"}},
		},
	})
	if err != nil {
		t.Fatalf("runManagedAIChat() error = %v", err)
	}
	session := payload["session"].(aiSession)
	if len(session.Messages) != 2 {
		t.Fatalf("message count = %d", len(session.Messages))
	}
	if session.Messages[1].Content != "検証エラーはありません。" {
		t.Fatalf("assistant content = %q", session.Messages[1].Content)
	}
	if received.Model != "system" {
		t.Fatalf("provider model = %q", received.Model)
	}
	if len(received.Messages) < 2 {
		t.Fatalf("provider messages = %#v", received.Messages)
	}
	requestData, _ := json.Marshal(received)
	if strings.Contains(string(requestData), "prod-core") {
		t.Fatalf("initial provider request should not include frontend context: %s", string(requestData))
	}
	loaded, err := s.loadAISession(session.SessionID)
	if err != nil {
		t.Fatalf("load persisted session: %v", err)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("persisted message count = %d", len(loaded.Messages))
	}
	logFile, err := s.aiSessionLogFile(session.SessionID, session.Runs[0].ID)
	if err != nil {
		t.Fatalf("aiSessionLogFile() error = %v", err)
	}
	kinds := readLogKinds(t, logFile)
	if !containsString(kinds, "request") || !containsString(kinds, "response") || !containsString(kinds, "assistant_message") {
		t.Fatalf("debug log kinds = %#v", kinds)
	}
}

func TestAIRunStreamReturnsDebugEventsAndFinalPayload(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "Done"}},
			},
		})
	}))
	defer provider.Close()

	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\ndescription: Initial\n")
	writeFile(t, filepath.Join(root, "masterdata", aiSettingsFileName), `version: 1
enabled: true
active_profile: test-provider
profiles:
  - id: test-provider
    display_name: Test Provider
    provider_type: openai_compatible
    base_url: `+provider.URL+`
    model: system
    supports_streaming: false
    supports_tool_calls: false
`)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"profileId":"test-provider","message":"hello","context":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ai/runs/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	s.routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	events := decodeNDJSONEvents(t, resp.Body.Bytes())
	if !hasStreamKind(events, "debug_event") || !hasStreamKind(events, "final") {
		t.Fatalf("stream events = %#v", events)
	}
}

func TestRunManagedAIChatBrokersGetCurrentContextAsFrontendTool(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	requestCount := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var received openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode provider request: %v", err)
		}
		if requestCount == 1 {
			data, _ := json.Marshal(received)
			if strings.Contains(string(data), "selectedTable") {
				t.Fatalf("first request leaked frontend context: %s", string(data))
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{
							{
								"id":   "call-context",
								"type": "function",
								"function": map[string]any{
									"name":      "get_current_context",
									"arguments": `{}`,
								},
							},
						},
					}},
				},
			})
			return
		}
		foundContext := false
		for _, message := range received.Messages {
			if message.Role == "tool" && message.ToolCallID == "call-context" && strings.Contains(message.Content, "slime") {
				foundContext = true
			}
		}
		if !foundContext {
			t.Fatalf("second request did not include frontend context tool result: %#v", received.Messages)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "スライムの状態を確認しました。"}},
			},
		})
	}))
	defer provider.Close()

	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\ndescription: Initial\n")
	writeFile(t, filepath.Join(root, "masterdata", aiSettingsFileName), `version: 1
enabled: true
active_profile: test-provider
profiles:
  - id: test-provider
    display_name: Test Provider
    provider_type: openai_compatible
    base_url: `+provider.URL+`
    model: system
    supports_streaming: false
    supports_tool_calls: true
`)

	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := s.runManagedAIChat(t.Context(), aiRunRequest{
		Message:   "現在の表を見て",
		ProfileID: "test-provider",
		FrontendToolSink: func(_ context.Context, call aiFrontendToolCall) (map[string]any, error) {
			if call.Name != "get_current_context" {
				t.Fatalf("unexpected frontend tool %s", call.Name)
			}
			return map[string]any{
				"success": true,
				"status":  "ok",
				"context": map[string]any{
					"selectedTable": "enemy",
					"visibleRows": []map[string]any{
						{"enemy_id": "slime", "defense": 6},
					},
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("runManagedAIChat() error = %v", err)
	}
	session := payload["session"].(aiSession)
	if session.Messages[1].Content != "スライムの状態を確認しました。" {
		t.Fatalf("assistant content = %q", session.Messages[1].Content)
	}
	if requestCount != 2 {
		t.Fatalf("provider request count = %d", requestCount)
	}
}

func TestAppleFMChatUsesInitialContextAndOmitsContextTool(t *testing.T) {
	var received openAIChatRequest
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode provider request: %v", err)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "見えている敵は slime です。"}},
			},
		})
	}))
	defer provider.Close()

	root := testBinaryWorkspace(t)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	session := aiSession{
		Messages: []aiSessionMessage{{
			ID:        "msg-user",
			Role:      "user",
			Content:   "すべての敵はだれ？",
			CreatedAt: nowString(),
		}},
	}
	contextPayload := map[string]any{
		"selectedTable": "enemy",
		"visibleRows": []map[string]any{
			{"enemy_id": "slime", "name": "Slime"},
		},
	}
	_, _, _, err = s.callOpenAICompatibleChat(t.Context(), aiProfile{
		ID:                appleFMServeProfileID,
		ProviderType:      "openai_compatible",
		BaseURL:           provider.URL,
		Model:             "system",
		SupportsToolCalls: true,
	}, buildManagedChatMessages(session, contextPayload), contextPayload, nil, nil)
	if err != nil {
		t.Fatalf("callOpenAICompatibleChat() error = %v", err)
	}
	requestData, _ := json.Marshal(received)
	if !strings.Contains(string(requestData), "selectedTable") || !strings.Contains(string(requestData), "slime") {
		t.Fatalf("initial Apple FM request should include scoped context: %s", string(requestData))
	}
	for _, tool := range received.Tools {
		if tool.Function.Name == "get_current_context" {
			t.Fatalf("Apple FM request with initial context should omit get_current_context tool: %#v", received.Tools)
		}
	}
}

func TestRunManagedAIChatReportsRejectedFrontendStage(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{
					"role": "assistant",
					"tool_calls": []map[string]any{
						{
							"id":   "call-stage",
							"type": "function",
							"function": map[string]any{
								"name":      "stage_table_changes",
								"arguments": `{"tableId":"enemy","generationId":"0000_initial","operations_json":"[{\"op\":\"update\",\"key\":\"missing\",\"values\":{\"hp\":1}}]"}`,
							},
						},
					},
				}},
			},
		})
	}))
	defer provider.Close()

	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\ndescription: Initial\n")
	writeFile(t, filepath.Join(root, "masterdata", aiSettingsFileName), `version: 1
enabled: true
active_profile: test-provider
profiles:
  - id: test-provider
    display_name: Test Provider
    provider_type: openai_compatible
    base_url: `+provider.URL+`
    model: system
    supports_streaming: false
    supports_tool_calls: true
`)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := s.runManagedAIChat(t.Context(), aiRunRequest{
		Message:   "missingを更新して",
		ProfileID: "test-provider",
		FrontendToolSink: func(_ context.Context, call aiFrontendToolCall) (map[string]any, error) {
			return map[string]any{
				"success":  false,
				"status":   "rejected",
				"summary":  "Staged 0 operation(s); 1 rejected.",
				"accepted": []any{},
				"rejected": []any{map[string]any{"index": float64(0), "reason": "target primary key was not found"}},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("runManagedAIChat() error = %v", err)
	}
	if _, ok := payload["stage_table_changes"]; ok {
		t.Fatalf("rejected frontend stage should not be returned for final staging: %#v", payload["stage_table_changes"])
	}
	session := payload["session"].(aiSession)
	if !strings.Contains(session.Messages[1].Content, "target primary key was not found") {
		t.Fatalf("assistant content should include reject reason, got %q", session.Messages[1].Content)
	}
}

func TestRunManagedAIChatAllowsContextReadValidateThenStage(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	requestCount := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var received openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode provider request: %v", err)
		}
		switch requestCount {
		case 1:
			writeJSON(w, http.StatusOK, map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{{
							"id":   "call-context",
							"type": "function",
							"function": map[string]any{
								"name":      "get_current_context",
								"arguments": `{}`,
							},
						}},
					}},
				},
			})
		case 2:
			writeJSON(w, http.StatusOK, map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{{
							"id":   "call-table",
							"type": "function",
							"function": map[string]any{
								"name":      "get_table",
								"arguments": `{"tableId":"product","generationId":"0000_initial","mode":"active_only","offset":0,"limit":5,"fields":["product_id"]}`,
							},
						}},
					}},
				},
			})
		case 3:
			writeJSON(w, http.StatusOK, map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{{
							"id":   "call-validate",
							"type": "function",
							"function": map[string]any{
								"name":      "validate_table",
								"arguments": `{"tableId":"product","generationId":"0000_initial","mode":"active_only","rows_json":"[]"}`,
							},
						}},
					}},
				},
			})
		case 4:
			foundValidateResult := false
			for _, message := range received.Messages {
				if message.Role == "tool" && message.ToolCallID == "call-validate" {
					foundValidateResult = true
				}
			}
			if !foundValidateResult {
				t.Fatalf("fourth request did not include validate tool result: %#v", received.Messages)
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{{
							"id":   "call-stage",
							"type": "function",
							"function": map[string]any{
								"name":      "stage_table_changes",
								"arguments": `{"tableId":"product","generationId":"0000_initial","operations_json":"[{\"op\":\"update\",\"key\":\"prod-core\",\"values\":{\"product_id\":\"prod-core\"}}]"}`,
							},
						}},
					}},
				},
			})
		default:
			t.Fatalf("unexpected provider request count %d", requestCount)
		}
	}))
	defer provider.Close()

	root := testBinaryWorkspace(t)
	writeFile(t, filepath.Join(root, "masterdata", aiSettingsFileName), `version: 1
enabled: true
active_profile: test-provider
profiles:
  - id: test-provider
    display_name: Test Provider
    provider_type: openai_compatible
    base_url: `+provider.URL+`
    model: system
    supports_streaming: false
    supports_tool_calls: true
`)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := s.runManagedAIChat(t.Context(), aiRunRequest{
		Message:   "全商品のIDを確認して",
		ProfileID: "test-provider",
		FrontendToolSink: func(_ context.Context, call aiFrontendToolCall) (map[string]any, error) {
			switch call.Name {
			case "get_current_context":
				return map[string]any{
					"success": true,
					"status":  "ok",
					"context": map[string]any{
						"selectedTable":      "product",
						"selectedGeneration": "0000_initial",
					},
				}, nil
			case "stage_table_changes":
				return map[string]any{
					"success":  true,
					"status":   "staged",
					"summary":  "Staged 1 operation(s).",
					"accepted": []any{map[string]any{"index": float64(0)}},
				}, nil
			default:
				t.Fatalf("unexpected frontend tool %s", call.Name)
				return nil, nil
			}
		},
	})
	if err != nil {
		t.Fatalf("runManagedAIChat() error = %v", err)
	}
	session := payload["session"].(aiSession)
	if !strings.Contains(session.Messages[1].Content, "Staged 1 operation") {
		t.Fatalf("assistant content = %q", session.Messages[1].Content)
	}
	if requestCount != 4 {
		t.Fatalf("provider request count = %d", requestCount)
	}
}

func TestRunManagedAIChatStopsRepeatedToolCallLoop(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	requestCount := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{
					"role": "assistant",
					"tool_calls": []map[string]any{{
						"id":   "call-overview",
						"type": "function",
						"function": map[string]any{
							"name":      "get_project_overview",
							"arguments": `{}`,
						},
					}},
				}},
			},
		})
	}))
	defer provider.Close()

	root := testBinaryWorkspace(t)
	writeFile(t, filepath.Join(root, "masterdata", aiSettingsFileName), `version: 1
enabled: true
active_profile: test-provider
profiles:
  - id: test-provider
    display_name: Test Provider
    provider_type: openai_compatible
    base_url: `+provider.URL+`
    model: system
    supports_streaming: false
    supports_tool_calls: true
`)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.runManagedAIChat(t.Context(), aiRunRequest{Message: "概要を確認して", ProfileID: "test-provider"})
	if err == nil {
		t.Fatal("expected repeated tool call error")
	}
	if !strings.Contains(err.Error(), "repeated a recent tool call") {
		t.Fatalf("error = %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("provider request count = %d", requestCount)
	}
}

func TestRunManagedAIChatReportsConfiguredToolRoundLimit(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{
					"role": "assistant",
					"tool_calls": []map[string]any{{
						"id":   "call-overview",
						"type": "function",
						"function": map[string]any{
							"name":      "get_project_overview",
							"arguments": `{}`,
						},
					}},
				}},
			},
		})
	}))
	defer provider.Close()

	root := testBinaryWorkspace(t)
	writeFile(t, filepath.Join(root, "masterdata", aiSettingsFileName), `version: 1
enabled: true
active_profile: test-provider
profiles:
  - id: test-provider
    display_name: Test Provider
    provider_type: openai_compatible
    base_url: `+provider.URL+`
    model: system
    supports_streaming: false
    supports_tool_calls: true
    max_tool_rounds: 1
    tool_loop_window: 1
`)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.runManagedAIChat(t.Context(), aiRunRequest{Message: "概要を確認して", ProfileID: "test-provider"})
	if err == nil {
		t.Fatal("expected configured tool round limit error")
	}
	if !strings.Contains(err.Error(), "AI settings limit this profile to 1 tool-call round") {
		t.Fatalf("error = %v", err)
	}
}

func TestAIRunStreamBrokersFrontendToolResultEndpoint(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	requestCount := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			writeJSON(w, http.StatusOK, map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{
							{
								"id":   "call-context",
								"type": "function",
								"function": map[string]any{
									"name":      "get_current_context",
									"arguments": `{}`,
								},
							},
						},
					}},
				},
			})
			return
		}
		var received openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode provider request: %v", err)
		}
		foundToolResult := false
		for _, message := range received.Messages {
			if message.Role == "tool" && message.ToolCallID == "call-context" && strings.Contains(message.Content, "slime") {
				foundToolResult = true
			}
		}
		if !foundToolResult {
			t.Fatalf("provider did not receive frontend tool result: %#v", received.Messages)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "Context received"}},
			},
		})
	}))
	defer provider.Close()

	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\ndescription: Initial\n")
	writeFile(t, filepath.Join(root, "masterdata", aiSettingsFileName), `version: 1
enabled: true
active_profile: test-provider
profiles:
  - id: test-provider
    display_name: Test Provider
    provider_type: openai_compatible
    base_url: `+provider.URL+`
    model: system
    supports_streaming: false
    supports_tool_calls: true
`)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	app := httptest.NewServer(s.routes())
	defer app.Close()

	reqBody := []byte(`{"profileId":"test-provider","message":"hello"}`)
	resp, err := http.Post(app.URL+"/api/ai/runs/stream", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("post stream: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stream status = %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(resp.Body)
	sawFrontendRequest := false
	sawFinal := false
	for scanner.Scan() {
		var event map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("decode stream event: %v", err)
		}
		switch event["kind"] {
		case "frontend_tool_request":
			sawFrontendRequest = true
			request := event["request"].(map[string]any)
			resultBody, _ := json.Marshal(map[string]any{
				"request_id": request["request_id"],
				"result": map[string]any{
					"success": true,
					"status":  "ok",
					"context": map[string]any{"selectedTable": "enemy", "visibleRows": []map[string]any{{"enemy_id": "slime"}}},
				},
			})
			resultResp, err := http.Post(app.URL+"/api/ai/frontend-tool-results", "application/json", bytes.NewReader(resultBody))
			if err != nil {
				t.Fatalf("post frontend tool result: %v", err)
			}
			_ = resultResp.Body.Close()
			if resultResp.StatusCode != http.StatusOK {
				t.Fatalf("frontend tool result status = %d", resultResp.StatusCode)
			}
		case "final":
			sawFinal = true
			goto done
		}
	}
done:
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan stream: %v", err)
	}
	if !sawFrontendRequest || !sawFinal {
		t.Fatalf("sawFrontendRequest=%v sawFinal=%v", sawFrontendRequest, sawFinal)
	}
	if requestCount != 2 {
		t.Fatalf("provider request count = %d", requestCount)
	}
}

func TestAIFrontendToolResultEndpointIgnoresExpiredRequest(t *testing.T) {
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\ndescription: Initial\n")
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"request_id":"ftool-expired","result":{"success":true}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ai/frontend-tool-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	s.routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["delivered"] != false || payload["status"] != "expired" {
		t.Fatalf("payload = %#v", payload)
	}
}

func decodeNDJSONEvents(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	scanner := bufio.NewScanner(bytes.NewReader(data))
	events := []map[string]any{}
	for scanner.Scan() {
		var event map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("decode stream line: %v", err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan stream: %v", err)
	}
	return events
}

func hasStreamKind(events []map[string]any, kind string) bool {
	for _, event := range events {
		if event["kind"] == kind {
			return true
		}
	}
	return false
}

func TestRunManagedAIChatReturnsStageTableChangesToolCall(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{
					"role": "assistant",
					"tool_calls": []map[string]any{
						{
							"id":   "call-stage",
							"type": "function",
							"function": map[string]any{
								"name":      "stage_table_changes",
								"arguments": `{"tableId":"product","generationId":"0000_initial","operations":[{"op":"update","key":"prod-core","values":{"name":"Core Product"}}]}`,
							},
						},
					},
				}},
			},
		})
	}))
	defer provider.Close()

	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\ndescription: Initial\n")
	writeFile(t, filepath.Join(root, "masterdata", aiSettingsFileName), `version: 1
enabled: true
active_profile: test-provider
profiles:
  - id: test-provider
    display_name: Test Provider
    provider_type: openai_compatible
    base_url: `+provider.URL+`
    model: system
    supports_streaming: false
    supports_tool_calls: true
`)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := s.runManagedAIChat(t.Context(), aiRunRequest{Message: "名前を直して", ProfileID: "test-provider"})
	if err != nil {
		t.Fatalf("runManagedAIChat() error = %v", err)
	}
	stage, ok := payload["stage_table_changes"].(map[string]any)
	if !ok {
		t.Fatalf("stage_table_changes missing: %#v", payload)
	}
	if stage["tableId"] != "product" {
		t.Fatalf("stage tableId = %#v", stage["tableId"])
	}
	ops, ok := stage["operations"].([]any)
	if !ok || len(ops) != 1 {
		t.Fatalf("stage operations = %#v", stage["operations"])
	}
	session := payload["session"].(aiSession)
	if len(session.Messages) != 2 {
		t.Fatalf("message count = %d", len(session.Messages))
	}
	if !strings.Contains(session.Messages[1].Content, "prepared table changes") {
		t.Fatalf("assistant message = %q", session.Messages[1].Content)
	}
	events, ok := payload["debug_events"].([]map[string]any)
	if !ok || len(events) == 0 {
		t.Fatalf("debug_events missing: %#v", payload["debug_events"])
	}
	foundStage := false
	for _, event := range events {
		if event["kind"] == "frontend_staging_requested" {
			foundStage = true
		}
	}
	if !foundStage {
		t.Fatalf("frontend staging event missing: %#v", events)
	}
}

func TestRunManagedAIChatReturnsStageTableChangesAfterReadTool(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	requestCount := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			writeJSON(w, http.StatusOK, map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{
							{
								"id":   "call-overview",
								"type": "function",
								"function": map[string]any{
									"name":      "get_project_overview",
									"arguments": `{}`,
								},
							},
						},
					}},
				},
			})
			return
		}
		var received openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode second provider request: %v", err)
		}
		foundToolResult := false
		for _, message := range received.Messages {
			if message.Role == "tool" && message.ToolCallID == "call-overview" {
				foundToolResult = true
			}
		}
		if !foundToolResult {
			t.Fatalf("second request did not include overview tool result: %#v", received.Messages)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{
					"role": "assistant",
					"tool_calls": []map[string]any{
						{
							"id":   "call-stage",
							"type": "function",
							"function": map[string]any{
								"name":      "stage_table_changes",
								"arguments": `{"tableId":"product","generationId":"0000_initial","operations":[{"op":"delete","key":"prod-core","values":{}}]}`,
							},
						},
					},
				}},
			},
		})
	}))
	defer provider.Close()

	root := testBinaryWorkspace(t)
	writeFile(t, filepath.Join(root, "masterdata", aiSettingsFileName), `version: 1
enabled: true
active_profile: test-provider
profiles:
  - id: test-provider
    display_name: Test Provider
    provider_type: openai_compatible
    base_url: `+provider.URL+`
    model: system
    supports_streaming: false
    supports_tool_calls: true
`)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := s.runManagedAIChat(t.Context(), aiRunRequest{Message: "概要を見てから消して", ProfileID: "test-provider"})
	if err != nil {
		t.Fatalf("runManagedAIChat() error = %v", err)
	}
	stage, ok := payload["stage_table_changes"].(map[string]any)
	if !ok {
		t.Fatalf("stage_table_changes missing: %#v", payload)
	}
	if stage["tableId"] != "product" {
		t.Fatalf("stage tableId = %#v", stage["tableId"])
	}
	if requestCount != 2 {
		t.Fatalf("provider request count = %d", requestCount)
	}
}

func TestExecuteAIToolsReadAndValidateTable(t *testing.T) {
	root := testBinaryWorkspace(t)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	overview, err := s.executeAITool("get_project_overview", map[string]any{})
	if err != nil {
		t.Fatalf("get_project_overview error = %v", err)
	}
	tables, ok := overview["tables"].([]map[string]any)
	if !ok || len(tables) != 1 || tables[0]["table_id"] != "product" {
		t.Fatalf("overview tables = %#v", overview["tables"])
	}
	tablePayload, err := s.executeAITool("get_table", map[string]any{
		"tableId":      "product",
		"generationId": "0000_initial",
		"mode":         "active_only",
		"offset":       0,
		"limit":        1,
		"fields":       []any{"product_id", "name"},
	})
	if err != nil {
		t.Fatalf("get_table error = %v", err)
	}
	records, ok := tablePayload["records"].([]map[string]any)
	if !ok || len(records) != 1 {
		t.Fatalf("records = %#v", tablePayload["records"])
	}
	if _, ok := records[0]["image"]; ok {
		t.Fatalf("field projection leaked image field: %#v", records[0])
	}
	validation, err := s.executeAITool("validate_table", map[string]any{
		"tableId":      "product",
		"generationId": "0000_initial",
		"mode":         "active_only",
	})
	if err != nil {
		t.Fatalf("validate_table error = %v", err)
	}
	if _, ok := validation["diagnostics"].([]map[string]any); !ok {
		t.Fatalf("diagnostics = %#v", validation["diagnostics"])
	}
}

func TestFMToolSchemaNormalization(t *testing.T) {
	body := openAIChatRequest{Tools: managedChatTools()}
	normalized := normalizeChatRequestForProvider(aiProfile{ID: appleFMServeProfileID}, body)
	for _, tool := range normalized.Tools {
		params := tool.Function.Parameters
		if _, ok := params["required"]; !ok {
			t.Fatalf("%s missing required in parameters: %#v", tool.Function.Name, params)
		}
		if params["additionalProperties"] != false {
			t.Fatalf("%s missing additionalProperties=false: %#v", tool.Function.Name, params)
		}
		if tool.Function.Name == "stage_table_changes" {
			properties := params["properties"].(map[string]any)
			operationsJSON := properties["operations_json"].(map[string]any)
			if operationsJSON["type"] != "string" {
				t.Fatalf("operations_json should be string for fm, got %#v", operationsJSON)
			}
			if _, ok := properties["operations"]; ok {
				t.Fatalf("stage_table_changes should not expose operations array schema for fm: %#v", properties)
			}
		}
		if tool.Function.Name == "validate_table" {
			properties := params["properties"].(map[string]any)
			rowsJSON := properties["rows_json"].(map[string]any)
			if rowsJSON["type"] != "string" {
				t.Fatalf("rows_json should be string for fm, got %#v", rowsJSON)
			}
			if _, ok := properties["rows"]; ok {
				t.Fatalf("validate_table should not expose rows array schema for fm: %#v", properties)
			}
		}
	}
}

func TestParseToolArgumentsRestoresFMJSONStringArrays(t *testing.T) {
	var call openAIToolCall
	call.Function.Arguments = `{"tableId":"enemy","generationId":"0000_initial","operations":["{\"op\":\"update\",\"key\":\"bat\",\"values\":{\"hp\":30}}"],"rows":["{\"enemy_id\":\"bat\",\"hp\":30}"]}`
	args := parseToolArguments(call)
	ops, ok := args["operations"].([]any)
	if !ok || len(ops) != 1 {
		t.Fatalf("operations = %#v", args["operations"])
	}
	op, ok := ops[0].(map[string]any)
	if !ok || op["op"] != "update" {
		t.Fatalf("operation was not restored: %#v", ops[0])
	}
	rows, ok := args["rows"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("rows = %#v", args["rows"])
	}
	row, ok := rows[0].(map[string]any)
	if !ok || row["enemy_id"] != "bat" {
		t.Fatalf("row was not restored: %#v", rows[0])
	}

	call.Function.Arguments = `{"tableId":"enemy","generationId":"0000_initial","operations_json":"[{\"op\":\"update\",\"key\":\"slime\",\"values\":{\"defense\":12}}]","rows_json":"[{\"enemy_id\":\"slime\",\"defense\":12}]"}`
	args = parseToolArguments(call)
	ops, ok = args["operations"].([]any)
	if !ok || len(ops) != 1 {
		t.Fatalf("operations_json was not restored: %#v", args["operations"])
	}
	op, ok = ops[0].(map[string]any)
	if !ok || op["key"] != "slime" {
		t.Fatalf("operation_json was not restored as object: %#v", ops[0])
	}
	rows, ok = args["rows"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("rows_json was not restored: %#v", args["rows"])
	}
	row, ok = rows[0].(map[string]any)
	if !ok || row["enemy_id"] != "slime" {
		t.Fatalf("rows_json was not restored as object: %#v", rows[0])
	}
}

func readLogKinds(t *testing.T, file string) []string {
	t.Helper()
	f, err := os.Open(file)
	if err != nil {
		t.Fatalf("open log %s: %v", file, err)
	}
	defer f.Close()
	kinds := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var event map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("decode log line: %v", err)
		}
		if kind, ok := event["kind"].(string); ok {
			kinds = append(kinds, kind)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan log: %v", err)
	}
	return kinds
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
