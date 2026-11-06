package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"aerobot/internal/models"
)

var (
	ErrTicketNotFound = errors.New("ticket not found")
)

// CreateTicket inserts a new open ticket into the database
func (d *DB) CreateTicket(channelID, userID string) (*models.Ticket, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC()
	query := `INSERT INTO tickets (channel_id, user_id, status, category, ingame_nickname, category_details, additional_info, problem_description, created_at)
			  VALUES (?, ?, ?, '', '', '', '', '', ?)`

	res, err := d.conn.Exec(query, channelID, userID, string(models.StatusOpen), now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert ticket: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &models.Ticket{
		ID:                 id,
		ChannelID:          channelID,
		UserID:             userID,
		Status:             models.StatusOpen,
		Category:           "",
		IngameNickname:     "",
		CategoryDetails:    "",
		AdditionalInfo:     "",
		ProblemDescription: "",
		CreatedAt:          now,
	}, nil
}

// CreateTicketWithInfo inserts a fully populated ticket as RESOLVED.
func (d *DB) CreateTicketWithInfo(channelID, userID, category, nickname, categoryDetails, additionalInfo, problemDesc string) (*models.Ticket, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC()
	query := `INSERT INTO tickets (channel_id, user_id, status, category, ingame_nickname, category_details, additional_info, problem_description, created_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := d.conn.Exec(
		query,
		channelID,
		userID,
		string(models.StatusResolved),
		category,
		nickname,
		categoryDetails,
		additionalInfo,
		problemDesc,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert ticket with info: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &models.Ticket{
		ID:                 id,
		ChannelID:          channelID,
		UserID:             userID,
		Status:             models.StatusResolved,
		Category:           category,
		IngameNickname:     nickname,
		CategoryDetails:    categoryDetails,
		AdditionalInfo:     additionalInfo,
		ProblemDescription: problemDesc,
		CreatedAt:          now,
	}, nil
}

// GetTicketByChannelID retrieves a ticket by Discord channel ID
func (d *DB) GetTicketByChannelID(channelID string) (*models.Ticket, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `SELECT id, channel_id, user_id, status, category, ingame_nickname, category_details, additional_info, problem_description, created_at
			  FROM tickets WHERE channel_id = ? LIMIT 1`

	var t models.Ticket
	var statusStr string
	err := d.conn.QueryRow(query, channelID).Scan(
		&t.ID,
		&t.ChannelID,
		&t.UserID,
		&statusStr,
		&t.Category,
		&t.IngameNickname,
		&t.CategoryDetails,
		&t.AdditionalInfo,
		&t.ProblemDescription,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTicketNotFound
		}
		return nil, fmt.Errorf("failed to query ticket by channel: %w", err)
	}
	t.Status = models.TicketStatus(statusStr)
	return &t, nil
}

// GetTicketByID retrieves a ticket by primary key ID
func (d *DB) GetTicketByID(id int64) (*models.Ticket, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `SELECT id, channel_id, user_id, status, category, ingame_nickname, category_details, additional_info, problem_description, created_at
			  FROM tickets WHERE id = ? LIMIT 1`

	var t models.Ticket
	var statusStr string
	err := d.conn.QueryRow(query, id).Scan(
		&t.ID,
		&t.ChannelID,
		&t.UserID,
		&statusStr,
		&t.Category,
		&t.IngameNickname,
		&t.CategoryDetails,
		&t.AdditionalInfo,
		&t.ProblemDescription,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTicketNotFound
		}
		return nil, fmt.Errorf("failed to query ticket by id: %w", err)
	}
	t.Status = models.TicketStatus(statusStr)
	return &t, nil
}

// UpdateTicketStatus updates the status of a ticket
func (d *DB) UpdateTicketStatus(channelID string, status models.TicketStatus) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE tickets SET status = ? WHERE channel_id = ?`
	res, err := d.conn.Exec(query, string(status), channelID)
	if err != nil {
		return fmt.Errorf("failed to update ticket status: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTicketNotFound
	}
	return nil
}

// UpdateTicketInfo saves all modal-submitted fields in one shot and advances the ticket to RESOLVED
func (d *DB) UpdateTicketInfo(channelID, category, nickname, categoryDetails, additionalInfo, problemDesc string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE tickets
			  SET category = ?, ingame_nickname = ?, category_details = ?, additional_info = ?, problem_description = ?, status = ?
			  WHERE channel_id = ?`
	res, err := d.conn.Exec(query, category, nickname, categoryDetails, additionalInfo, problemDesc, string(models.StatusResolved), channelID)
	if err != nil {
		return fmt.Errorf("failed to update ticket info: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTicketNotFound
	}
	return nil
}

// CountActiveTicketsForUser returns the number of tickets a user has that are not closed.
func (d *DB) CountActiveTicketsForUser(userID string) (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `SELECT COUNT(*) FROM tickets WHERE user_id = ? AND status != ?`
	var count int
	err := d.conn.QueryRow(query, userID, string(models.StatusClosed)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active tickets: %w", err)
	}
	return count, nil
}

// GetActiveTicketForUser returns the most recently created non-closed ticket for a user.
func (d *DB) GetActiveTicketForUser(userID string) (*models.Ticket, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `SELECT id, channel_id, user_id, status, category, ingame_nickname, category_details, additional_info, problem_description, created_at
			  FROM tickets WHERE user_id = ? AND status != ? ORDER BY created_at DESC LIMIT 1`
	var t models.Ticket
	var status string
	err := d.conn.QueryRow(query, userID, string(models.StatusClosed)).Scan(
		&t.ID, &t.ChannelID, &t.UserID, &status, &t.Category, &t.IngameNickname,
		&t.CategoryDetails, &t.AdditionalInfo, &t.ProblemDescription, &t.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query active ticket: %w", err)
	}
	t.Status = models.TicketStatus(status)
	return &t, nil
}

// GetActiveTickets returns every ticket that has not been closed yet.
func (d *DB) GetActiveTickets() ([]models.Ticket, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `SELECT id, channel_id, user_id, status, category, ingame_nickname, category_details, additional_info, problem_description, created_at
			  FROM tickets WHERE status != ? ORDER BY created_at DESC`
	rows, err := d.conn.Query(query, string(models.StatusClosed))
	if err != nil {
		return nil, fmt.Errorf("failed to query active tickets: %w", err)
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var t models.Ticket
		var statusStr string
		if err := rows.Scan(
			&t.ID,
			&t.ChannelID,
			&t.UserID,
			&statusStr,
			&t.Category,
			&t.IngameNickname,
			&t.CategoryDetails,
			&t.AdditionalInfo,
			&t.ProblemDescription,
			&t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan active ticket: %w", err)
		}
		t.Status = models.TicketStatus(statusStr)
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate active tickets: %w", err)
	}

	return tickets, nil
}
