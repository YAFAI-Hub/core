package database

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Agent struct {
	ID           int64
	Name         string
	Capabilities string
	Description  string
	Model        string
	Provider     string
	Goal         string
	Status       string
}

type Orchestrator struct {
	Name        string
	Description string
	Scope       string
	Model       string
	Provider    string
	Goal        string
	Team        []Agent
}

type Message struct {
	ID        int64
	ThreadID  int64
	From      string
	To        string
	Content   string
	Timestamp time.Time
	Media     []string
}

type Thread struct {
	ID          int64
	Name        string
	WorkspaceID int64
	Messages    []Message
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Workspace struct {
	ID           int64
	Name         string
	Description  string
	Orchestrator Orchestrator
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type DBWrapper struct {
	DbClient *sql.DB
}

func NewDBWrapper(ctx context.Context, dbPath string) (*DBWrapper, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Ensure database is responsive
	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	err = createTables(ctx, db)
	if err != nil {
		return nil, err
	}
	slog.Info("YAFAI DB Connected.")
	return &DBWrapper{DbClient: db}, nil
}

func createTables(ctx context.Context, db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS agents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			capabilities TEXT,
			description TEXT,
			model TEXT,
			provider TEXT,
			goal TEXT,
			status TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS workspaces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			description TEXT,
			orchestrator_name TEXT,
			orchestrator_description TEXT,
			orchestrator_scope TEXT,
			orchestrator_model TEXT,
			orchestrator_provider TEXT,
			orchestrator_goal TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS threads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			workspace_id INTEGER,
			status TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			thread_id INTEGER,
			sender TEXT,
			recipient TEXT,
			content TEXT,
			timestamp DATETIME,
			media TEXT,
			FOREIGN KEY(thread_id) REFERENCES threads(id)
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_agents (
		workspace_id INTEGER,
		agent_id INTEGER,
		PRIMARY KEY (workspace_id, agent_id),
		FOREIGN KEY(workspace_id) REFERENCES workspaces(id),
		FOREIGN KEY(agent_id) REFERENCES agents(id)
	);`}

	for _, table := range tables {
		_, err := db.ExecContext(ctx, table)
		if err != nil {
			return err
		}
	}

	return nil
}

// Agent CRUD
func (w *DBWrapper) CreateAgent(ctx context.Context, agent *Agent) error {
	result, err := w.DbClient.ExecContext(ctx, `
		INSERT INTO agents 
		(name, capabilities, description, model, provider, goal, status) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, agent.Name, agent.Capabilities, agent.Description,
		agent.Model, agent.Provider, agent.Goal, agent.Status)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	agent.ID = id
	return nil
}

func (w *DBWrapper) GetAgentByID(ctx context.Context, id int64) (*Agent, error) {
	agent := &Agent{}
	err := w.DbClient.QueryRowContext(ctx, `
		SELECT id, name, capabilities, description, 
		model, provider, goal, status 
		FROM agents WHERE id = ?
	`, id).Scan(
		&agent.ID, &agent.Name, &agent.Capabilities,
		&agent.Description, &agent.Model, &agent.Provider,
		&agent.Goal, &agent.Status,
	)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

// Workspace CRUD
func (w *DBWrapper) CreateWorkspace(ctx context.Context, workspace *Workspace) error {
	workspace.CreatedAt = time.Now()
	workspace.UpdatedAt = time.Now()

	result, err := w.DbClient.ExecContext(ctx, `
		INSERT INTO workspaces (
			name, description, 
			orchestrator_name, orchestrator_description, 
			orchestrator_scope, orchestrator_model, 
			orchestrator_provider, orchestrator_goal, 
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		workspace.Name,
		workspace.Description,
		workspace.Orchestrator.Name,
		workspace.Orchestrator.Description,
		workspace.Orchestrator.Scope,
		workspace.Orchestrator.Model,
		workspace.Orchestrator.Provider,
		workspace.Orchestrator.Goal,
		workspace.CreatedAt,
		workspace.UpdatedAt,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	workspace.ID = id
	return nil
}

func (w *DBWrapper) GetWorkspaceByID(ctx context.Context, workspaceID int64) (*Workspace, error) {
	var ws Workspace
	// Fetch workspace row
	err := w.DbClient.QueryRowContext(ctx, `
        SELECT id, name, description, orchestrator_name, orchestrator_description, orchestrator_scope, orchestrator_model, orchestrator_provider, orchestrator_goal, created_at, updated_at
        FROM workspaces
        WHERE id = ?
    `, workspaceID).Scan(
		&ws.ID, &ws.Name, &ws.Description,
		&ws.Orchestrator.Name, &ws.Orchestrator.Description, &ws.Orchestrator.Scope,
		&ws.Orchestrator.Model, &ws.Orchestrator.Provider, &ws.Orchestrator.Goal,
		&ws.CreatedAt, &ws.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Fetch agent IDs from workspace_agents
	rows, err := w.DbClient.QueryContext(ctx, `
        SELECT agent_id FROM workspace_agents WHERE workspace_id = ?
    `, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agentIDs []int64
	for rows.Next() {
		var agentID int64
		if err := rows.Scan(&agentID); err != nil {
			return nil, err
		}
		agentIDs = append(agentIDs, agentID)
	}

	// Fetch each agent and populate Orchestrator.Team
	ws.Orchestrator.Team = []Agent{}
	for _, agentID := range agentIDs {
		var agent Agent
		err := w.DbClient.QueryRowContext(ctx, `
            SELECT id, name, capabilities, description, model, provider, goal, status
            FROM agents WHERE id = ?
        `, agentID).Scan(
			&agent.ID, &agent.Name, &agent.Capabilities, &agent.Description,
			&agent.Model, &agent.Provider, &agent.Goal, &agent.Status,
		)
		if err != nil {
			return nil, err
		}
		ws.Orchestrator.Team = append(ws.Orchestrator.Team, agent)
	}

	return &ws, nil
}

// ListWorkspaces returns all workspaces with basic info (without loading agents)
func (w *DBWrapper) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	rows, err := w.DbClient.QueryContext(ctx, `
        SELECT id, name, description, orchestrator_name, orchestrator_description, orchestrator_scope, orchestrator_model, orchestrator_provider, orchestrator_goal, created_at, updated_at
        FROM workspaces
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []Workspace
	for rows.Next() {
		var ws Workspace
		err := rows.Scan(
			&ws.ID, &ws.Name, &ws.Description,
			&ws.Orchestrator.Name, &ws.Orchestrator.Description, &ws.Orchestrator.Scope,
			&ws.Orchestrator.Model, &ws.Orchestrator.Provider, &ws.Orchestrator.Goal,
			&ws.CreatedAt, &ws.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, ws)
	}
	return workspaces, nil
}

// Thread CRUD
func (w *DBWrapper) CreateThread(ctx context.Context, thread *Thread) error {
	thread.CreatedAt = time.Now()
	thread.UpdatedAt = time.Now()

	result, err := w.DbClient.ExecContext(ctx, `
		INSERT INTO threads (
			name, workspace_id, status, 
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?)
	`,
		thread.Name,
		thread.WorkspaceID,
		thread.Status,
		thread.CreatedAt,
		thread.UpdatedAt,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	thread.ID = id
	return nil
}

func (w *DBWrapper) GetThreadByID(ctx context.Context, id int64) (*Thread, error) {
	thread := &Thread{}

	err := w.DbClient.QueryRowContext(ctx, `
		SELECT 
			id, name, workspace_id, status, 
			created_at, updated_at
		FROM threads WHERE id = ?
	`, id).Scan(
		&thread.ID,
		&thread.Name,
		&thread.WorkspaceID,
		&thread.Status,
		&thread.CreatedAt,
		&thread.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Fetch messages for this thread
	rows, err := w.DbClient.QueryContext(ctx, `
		SELECT id, sender, recipient, content, timestamp, media 
		FROM messages WHERE thread_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var msg Message
		var mediaStr string
		err := rows.Scan(
			&msg.ID,
			&msg.From,
			&msg.To,
			&msg.Content,
			&msg.Timestamp,
			&mediaStr,
		)
		if err != nil {
			return nil, err
		}

		if mediaStr != "" {
			msg.Media = strings.Split(mediaStr, ",")
		}

		thread.Messages = append(thread.Messages, msg)
	}

	return thread, nil
}

// ListThreadsForWorkspace returns all threads for a given workspace ID
func (w *DBWrapper) ListThreadsForWorkspace(ctx context.Context, workspaceID int64) ([]Thread, error) {
    rows, err := w.DbClient.QueryContext(ctx, `
        SELECT id, name, workspace_id, status, created_at, updated_at
        FROM threads
        WHERE workspace_id = ?
    `, workspaceID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var threads []Thread
    for rows.Next() {
        var th Thread
        err := rows.Scan(
            &th.ID,
            &th.Name,
            &th.WorkspaceID,
            &th.Status,
            &th.CreatedAt,
            &th.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }
        threads = append(threads, th)
    }
    return threads, nil
}

// Message Operations
func (w *DBWrapper) AddMessageToThread(ctx context.Context, message *Message) error {
	message.Timestamp = time.Now()

	result, err := w.DbClient.ExecContext(ctx, `
		INSERT INTO messages (
			thread_id, sender, recipient, 
			content, timestamp, media
		) VALUES (?, ?, ?, ?, ?, ?)
	`,
		message.ThreadID,
		message.From,
		message.To,
		message.Content,
		message.Timestamp,
		strings.Join(message.Media, ","),
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	message.ID = id

	return nil
}

func (w *DBWrapper) AddMessagesToThread(ctx context.Context, threadID int64, messages []Message) error {
	tx, err := w.DbClient.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO messages (
			thread_id, sender, recipient, 
			content, timestamp, media
		) VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, message := range messages {
		message.ThreadID = threadID
		message.Timestamp = time.Now()

		_, err = stmt.ExecContext(ctx,
			threadID,
			message.From,
			message.To,
			message.Content,
			message.Timestamp,
			strings.Join(message.Media, ","),
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// ListMessagesByThread returns all messages for a given thread ID, ordered by timestamp ascending.
func (w *DBWrapper) ListMessagesByThread(ctx context.Context, threadID int64) ([]Message, error) {
    rows, err := w.DbClient.QueryContext(ctx, `
        SELECT id, thread_id, sender, recipient, content, timestamp, media
        FROM messages
        WHERE thread_id = ?
        ORDER BY timestamp ASC
    `, threadID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var messages []Message
    for rows.Next() {
        var msg Message
        var mediaStr string
        err := rows.Scan(
            &msg.ID,
            &msg.ThreadID,
            &msg.From,
            &msg.To,
            &msg.Content,
            &msg.Timestamp,
            &mediaStr,
        )
        if err != nil {
            return nil, err
        }
        if mediaStr != "" {
            msg.Media = strings.Split(mediaStr, ",")
        }
        messages = append(messages, msg)
    }
    return messages, nil
}

func (w *DBWrapper) AddAgentToWorkspace(ctx context.Context, workspaceID, agentID int64) error {
	_, err := w.DbClient.ExecContext(ctx, `
        INSERT OR IGNORE INTO workspace_agents (workspace_id, agent_id) VALUES (?, ?)
    `, workspaceID, agentID)
	return err
}

// Utility Methods
func (w *DBWrapper) Close() error {
	return w.DbClient.Close()
}
