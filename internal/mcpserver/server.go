package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/executionhistory"
	"github.com/sergii/specview/internal/federationruntime"
)

const ProtocolVersion = "2025-11-25"

const (
	parseErrorCode     = -32700
	invalidRequestCode = -32600
	methodNotFoundCode = -32601
	invalidParamsCode  = -32602
)

type Reader interface {
	ListRepositories(context.Context) (controlplane.ListRepositoriesResult, error)
	GetRepository(context.Context, string) (controlplane.GetRepositoryResult, error)
	ListActiveSessions(context.Context) (controlplane.ListActiveSessionsResult, error)
	ListWorktrees(context.Context, string) (controlplane.ListWorktreesResult, error)
	ListWorkItems(context.Context, string) (controlplane.ListWorkItemsResult, error)
	GetWorkItem(context.Context, string, string) (controlplane.GetWorkItemResult, error)
	GetEvidence(context.Context, string, string) (controlplane.GetEvidenceResult, error)
	GetAcceptance(context.Context, string, string) (controlplane.GetAcceptanceResult, error)
}

type HistoryReader interface {
	GetExecutionHistory(context.Context) (executionhistory.Projection, error)
}

type HostControlPlaneReader interface {
	GetHostControlPlane(context.Context) (controlplane.GetHostControlPlaneResult, error)
}

type RepositoryControlPlaneReader interface {
	GetRepositoryControlPlane(context.Context, string) (controlplane.GetRepositoryControlPlaneResult, error)
}

type FederationReader interface {
	Build(context.Context) (federationruntime.Projection, error)
}

type Server struct {
	reader     Reader
	federation FederationReader
	version    string
}

func New(reader Reader, version string) *Server {
	return NewWithFederation(reader, nil, version)
}

func NewWithFederation(reader Reader, federation FederationReader, version string) *Server {
	return &Server{reader: reader, federation: federation, version: version}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type initializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities,omitempty"`
	ClientInfo      map[string]any `json:"clientInfo,omitempty"`
	Meta            map[string]any `json:"_meta,omitempty"`
}

type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Meta      map[string]any  `json:"_meta,omitempty"`
}

type repositoryIDArgs struct {
	RepositoryID string `json:"repository_id"`
}

type hostIDArgs struct {
	HostID string `json:"host_id"`
}

type federationRepositoryArgs struct {
	HostID     string `json:"host_id"`
	InstanceID string `json:"instance_id"`
}

type workItemArgs struct {
	RepositoryID string `json:"repository_id"`
	WorkItemID   string `json:"work_item_id"`
}

