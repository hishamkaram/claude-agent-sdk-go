package internal

// Test fixtures for message parser testing.
// These represent realistic JSON messages from the Claude Code CLI.

var (
	// User messages
	userMessageSimple = []byte(`{
		"type": "user",
		"content": "Hello Claude"
	}`)

	userMessageComplex = []byte(`{
		"type": "user",
		"content": [
			{
				"type": "text",
				"text": "Here is the tool result:"
			},
			{
				"type": "tool_result",
				"tool_use_id": "toolu_123",
				"content": "Operation completed successfully"
			}
		],
		"parent_tool_use_id": "parent_456"
	}`)

	//nolint:unused
	userMessageWithToolResult = []byte(`{
		"type": "user",
		"content": [
			{
				"type": "tool_result",
				"tool_use_id": "toolu_789",
				"content": "42",
				"is_error": false
			}
		]
	}`)

	// Assistant messages
	assistantMessageText = []byte(`{
		"type": "assistant",
		"content": [
			{
				"type": "text",
				"text": "Hi there! How can I help you today?"
			}
		],
		"model": "claude-sonnet-4-5-20250929"
	}`)

	assistantMessageToolUse = []byte(`{
		"type": "assistant",
		"content": [
			{
				"type": "text",
				"text": "I'll calculate that for you."
			},
			{
				"type": "tool_use",
				"id": "toolu_calculator_1",
				"name": "calculator",
				"input": {
					"expression": "2 + 2"
				}
			}
		],
		"model": "claude-sonnet-4-5-20250929"
	}`)

	assistantMessageThinking = []byte(`{
		"type": "assistant",
		"content": [
			{
				"type": "thinking",
				"thinking": "Let me analyze this problem step by step...",
				"signature": "sig_abc123"
			},
			{
				"type": "text",
				"text": "Based on my analysis..."
			}
		],
		"model": "claude-sonnet-4-5-20250929",
		"parent_tool_use_id": "parent_tool_xyz"
	}`)

	assistantMessageMixed = []byte(`{
		"type": "assistant",
		"content": [
			{
				"type": "thinking",
				"thinking": "I need to use the bash tool",
				"signature": "sig_def456"
			},
			{
				"type": "text",
				"text": "Running command..."
			},
			{
				"type": "tool_use",
				"id": "toolu_bash_1",
				"name": "bash",
				"input": {
					"command": "ls -la"
				}
			}
		],
		"model": "claude-sonnet-4-5-20250929"
	}`)

	// System messages
	systemMessageMetadata = []byte(`{
		"type": "system",
		"subtype": "metadata",
		"data": {
			"session_id": "sess_abc123",
			"version": "1.0.0"
		}
	}`)

	systemMessageWarning = []byte(`{
		"type": "system",
		"subtype": "warning",
		"data": {
			"message": "API rate limit approaching",
			"current_usage": 80,
			"limit": 100
		}
	}`)

	// Result messages
	resultMessageSuccess = []byte(`{
		"type": "result",
		"subtype": "success",
		"duration_ms": 1234,
		"duration_api_ms": 987,
		"is_error": false,
		"num_turns": 3,
		"session_id": "sess_result_123",
		"total_cost_usd": 0.0045,
		"usage": {
			"input_tokens": 150,
			"output_tokens": 75,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens": 0
		},
		"result": "Task completed successfully"
	}`)

	resultMessageError = []byte(`{
		"type": "result",
		"subtype": "error",
		"duration_ms": 567,
		"duration_api_ms": 234,
		"is_error": true,
		"num_turns": 1,
		"session_id": "sess_error_456",
		"result": "An error occurred during processing"
	}`)

	// Stream events
	streamEventMessageStart = []byte(`{
		"type": "stream_event",
		"uuid": "evt_uuid_123",
		"session_id": "sess_stream_789",
		"event": {
			"type": "message_start",
			"message": {
				"id": "msg_abc",
				"type": "message",
				"role": "assistant",
				"content": [],
				"model": "claude-sonnet-4-5-20250929"
			}
		}
	}`)

	streamEventContentBlockDelta = []byte(`{
		"type": "stream_event",
		"uuid": "evt_uuid_456",
		"session_id": "sess_stream_789",
		"event": {
			"type": "content_block_delta",
			"index": 0,
			"delta": {
				"type": "text_delta",
				"text": "Hello"
			}
		},
		"parent_tool_use_id": "parent_stream_xyz"
	}`)

	streamEventMessageDelta = []byte(`{
		"type": "stream_event",
		"uuid": "evt_uuid_789",
		"session_id": "sess_stream_789",
		"event": {
			"type": "message_delta",
			"delta": {
				"stop_reason": "end_turn",
				"stop_sequence": null
			},
			"usage": {
				"output_tokens": 42
			}
		}
	}`)

	// Individual content blocks
	textBlockJSON = []byte(`{
		"type": "text",
		"text": "This is a text block"
	}`)

	thinkingBlockJSON = []byte(`{
		"type": "thinking",
		"thinking": "Let me think about this...",
		"signature": "sig_thinking_123"
	}`)

	toolUseBlockJSON = []byte(`{
		"type": "tool_use",
		"id": "toolu_block_123",
		"name": "calculator",
		"input": {
			"operation": "add",
			"a": 10,
			"b": 20
		}
	}`)

	toolResultBlockJSON = []byte(`{
		"type": "tool_result",
		"tool_use_id": "toolu_result_456",
		"content": "The result is 30"
	}`)

	toolResultBlockJSONWithError = []byte(`{
		"type": "tool_result",
		"tool_use_id": "toolu_error_789",
		"content": "Command failed with exit code 1",
		"is_error": true
	}`)

	toolResultBlockJSONComplex = []byte(`{
		"type": "tool_result",
		"tool_use_id": "toolu_complex_999",
		"content": [
			{
				"type": "text",
				"text": "Multi-part result"
			}
		],
		"is_error": false
	}`)

	// Invalid/malformed messages for error testing
	invalidJSONMalformed = []byte(`{
		"type": "user",
		"content": "Missing closing brace"`)

	invalidJSONMissingType = []byte(`{
		"content": "No type field"
	}`)

	invalidJSONUnknownType = []byte(`{
		"type": "unknown_message_type",
		"data": {}
	}`)

	invalidJSONEmptyBytes = []byte(``)

	invalidJSONNullType = []byte(`{
		"type": null,
		"content": "Type is null"
	}`)

	invalidJSONNumberType = []byte(`{
		"type": 123,
		"content": "Type is a number"
	}`)

	// Messages with extra fields (forward compatibility)
	userMessageExtraFields = []byte(`{
		"type": "user",
		"content": "Hello",
		"future_field": "should be ignored",
		"another_unknown_field": {
			"nested": "data"
		}
	}`)

	assistantMessageExtraFields = []byte(`{
		"type": "assistant",
		"content": [
			{
				"type": "text",
				"text": "Response",
				"extra_block_field": "ignored"
			}
		],
		"model": "claude-sonnet-4-5-20250929",
		"new_api_feature": true
	}`)

	// Content blocks with errors
	contentBlockMissingType = []byte(`{
		"text": "No type field"
	}`)

	contentBlockUnknownType = []byte(`{
		"type": "unknown_block_type",
		"data": "something"
	}`)

	contentBlockMalformed = []byte(`{
		"type": "text",
		"text": "Unclosed`)

	// Multiple content blocks for batch parsing
	multipleContentBlocks = []string{
		`{"type": "text", "text": "First block"}`,
		`{"type": "thinking", "thinking": "Second block", "signature": "sig_2"}`,
		`{"type": "tool_use", "id": "tool_3", "name": "test", "input": {}}`,
	}

	//nolint:unused
	emptyContentBlocks = []string{}

	// User message variants for comprehensive testing
	userMessageOnlyText = []byte(`{
		"type": "user",
		"content": "Simple text content"
	}`)

	userMessageContentBlocks = []byte(`{
		"type": "user",
		"content": [
			{
				"type": "text",
				"text": "Part 1"
			},
			{
				"type": "text",
				"text": "Part 2"
			}
		]
	}`)

	// Assistant message with all block types
	assistantMessageAllBlocks = []byte(`{
		"type": "assistant",
		"content": [
			{
				"type": "thinking",
				"thinking": "Analyzing request",
				"signature": "sig_all"
			},
			{
				"type": "text",
				"text": "I'll help with that"
			},
			{
				"type": "tool_use",
				"id": "toolu_all_1",
				"name": "bash",
				"input": {"command": "echo test"}
			}
		],
		"model": "claude-sonnet-4-5-20250929"
	}`)

	// -----------------------------------------------------------------------
	// New message type fixtures (016-sdk-message-types)
	// -----------------------------------------------------------------------

	// Top-level message types
	toolProgressMessage = []byte(`{
		"type": "tool_progress",
		"tool_use_id": "toolu_tp_001",
		"tool_name": "Bash",
		"parent_tool_use_id": "parent_tp_001",
		"elapsed_time_seconds": 5.3,
		"task_id": "task_tp_001",
		"uuid": "uuid_tp_001",
		"session_id": "sess_tp_001"
	}`)

	toolProgressMessageMinimal = []byte(`{
		"type": "tool_progress",
		"tool_use_id": "toolu_tp_002",
		"tool_name": "Read",
		"elapsed_time_seconds": 1.2,
		"uuid": "uuid_tp_002",
		"session_id": "sess_tp_002"
	}`)

	authStatusMessage = []byte(`{
		"type": "auth_status",
		"isAuthenticating": true,
		"output": ["Checking credentials...", "Contacting auth server..."],
		"error": "token expired",
		"uuid": "uuid_auth_001",
		"session_id": "sess_auth_001"
	}`)

	authStatusMessageNoError = []byte(`{
		"type": "auth_status",
		"isAuthenticating": false,
		"output": ["Authenticated successfully"],
		"uuid": "uuid_auth_002",
		"session_id": "sess_auth_002"
	}`)

	toolUseSummaryMessage = []byte(`{
		"type": "tool_use_summary",
		"summary": "Used Bash to run tests, Read to check output",
		"preceding_tool_use_ids": ["toolu_sum_001", "toolu_sum_002", "toolu_sum_003"],
		"uuid": "uuid_sum_001",
		"session_id": "sess_sum_001"
	}`)

	rateLimitEventAllowed = []byte(`{
		"type": "rate_limit_event",
		"rate_limit_info": {
			"status": "allowed"
		},
		"uuid": "uuid_rl_001",
		"session_id": "sess_rl_001"
	}`)

	rateLimitEventWarning = []byte(`{
		"type": "rate_limit_event",
		"rate_limit_info": {
			"status": "allowed_warning",
			"utilization": 0.85
		},
		"uuid": "uuid_rl_002",
		"session_id": "sess_rl_002"
	}`)

	rateLimitEventRejected = []byte(`{
		"type": "rate_limit_event",
		"rate_limit_info": {
			"status": "rejected",
			"resetsAt": 1711036800.0,
			"utilization": 1.0
		},
		"uuid": "uuid_rl_003",
		"session_id": "sess_rl_003"
	}`)

	promptSuggestionMessage = []byte(`{
		"type": "prompt_suggestion",
		"suggestion": "Can you also add tests for the new feature?",
		"uuid": "uuid_ps_001",
		"session_id": "sess_ps_001"
	}`)

	// System subtype messages
	systemCompactBoundary = []byte(`{
		"type": "system",
		"subtype": "compact_boundary",
		"compact_metadata": {
			"trigger": "auto",
			"pre_tokens": 50000
		},
		"uuid": "uuid_cb_001",
		"session_id": "sess_cb_001"
	}`)

	systemStatus = []byte(`{
		"type": "system",
		"subtype": "status",
		"status": "compacting",
		"permissionMode": "default",
		"uuid": "uuid_st_001",
		"session_id": "sess_st_001"
	}`)

	systemStatusNilStatus = []byte(`{
		"type": "system",
		"subtype": "status",
		"uuid": "uuid_st_002",
		"session_id": "sess_st_002"
	}`)

	systemHookStarted = []byte(`{
		"type": "system",
		"subtype": "hook_started",
		"hook_id": "hook_001",
		"hook_name": "pre-commit",
		"hook_event": "PostToolUse",
		"uuid": "uuid_hs_001",
		"session_id": "sess_hs_001"
	}`)

	systemHookProgress = []byte(`{
		"type": "system",
		"subtype": "hook_progress",
		"hook_id": "hook_002",
		"hook_name": "lint-check",
		"hook_event": "PostToolUse",
		"stdout": "Running eslint...\n",
		"stderr": "",
		"output": "Running eslint...\n",
		"uuid": "uuid_hp_001",
		"session_id": "sess_hp_001"
	}`)

	systemHookResponse = []byte(`{
		"type": "system",
		"subtype": "hook_response",
		"hook_id": "hook_003",
		"hook_name": "pre-commit",
		"hook_event": "PostToolUse",
		"output": "Hook completed",
		"stdout": "All checks passed",
		"stderr": "",
		"exit_code": 0,
		"outcome": "success",
		"uuid": "uuid_hr_001",
		"session_id": "sess_hr_001"
	}`)

	systemTaskNotification = []byte(`{
		"type": "system",
		"subtype": "task_notification",
		"task_id": "task_001",
		"tool_use_id": "toolu_tn_001",
		"status": "completed",
		"output_file": "/tmp/task_001_output.txt",
		"summary": "Task completed: implemented feature X",
		"usage": {
			"total_tokens": 5000,
			"tool_uses": 12,
			"duration_ms": 30000
		},
		"uuid": "uuid_tn_001",
		"session_id": "sess_tn_001"
	}`)

	systemTaskStarted = []byte(`{
		"type": "system",
		"subtype": "task_started",
		"task_id": "task_002",
		"tool_use_id": "toolu_ts_001",
		"description": "Implementing user authentication",
		"task_type": "agent",
		"uuid": "uuid_ts_001",
		"session_id": "sess_ts_001"
	}`)

	systemTaskProgress = []byte(`{
		"type": "system",
		"subtype": "task_progress",
		"task_id": "task_003",
		"description": "Running tests",
		"usage": {
			"total_tokens": 2500,
			"tool_uses": 5,
			"duration_ms": 15000
		},
		"last_tool_name": "Bash",
		"uuid": "uuid_tprog_001",
		"session_id": "sess_tprog_001"
	}`)

	systemFilesPersisted = []byte(`{
		"type": "system",
		"subtype": "files_persisted",
		"files": [
			{"filename": "src/main.go", "file_id": "file_001"},
			{"filename": "src/main_test.go", "file_id": "file_002"}
		],
		"failed": [
			{"filename": "src/binary.dat", "error": "file too large"}
		],
		"processed_at": "2026-03-19T10:30:00Z",
		"uuid": "uuid_fp_001",
		"session_id": "sess_fp_001"
	}`)

	// Enhanced ResultMessage fixtures
	resultMessageMaxTurns = []byte(`{
		"type": "result",
		"subtype": "error_max_turns",
		"duration_ms": 120000,
		"duration_api_ms": 115000,
		"is_error": true,
		"num_turns": 25,
		"session_id": "sess_mt_001",
		"errors": ["Maximum number of turns (25) reached", "Task incomplete"],
		"uuid": "uuid_mt_001"
	}`)

	resultMessageMaxBudget = []byte(`{
		"type": "result",
		"subtype": "error_max_budget_usd",
		"duration_ms": 60000,
		"duration_api_ms": 55000,
		"is_error": true,
		"num_turns": 10,
		"session_id": "sess_mb_001",
		"errors": ["Budget of $1.00 exceeded"],
		"total_cost_usd": 1.05,
		"uuid": "uuid_mb_001"
	}`)

	resultMessageWithPermissionDenials = []byte(`{
		"type": "result",
		"subtype": "success",
		"duration_ms": 5000,
		"duration_api_ms": 4500,
		"is_error": false,
		"num_turns": 3,
		"session_id": "sess_pd_001",
		"permission_denials": [
			{
				"tool_name": "Bash",
				"tool_use_id": "toolu_pd_001",
				"tool_input": {"command": "rm -rf /"}
			}
		],
		"uuid": "uuid_pd_001"
	}`)

	resultMessageSuccessWithModelUsage = []byte(`{
		"type": "result",
		"subtype": "success",
		"duration_ms": 8000,
		"duration_api_ms": 7500,
		"is_error": false,
		"num_turns": 5,
		"session_id": "sess_mu_001",
		"total_cost_usd": 0.15,
		"stop_reason": "end_turn",
		"modelUsage": {
			"claude-sonnet-4-5-20250929": {
				"input_tokens": 1000,
				"output_tokens": 500,
				"cache_creation_input_tokens": 200,
				"cache_read_input_tokens": 100
			}
		},
		"uuid": "uuid_mu_001",
		"result": "All tasks completed"
	}`)

	// User message with replay flag
	userMessageReplay = []byte(`{
		"type": "user",
		"content": "Previously sent message",
		"isReplay": true
	}`)

	// Unknown message type (forward compatibility)
	unknownTypeMessage = []byte(`{
		"type": "future_feature_xyz",
		"data": {"key": "value"},
		"uuid": "uuid_unk_001"
	}`)
)
