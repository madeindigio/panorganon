package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// ServerRecord represents a server in the database
type ServerRecord struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Config   string    `json:"config"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
}

// ToolRecord represents a tool in the database
type ToolRecord struct {
	ID          string    `json:"id"`
	ServerID    string    `json:"server_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	InputSchema string    `json:"input_schema"`
	LastUpdated time.Time `json:"last_updated"`
}

// SaveServer inserts or updates a server record
func (db *DB) SaveServer(server ServerRecord) error {
	query := `
		INSERT INTO servers (id, name, type, config, status, last_seen)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			config = excluded.config,
			status = excluded.status,
			last_seen = excluded.last_seen
	`

	_, err := db.conn.Exec(query,
		server.ID,
		server.Name,
		server.Type,
		server.Config,
		server.Status,
		server.LastSeen,
	)

	if err != nil {
		db.logger.Error("Failed to save server", zap.Error(err), zap.String("server", server.Name))
		return fmt.Errorf("failed to save server: %w", err)
	}

	db.logger.Debug("Saved server", zap.String("server", server.Name))
	return nil
}

// GetServer retrieves a server by name
func (db *DB) GetServer(name string) (*ServerRecord, error) {
	query := `SELECT id, name, type, config, status, last_seen FROM servers WHERE name = ?`

	var server ServerRecord
	err := db.conn.QueryRow(query, name).Scan(
		&server.ID,
		&server.Name,
		&server.Type,
		&server.Config,
		&server.Status,
		&server.LastSeen,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return &server, nil
}

// SaveTool inserts or updates a tool record
func (db *DB) SaveTool(tool ToolRecord) error {
	query := `
		INSERT INTO tools (id, server_id, name, description, input_schema, last_updated)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			description = excluded.description,
			input_schema = excluded.input_schema,
			last_updated = excluded.last_updated
	`

	_, err := db.conn.Exec(query,
		tool.ID,
		tool.ServerID,
		tool.Name,
		tool.Description,
		tool.InputSchema,
		tool.LastUpdated,
	)

	if err != nil {
		db.logger.Error("Failed to save tool", zap.Error(err), zap.String("tool", tool.Name))
		return fmt.Errorf("failed to save tool: %w", err)
	}

	db.logger.Debug("Saved tool", zap.String("tool", tool.Name))
	return nil
}

// GetTool retrieves a tool by name
func (db *DB) GetTool(name string) (*ToolRecord, error) {
	query := `SELECT id, server_id, name, description, input_schema, last_updated FROM tools WHERE name = ? LIMIT 1`

	var tool ToolRecord
	err := db.conn.QueryRow(query, name).Scan(
		&tool.ID,
		&tool.ServerID,
		&tool.Name,
		&tool.Description,
		&tool.InputSchema,
		&tool.LastUpdated,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tool: %w", err)
	}

	return &tool, nil
}

// GetToolsByServer retrieves all tools for a specific server
func (db *DB) GetToolsByServer(serverID string) ([]ToolRecord, error) {
	query := `SELECT id, server_id, name, description, input_schema, last_updated FROM tools WHERE server_id = ?`

	rows, err := db.conn.Query(query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tools: %w", err)
	}
	defer rows.Close()

	var tools []ToolRecord
	for rows.Next() {
		var tool ToolRecord
		if err := rows.Scan(
			&tool.ID,
			&tool.ServerID,
			&tool.Name,
			&tool.Description,
			&tool.InputSchema,
			&tool.LastUpdated,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tool: %w", err)
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

// SearchTools searches for tools by name (substring match)
func (db *DB) SearchTools(query string) ([]ToolRecord, error) {
	sqlQuery := `
		SELECT id, server_id, name, description, input_schema, last_updated
		FROM tools
		WHERE name LIKE ? OR description LIKE ?
		ORDER BY name
		LIMIT 100
	`

	pattern := "%" + query + "%"
	rows, err := db.conn.Query(sqlQuery, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search tools: %w", err)
	}
	defer rows.Close()

	var tools []ToolRecord
	for rows.Next() {
		var tool ToolRecord
		if err := rows.Scan(
			&tool.ID,
			&tool.ServerID,
			&tool.Name,
			&tool.Description,
			&tool.InputSchema,
			&tool.LastUpdated,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tool: %w", err)
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

// GetAllTools retrieves all tools
func (db *DB) GetAllTools() ([]ToolRecord, error) {
	query := `SELECT id, server_id, name, description, input_schema, last_updated FROM tools ORDER BY name`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all tools: %w", err)
	}
	defer rows.Close()

	var tools []ToolRecord
	for rows.Next() {
		var tool ToolRecord
		if err := rows.Scan(
			&tool.ID,
			&tool.ServerID,
			&tool.Name,
			&tool.Description,
			&tool.InputSchema,
			&tool.LastUpdated,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tool: %w", err)
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

// DeleteToolsByServer deletes all tools for a specific server
func (db *DB) DeleteToolsByServer(serverID string) error {
	query := `DELETE FROM tools WHERE server_id = ?`

	result, err := db.conn.Exec(query, serverID)
	if err != nil {
		return fmt.Errorf("failed to delete tools: %w", err)
	}

	affected, _ := result.RowsAffected()
	db.logger.Info("Deleted tools", zap.String("server_id", serverID), zap.Int64("count", affected))

	return nil
}

// SaveToolsFromMCP saves tools from MCP format to database
func (db *DB) SaveToolsFromMCP(serverID string, serverName string, tools []mcp.Tool) error {
	now := time.Now()

	for _, tool := range tools {
		// Marshal input schema to JSON
		schemaJSON, err := json.Marshal(tool.InputSchema)
		if err != nil {
			db.logger.Warn("Failed to marshal tool schema",
				zap.String("tool", tool.Name),
				zap.Error(err),
			)
			schemaJSON = []byte("{}")
		}

		toolRecord := ToolRecord{
			ID:          fmt.Sprintf("%s/%s", serverName, tool.Name),
			ServerID:    serverID,
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: string(schemaJSON),
			LastUpdated: now,
		}

		if err := db.SaveTool(toolRecord); err != nil {
			return fmt.Errorf("failed to save tool %s: %w", tool.Name, err)
		}
	}

	db.logger.Info("Saved tools from MCP",
		zap.String("server", serverName),
		zap.Int("count", len(tools)),
	)

	return nil
}