type toolResult struct {
	Content           []content `json:"content"`
	StructuredContent any       `json:"structuredContent,omitempty"`
	IsError           bool      `json:"isError,omitempty"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	if s.reader == nil {
		return errors.New("MCP reader is required")
	}

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := s.handleLine(ctx, encoder, []byte(line)); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (s *Server) handleLine(ctx context.Context, encoder *json.Encoder, line []byte) error {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return encoder.Encode(errorResponse(nil, parseErrorCode, "Parse error", nil))
	}
	if req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
		if isNotification(req.ID) {
			return nil
		}
		return encoder.Encode(errorResponse(req.ID, invalidRequestCode, "Invalid Request", nil))
	}

	if isNotification(req.ID) {
		s.handleNotification(req)
		return nil
	}

	result, rpcErr := s.dispatch(ctx, req)
	if rpcErr != nil {
		return encoder.Encode(response{JSONRPC: "2.0", ID: req.ID, Error: rpcErr})
	}
	return encoder.Encode(response{JSONRPC: "2.0", ID: req.ID, Result: result})
}

func (s *Server) handleNotification(req request) {
	// Legacy MCP clients send notifications/initialized after initialize.
	// Unknown notifications are intentionally ignored because JSON-RPC does not
	// permit a response to notifications.
	_ = req
}

func (s *Server) dispatch(ctx context.Context, req request) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return s.initialize(req.Params)
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": toolDefinitionsForServer(s.reader, s.federation != nil)}, nil
	case "tools/call":
		return s.callTool(ctx, req.Params)
	default:
		return nil, &rpcError{Code: methodNotFoundCode, Message: "Method not found"}
	}
}

func (s *Server) initialize(raw json.RawMessage) (any, *rpcError) {
	if len(raw) > 0 {
		var params initializeParams
		if err := decodeStrict(raw, &params); err != nil {
			return nil, invalidParams("initialize", err)
		}
	}
	return map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{"listChanged": false},
		},
		"serverInfo": map[string]any{
			"name":    "specview",
			"version": s.version,
		},
		"instructions": "Specview exposes read-only deterministic facts about Host and repository control-plane summaries, repositories, work items, evidence, acceptance, worktrees, active and historical coding-agent sessions, and the current multi-host federation projection.",
	}, nil
}

func (s *Server) callTool(ctx context.Context, raw json.RawMessage) (any, *rpcError) {
	var params callToolParams
	if err := decodeStrict(raw, &params); err != nil {
		return nil, invalidParams("tools/call", err)
	}
	if strings.TrimSpace(params.Name) == "" {
		return nil, invalidParams("tools/call", errors.New("name is required"))
	}

	switch params.Name {
	case "list_repositories":
		if err := requireEmptyArguments(params.Arguments); err != nil {
			return nil, invalidParams(params.Name, err)
		}
		value, err := s.reader.ListRepositories(ctx)
		return toolResultFor(value, err), nil
	case "get_repository":
		arguments, err := decodeRepositoryID(params.Arguments)
		if err != nil {
			return nil, invalidParams(params.Name, err)
		}
		value, readErr := s.reader.GetRepository(ctx, arguments.RepositoryID)
		return toolResultFor(value, readErr), nil
	case "get_host_control_plane":
		if err := requireEmptyArguments(params.Arguments); err != nil {
			return nil, invalidParams(params.Name, err)
		}
		controlPlaneReader, ok := s.reader.(HostControlPlaneReader)
		if !ok {
			return toolResultFor(nil, errors.New("Host control-plane reader is not configured")), nil
		}
		value, readErr := controlPlaneReader.GetHostControlPlane(ctx)
		return toolResultFor(value, readErr), nil
	case "get_repository_control_plane":
		arguments, err := decodeRepositoryID(params.Arguments)
		if err != nil {
			return nil, invalidParams(params.Name, err)
		}
		controlPlaneReader, ok := s.reader.(RepositoryControlPlaneReader)
		if !ok {
			return toolResultFor(nil, errors.New("repository control-plane reader is not configured")), nil
		}
		value, readErr := controlPlaneReader.GetRepositoryControlPlane(ctx, arguments.RepositoryID)
		return toolResultFor(value, readErr), nil
	case "list_active_sessions":
		if err := requireEmptyArguments(params.Arguments); err != nil {
			return nil, invalidParams(params.Name, err)
		}
		value, err := s.reader.ListActiveSessions(ctx)
		return toolResultFor(value, err), nil
	case "get_execution_history":
		if err := requireEmptyArguments(params.Arguments); err != nil {
			return nil, invalidParams(params.Name, err)
		}
		historyReader, ok := s.reader.(HistoryReader)
		if !ok {
			return toolResultFor(nil, errors.New("execution history reader is not configured")), nil
		}
		value, err := historyReader.GetExecutionHistory(ctx)
		return toolResultFor(value, err), nil
	case "list_worktrees":
		arguments, err := decodeRepositoryID(params.Arguments)
		if err != nil {
			return nil, invalidParams(params.Name, err)
		}
		value, readErr := s.reader.ListWorktrees(ctx, arguments.RepositoryID)
		return toolResultFor(value, readErr), nil
	case "list_work_items":
		arguments, err := decodeRepositoryID(params.Arguments)
		if err != nil {
			return nil, invalidParams(params.Name, err)
		}
		value, readErr := s.reader.ListWorkItems(ctx, arguments.RepositoryID)
		return toolResultFor(value, readErr), nil
	case "get_work_item":
		arguments, err := decodeWorkItemArgs(params.Arguments)
		if err != nil {
			return nil, invalidParams(params.Name, err)
		}
		value, readErr := s.reader.GetWorkItem(ctx, arguments.RepositoryID, arguments.WorkItemID)
		return toolResultFor(value, readErr), nil
	case "get_evidence":
		arguments, err := decodeWorkItemArgs(params.Arguments)
		if err != nil {
			return nil, invalidParams(params.Name, err)
		}
		value, readErr := s.reader.GetEvidence(ctx, arguments.RepositoryID, arguments.WorkItemID)
		return toolResultFor(value, readErr), nil
	case "get_acceptance":
		arguments, err := decodeWorkItemArgs(params.Arguments)
		if err != nil {
			return nil, invalidParams(params.Name, err)
		}
		value, readErr := s.reader.GetAcceptance(ctx, arguments.RepositoryID, arguments.WorkItemID)
		return toolResultFor(value, readErr), nil
	case "get_federation_status":
		if err := requireEmptyArguments(params.Arguments); err != nil {
			return nil, invalidParams(params.Name, err)
		}
		if s.federation == nil {
			return toolResultFor(nil, errors.New("federation reader is not configured")), nil
		}
		value, err := s.federation.Build(ctx)
		return toolResultFor(value, err), nil
	case "get_federation_host":
		arguments, err := decodeHostID(params.Arguments)
		if err != nil {
			return nil, invalidParams(params.Name, err)
		}
		if s.federation == nil {
			return toolResultFor(nil, errors.New("federation reader is not configured")), nil
		}
		projection, readErr := s.federation.Build(ctx)
		if readErr != nil {
			return toolResultFor(nil, readErr), nil
		}
		value, projectErr := projectFederationHost(projection, arguments.HostID)
		return toolResultFor(value, projectErr), nil
	case "get_federation_repository":
		arguments, err := decodeFederationRepositoryArgs(params.Arguments)
		if err != nil {
			return nil, invalidParams(params.Name, err)
		}
		if s.federation == nil {
			return toolResultFor(nil, errors.New("federation reader is not configured")), nil
		}
		projection, readErr := s.federation.Build(ctx)
		if readErr != nil {
			return toolResultFor(nil, readErr), nil
		}
		value, projectErr := projectFederationRepository(projection, arguments.HostID, arguments.InstanceID)
		return toolResultFor(value, projectErr), nil
	default:
		return toolResultFor(nil, fmt.Errorf("unknown Specview tool %q", params.Name)), nil
	}
}

func toolDefinitionsForServer(reader Reader, hasFederation bool) []map[string]any {
	definitions := toolDefinitions()
	_, hasHistory := reader.(HistoryReader)
	_, hasHostControlPlane := reader.(HostControlPlaneReader)
	_, hasRepositoryControlPlane := reader.(RepositoryControlPlaneReader)
	filtered := make([]map[string]any, 0, len(definitions))
	for _, definition := range definitions {
		name, _ := definition["name"].(string)
		if name == "get_execution_history" && !hasHistory {
			continue
		}
		if name == "get_host_control_plane" && !hasHostControlPlane {
			continue
		}
		if name == "get_repository_control_plane" && !hasRepositoryControlPlane {
			continue
		}
		if (name == "get_federation_host" || name == "get_federation_repository") && !hasFederation {
			continue
		}
		filtered = append(filtered, definition)
	}
	return filtered
}

func toolDefinitions() []map[string]any {
	readOnly := map[string]any{
		"readOnlyHint":    true,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
	emptySchema := map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
	repositoryProperty := map[string]any{
		"type":        "string",
		"description": "Opaque host-local repository ID returned by list_repositories.",
	}
	repositorySchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"repository_id": repositoryProperty,
		},
		"required":             []string{"repository_id"},
		"additionalProperties": false,
	}
	hostSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"host_id": map[string]any{
				"type":        "string",
				"description": "Exact Host ID returned by get_federation_status.",
			},
		},
		"required":             []string{"host_id"},
		"additionalProperties": false,
	}
	federationRepositorySchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"host_id": map[string]any{
				"type":        "string",
				"description": "Exact Host ID returned by get_federation_status or get_federation_host.",
			},
			"instance_id": map[string]any{
				"type":        "string",
				"description": "Exact RepositoryInstance ID returned by the federation projection.",
			},
		},
		"required":             []string{"host_id", "instance_id"},
		"additionalProperties": false,
	}
	workItemSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"repository_id": repositoryProperty,
			"work_item_id": map[string]any{
				"type":        "string",
				"description": "Normalized WorkItem ID returned by list_work_items.",
			},
		},
		"required":             []string{"repository_id", "work_item_id"},
		"additionalProperties": false,
	}
	return []map[string]any{
		{
			"name":        "list_repositories",
			"description": "List repositories known on this Specview host, combining persisted history with live agent execution state.",
			"inputSchema": emptySchema,
			"annotations": readOnly,
		},
		{
			"name":        "get_repository",
			"description": "Get one repository with live agent state plus degradable Git and forge context.",
			"inputSchema": repositorySchema,
			"annotations": readOnly,
		},
		{
			"name":        "get_host_control_plane",
			"description": "Get this Host's read-only control-plane summary across Intent, logical Execution, native Evidence, Acceptance, and factual attention signals without inventing aggregate health.",
			"inputSchema": emptySchema,
			"annotations": readOnly,
		},
		{
			"name":        "get_repository_control_plane",
			"description": "Get one repository read-only control-plane summary across Intent, logical Execution, native Evidence, and Acceptance without inventing aggregate health.",
			"inputSchema": repositorySchema,
			"annotations": readOnly,
		},
		{
			"name":        "list_active_sessions",
			"description": "List active coding-agent execution sessions observed on this host.",
			"inputSchema": emptySchema,
			"annotations": readOnly,
		},
		{
			"name":        "get_execution_history",
			"description": "Get deterministic local Host execution history, including active and ended logical sessions with repository attribution.",
			"inputSchema": emptySchema,
			"annotations": readOnly,
		},
		{
			"name":        "list_worktrees",
			"description": "List Git worktrees and branch/revision/dirty state for one repository.",
			"inputSchema": repositorySchema,
			"annotations": readOnly,
		},
		{
			"name":        "list_work_items",
			"description": "List normalized WorkItems for one repository so agent clients can discover stable work_item_id values before requesting details.",
			"inputSchema": repositorySchema,
			"annotations": readOnly,
		},
		{
			"name":        "get_work_item",
			"description": "Get one normalized WorkItem, including its Intent content and relationships.",
			"inputSchema": workItemSchema,
			"annotations": readOnly,
		},
		{
			"name":        "get_evidence",
			"description": "Get normalized revision-scoped Evidence records for one WorkItem, newest observations first.",
			"inputSchema": workItemSchema,
			"annotations": readOnly,
		},
		{
			"name":        "get_acceptance",
			"description": "Evaluate and return deterministic Acceptance for one WorkItem against its exact current revision.",
			"inputSchema": workItemSchema,
			"annotations": readOnly,
		},
		{
			"name":        "get_federation_status",
			"description": "Get the deterministic local plus configured remote Host federation projection, including peer freshness and correlated repository groups.",
			"inputSchema": emptySchema,
			"annotations": readOnly,
		},
		{
			"name":        "get_federation_host",
			"description": "Get one exact Host from the current federation projection with source/freshness, Host control-plane facts, and only that Host's correlated repository instances.",
			"inputSchema": hostSchema,
			"annotations": readOnly,
		},
		{
			"name":        "get_federation_repository",
			"description": "Get one exact Host-scoped RepositoryInstance from the current federation projection with source Host, correlation-group metadata, sessions, worktrees, fingerprint, and source repository attribution.",
			"inputSchema": federationRepositorySchema,
			"annotations": readOnly,
		},
	}
}

func decodeRepositoryID(raw json.RawMessage) (repositoryIDArgs, error) {
	var arguments repositoryIDArgs
	if len(raw) == 0 || string(raw) == "null" {
		return arguments, errors.New("arguments.repository_id is required")
	}
	if err := decodeStrict(raw, &arguments); err != nil {
		return arguments, err
	}
	arguments.RepositoryID = strings.TrimSpace(arguments.RepositoryID)
	if arguments.RepositoryID == "" {
		return arguments, errors.New("arguments.repository_id is required")
	}
	return arguments, nil
}

func decodeHostID(raw json.RawMessage) (hostIDArgs, error) {
	var arguments hostIDArgs
	if len(raw) == 0 || string(raw) == "null" {
		return arguments, errors.New("arguments.host_id is required")
	}
	if err := decodeStrict(raw, &arguments); err != nil {
		return arguments, err
	}
	arguments.HostID = strings.TrimSpace(arguments.HostID)
	if arguments.HostID == "" {
		return arguments, errors.New("arguments.host_id is required")
	}
	return arguments, nil
}

func decodeFederationRepositoryArgs(raw json.RawMessage) (federationRepositoryArgs, error) {
	var arguments federationRepositoryArgs
	if len(raw) == 0 || string(raw) == "null" {
		return arguments, errors.New("arguments.host_id and arguments.instance_id are required")
	}
	if err := decodeStrict(raw, &arguments); err != nil {
		return arguments, err
	}
	arguments.HostID = strings.TrimSpace(arguments.HostID)
	arguments.InstanceID = strings.TrimSpace(arguments.InstanceID)
	if arguments.HostID == "" {
		return arguments, errors.New("arguments.host_id is required")
	}
	if arguments.InstanceID == "" {
		return arguments, errors.New("arguments.instance_id is required")
	}
	return arguments, nil
}

func decodeWorkItemArgs(raw json.RawMessage) (workItemArgs, error) {
	var arguments workItemArgs
	if len(raw) == 0 || string(raw) == "null" {
		return arguments, errors.New("arguments.repository_id and arguments.work_item_id are required")
	}
	if err := decodeStrict(raw, &arguments); err != nil {
		return arguments, err
	}
	arguments.RepositoryID = strings.TrimSpace(arguments.RepositoryID)
	arguments.WorkItemID = strings.TrimSpace(arguments.WorkItemID)
	if arguments.RepositoryID == "" {
		return arguments, errors.New("arguments.repository_id is required")
	}
	if arguments.WorkItemID == "" {
		return arguments, errors.New("arguments.work_item_id is required")
	}
	return arguments, nil
}

func requireEmptyArguments(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var arguments map[string]json.RawMessage
	if err := decodeStrict(raw, &arguments); err != nil {
		return err
	}
	if len(arguments) != 0 {
		return errors.New("tool does not accept arguments")
	}
	return nil
}

func decodeStrict(raw json.RawMessage, destination any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func toolResultFor(value any, err error) toolResult {
	if err != nil {
		return toolResult{
			Content: []content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}
	}
	encoded, marshalErr := json.MarshalIndent(value, "", "  ")
	if marshalErr != nil {
		return toolResult{
			Content: []content{{Type: "text", Text: marshalErr.Error()}},
			IsError: true,
		}
	}
	return toolResult{
		Content:           []content{{Type: "text", Text: string(encoded)}},
		StructuredContent: value,
	}
}

func invalidParams(method string, err error) *rpcError {
	return &rpcError{
		Code:    invalidParamsCode,
		Message: "Invalid params",
		Data: map[string]any{
			"method": method,
			"error":  err.Error(),
		},
	}
}

func errorResponse(id json.RawMessage, code int, message string, data any) response {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message, Data: data},
	}
}

func isNotification(id json.RawMessage) bool {
	return len(id) == 0
}
