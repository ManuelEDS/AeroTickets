package database_test

import (
	"path/filepath"
	"testing"

	"aerobot/internal/database"
	"aerobot/internal/models"
)

func TestDatabaseTicketOperations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_aerobot.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}
	defer db.Close()

	// 1. Create Ticket — initial status must be OPEN
	ticket, err := db.CreateTicket("chan-1001", "user-2001")
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}
	if ticket.ID <= 0 {
		t.Errorf("expected positive ticket ID, got %d", ticket.ID)
	}
	if ticket.Status != models.StatusOpen {
		t.Errorf("expected initial status %s, got %s", models.StatusOpen, ticket.Status)
	}

	// 2. Fetch Ticket by Channel ID
	fetched, err := db.GetTicketByChannelID("chan-1001")
	if err != nil {
		t.Fatalf("failed to get ticket by channel ID: %v", err)
	}
	if fetched.ID != ticket.ID || fetched.UserID != "user-2001" {
		t.Errorf("fetched ticket mismatch: %+v", fetched)
	}

	// 3. UpdateTicketInfo — simulates modal submission (category + nickname + extra + desc -> RESOLVED)
	err = db.UpdateTicketInfo(
		"chan-1001",
		string(models.CategoryShip),
		"CaptainNemo",
		"Millennium Falcon (Claimed)",
		"",
		"Hyperdrive hyper-engine failure in sector 7",
	)
	if err != nil {
		t.Fatalf("failed to update ticket info: %v", err)
	}

	fetched, _ = db.GetTicketByChannelID("chan-1001")
	if fetched.IngameNickname != "CaptainNemo" {
		t.Errorf("expected nickname CaptainNemo, got %s", fetched.IngameNickname)
	}
	if fetched.Category != string(models.CategoryShip) {
		t.Errorf("expected category %s, got %s", models.CategoryShip, fetched.Category)
	}
	if fetched.CategoryDetails != "Millennium Falcon (Claimed)" {
		t.Errorf("expected category details, got %s", fetched.CategoryDetails)
	}
	if fetched.ProblemDescription != "Hyperdrive hyper-engine failure in sector 7" {
		t.Errorf("expected problem description, got %s", fetched.ProblemDescription)
	}
	if fetched.Status != models.StatusResolved {
		t.Errorf("expected status RESOLVED after UpdateTicketInfo, got %s", fetched.Status)
	}

	// 4. Check Active Ticket for User
	active, err := db.GetActiveTicketForUser("user-2001")
	if err != nil || active == nil {
		t.Fatalf("expected active ticket for user-2001, got err: %v, active: %v", err, active)
	}

	// 5. Log Transcript
	transcriptPath := filepath.Join(tempDir, "ticket-1-user.md")
	tLog, err := db.LogTranscript(ticket.ID, transcriptPath)
	if err != nil {
		t.Fatalf("failed to log transcript: %v", err)
	}
	if tLog.ID <= 0 {
		t.Errorf("expected positive transcript log ID, got %d", tLog.ID)
	}

	logs, err := db.GetTranscriptsByTicketID(ticket.ID)
	if err != nil || len(logs) == 0 {
		t.Fatalf("failed to get transcripts by ticket ID: %v, logs: %v", err, logs)
	}

	// 6. Close Ticket
	err = db.UpdateTicketStatus("chan-1001", models.StatusClosed)
	if err != nil {
		t.Fatalf("failed to update ticket to closed: %v", err)
	}

	// 7. Check Active Ticket for User is now nil after closing
	activeAfterClose, err := db.GetActiveTicketForUser("user-2001")
	if err != nil {
		t.Fatalf("error querying active ticket after close: %v", err)
	}
	if activeAfterClose != nil {
		t.Errorf("expected no active ticket after close, got: %+v", activeAfterClose)
	}
}

func TestCreateTicketWithInfo(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.New(filepath.Join(tempDir, "test_create_with_info.db"))
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}
	defer db.Close()

	ticket, err := db.CreateTicketWithInfo(
		"chan-3001",
		"user-4001",
		string(models.CategoryGriefing),
		"StarPilot99",
		"X: 1024, Y: -300, Z: 512",
		"Another player stole my ship",
		"",
	)
	if err != nil {
		t.Fatalf("failed to create ticket with info: %v", err)
	}
	if ticket.ID <= 0 {
		t.Errorf("expected positive ticket ID, got %d", ticket.ID)
	}
	if ticket.Status != models.StatusResolved {
		t.Errorf("expected status %s, got %s", models.StatusResolved, ticket.Status)
	}

	fetched, err := db.GetTicketByChannelID("chan-3001")
	if err != nil {
		t.Fatalf("failed to fetch created ticket: %v", err)
	}
	if fetched.Category != string(models.CategoryGriefing) {
		t.Errorf("expected category %s, got %s", models.CategoryGriefing, fetched.Category)
	}
	if fetched.IngameNickname != "StarPilot99" {
		t.Errorf("expected nickname StarPilot99, got %s", fetched.IngameNickname)
	}
	if fetched.CategoryDetails != "X: 1024, Y: -300, Z: 512" {
		t.Errorf("expected category details to match, got %s", fetched.CategoryDetails)
	}
	if fetched.AdditionalInfo != "Another player stole my ship" {
		t.Errorf("expected additional info to match, got %s", fetched.AdditionalInfo)
	}
	if fetched.Status != models.StatusResolved {
		t.Errorf("expected fetched status %s, got %s", models.StatusResolved, fetched.Status)
	}

	active, err := db.GetActiveTicketForUser("user-4001")
	if err != nil {
		t.Fatalf("failed to check active ticket: %v", err)
	}
	if active == nil || active.ID != ticket.ID {
		t.Errorf("expected active ticket %d, got %+v", ticket.ID, active)
	}
}

func TestGetActiveTickets(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.New(filepath.Join(tempDir, "test_active.db"))
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}
	defer db.Close()

	if _, err := db.CreateTicket("chan-a", "user-a"); err != nil {
		t.Fatalf("failed to create first ticket: %v", err)
	}
	second, err := db.CreateTicket("chan-b", "user-b")
	if err != nil {
		t.Fatalf("failed to create second ticket: %v", err)
	}

	active, err := db.GetActiveTickets()
	if err != nil {
		t.Fatalf("failed to get active tickets: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("expected 2 active tickets, got %d", len(active))
	}

	if err := db.UpdateTicketStatus("chan-a", models.StatusClosed); err != nil {
		t.Fatalf("failed to close first ticket: %v", err)
	}

	active, err = db.GetActiveTickets()
	if err != nil {
		t.Fatalf("failed to get active tickets after close: %v", err)
	}
	if len(active) != 1 || active[0].ID != second.ID {
		t.Errorf("expected only ticket %d to remain active, got %+v", second.ID, active)
	}
}
