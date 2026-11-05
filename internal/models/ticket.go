package models

import (
	"fmt"
	"time"
)

// TicketStatus represents the state of a support ticket
type TicketStatus string

const (
	StatusOpen     TicketStatus = "OPEN"
	StatusResolved TicketStatus = "RESOLVED"
	StatusClosed   TicketStatus = "CLOSED"
)

// TicketCategory represents the category of the issue
type TicketCategory string

const (
	CategoryGriefing TicketCategory = "Griefing / Stealing"
	CategoryShip     TicketCategory = "Ship Related"
	CategoryBug      TicketCategory = "Bug / Exploit Report"
	CategoryCrash    TicketCategory = "Crash Report"
	CategoryOther    TicketCategory = "Other"
)

func CategoryDetailLabel(category string) string {
	switch TicketCategory(category) {
	case CategoryGriefing:
		return "Coordinates"
	case CategoryShip:
		return "Ship Claim Name"
	case CategoryCrash:
		return "Crash Logs"
	default:
		return ""
	}
}

func CategoryAbbreviation(category string) string {
	switch TicketCategory(category) {
	case CategoryGriefing:
		return "grief"
	case CategoryShip:
		return "ship"
	case CategoryBug:
		return "bug"
	case CategoryCrash:
		return "crash"
	case CategoryOther:
		return "other"
	default:
		return "ticket"
	}
}

func TicketChannelName(id int64, category string) string {
	return fmt.Sprintf("ticket-%04d-%s", id, CategoryAbbreviation(category))
}

// Ticket represents a support ticket in the database
type Ticket struct {
	ID                 int64        `json:"id"`
	ChannelID          string       `json:"channel_id"`
	UserID             string       `json:"user_id"`
	Status             TicketStatus `json:"status"`
	Category           string       `json:"category"`
	IngameNickname     string       `json:"ingame_nickname"`
	CategoryDetails    string       `json:"category_details"`
	AdditionalInfo     string       `json:"additional_info"`
	ProblemDescription string       `json:"problem_description"`
	CreatedAt          time.Time    `json:"created_at"`
}

// TranscriptLog represents a transcript record in the database
type TranscriptLog struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticket_id"`
	FilePath  string    `json:"file_path"`
	CreatedAt time.Time `json:"created_at"`
}
