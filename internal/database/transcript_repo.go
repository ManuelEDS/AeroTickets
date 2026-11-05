package database

import (
	"fmt"
	"time"

	"aerobot/internal/models"
)

// LogTranscript inserts a new transcript log record
func (d *DB) LogTranscript(ticketID int64, filePath string) (*models.TranscriptLog, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC()
	query := `INSERT INTO transcripts_log (ticket_id, file_path, created_at) VALUES (?, ?, ?)`

	res, err := d.conn.Exec(query, ticketID, filePath, now)
	if err != nil {
		return nil, fmt.Errorf("failed to log transcript: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last transcript log id: %w", err)
	}

	return &models.TranscriptLog{
		ID:        id,
		TicketID:  ticketID,
		FilePath:  filePath,
		CreatedAt: now,
	}, nil
}

// GetTranscriptsByTicketID retrieves all transcript logs for a ticket
func (d *DB) GetTranscriptsByTicketID(ticketID int64) ([]models.TranscriptLog, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `SELECT id, ticket_id, file_path, created_at FROM transcripts_log WHERE ticket_id = ? ORDER BY created_at DESC`
	rows, err := d.conn.Query(query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transcripts: %w", err)
	}
	defer rows.Close()

	var logs []models.TranscriptLog
	for rows.Next() {
		var l models.TranscriptLog
		if err := rows.Scan(&l.ID, &l.TicketID, &l.FilePath, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan transcript log: %w", err)
		}
		logs = append(logs, l)
	}

	return logs, nil
}
