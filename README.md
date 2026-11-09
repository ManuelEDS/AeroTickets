# ✈️ AeroBot — Discord Support Ticket Bot in Go

A robust, interactive Discord support ticket management bot written in **Go (Golang)** with modal-based ticket forms, private channel provisioning, SQLite state persistence, and Markdown (`.md`) transcript generation upon ticket closure.

---

## 🌟 Key Features

* **⚡ Interactive Ticket Panel**: Deploy an embed with six category buttons (Griefing, Ship, Bug, Crash, Appeals, Other) in any public channel using `/setup-panel`.
* **🟢 Staff Availability Role**: The bot automatically creates the `available staff` role. Members holding any of the configured staff roles can self-assign it with `/available` or remove it with `/unavailable`, and the role is pinged whenever a new ticket is opened.
* **🔒 Private Channel Provisioning**: Creates isolated ticket channels under your specified category with locked-down permissions (`@everyone` denied; ticket creator and staff roles granted access).
* **📝 Category-Specific Modal Forms**: Each panel button opens a modal tailored to that category:
  - Griefing: nickname, incident coordinates, optional additional info
  - Ship: nickname, ship claim name, optional additional info
  - Bug: nickname, problem description
  - Crash: nickname, crash logs, optional additional info
  - Appeals: nickname, punishment being appealed, reason the punishment should be revoked
  - Other: nickname, problem description
  - Private channel and staff notification are created only after the form is submitted (`RESOLVED`)
* **⚖️ Appeals Category**: A dedicated appeal flow that asks which punishment is being appealed and why it should be revoked, so staff can review the case with all the context up front.
* **📄 Markdown Transcript Archive & User DM**: When a ticket is closed (`🔴 Close Ticket` -> `⚠️ Confirm Closure`):
  - All messages and attachments are formatted into a clean `.md` document and saved locally on disk.
  - The record is registered in SQLite (`transcripts_log`).
  - The `.md` file is uploaded to the server's staff transcript archive channel.
  - **Direct Message (DM) to User**: The bot sends a direct message to the user who created the ticket, informing them that their ticket has been closed and attaching the `.md` transcript file for their personal records.
* **🛡️ Concurrency-Safe SQLite**: Pure Go SQLite implementation (`modernc.org/sqlite`, zero CGO requirements) with WAL mode and thread-safe operations.
* **🐳 Production Ready**: Includes multi-stage `Dockerfile` and `docker-compose.yml` for VPS / Linux deployment.

---

## 🆕 Recent Updates

* **Appeals ticket category**: new panel button and modal that collect the in-game nickname, the punishment being appealed, and the reason it should be revoked.
* **`available staff` role**: created automatically by the bot. Eligible staff can self-assign it with `/available` and remove it with `/unavailable`, and it is pinged every time a new ticket is opened.
* **New permission required**: `Manage Roles` is now required so the bot can create the `available staff` role and add/remove it from members.

---

## 📁 Project Structure

```
AeroBot/
├── cmd/
│   └── bot/
│       └── main.go                 # Application entry point & graceful shutdown
├── internal/
│   ├── bot/
│   │   ├── bot.go                  # Discord session lifecycle, transcript upload & orphan cleanup
│   │   ├── commands.go             # Slash commands (/setup-panel, /close, /ticket-info, /add-user, /available, /unavailable)
│   │   ├── interactions.go         # Buttons, category modals & close/confirm handlers
│   │   └── views.go                # Embeds, modals and interactive UI components
│   ├── config/
│   │   ├── config.go               # Configuration loader & validation
│   │   └── config_test.go          # Config unit tests
│   ├── database/
│   │   ├── db.go                   # SQLite connection with WAL mode & migrations
│   │   ├── db_test.go              # Database unit tests
│   │   ├── ticket_repo.go          # Ticket queries & updates
│   │   └── transcript_repo.go      # Transcript logging queries
│   ├── models/
│   │   └── ticket.go               # Domain models and status constants
│   └── transcript/
│       ├── generator.go            # Markdown transcript formatter & disk writer
│       └── generator_test.go       # Transcript generation unit tests
├── transcripts/                    # Directory where .md transcripts are stored
├── config.example.json             # Sample configuration file
├── Dockerfile                      # Multi-stage container build
├── docker-compose.yml              # Docker Compose deployment definition
├── go.mod
├── go.sum
└── README.md
```

---

## ⚙️ Configuration

Copy `config.example.json` to `config.json`:

```bash
cp config.example.json config.json
```

### `config.json` Schema

```json
{
  "bot_token": "YOUR_DISCORD_BOT_TOKEN",
  "guild_id": "123456789012345678",
  "category_id": "123456789012345678",
  "transcript_channel_id": "123456789012345678",
  "transcript_folder_path": "./transcripts",
  "staff_role_ids": [
    "123456789012345678",
    "234567890123456789"
  ],
  "database_path": "./aerobot.db"
}
```

### Environment Variable Overrides

All settings can alternatively be provided or overridden via environment variables:

| Environment Variable | Description |
| :--- | :--- |
| `BOT_TOKEN` / `DISCORD_BOT_TOKEN` | Discord Bot Token |
| `GUILD_ID` / `DISCORD_GUILD_ID` | Main Discord Guild / Server ID |
| `CATEGORY_ID` / `DISCORD_CATEGORY_ID` | Category ID where ticket channels will be placed |
| `TRANSCRIPT_CHANNEL_ID` | Discord Channel ID where `.md` transcripts are uploaded |
| `TRANSCRIPT_FOLDER_PATH` | Local file system folder to save transcripts (default: `./transcripts`) |
| `STAFF_ROLE_IDS` | Comma-separated list of Staff Role IDs (e.g. `111,222`) |
| `DATABASE_PATH` | SQLite file path (default: `./aerobot.db`) |

---

## 🔑 Discord Developer Portal Setup

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications) and create a **New Application**.
2. Navigate to the **Bot** tab:
   - Click **Reset Token** and copy the token to your `config.json`.
   - Under **Privileged Gateway Intents**, enable **Message Content Intent**. Transcript generation fetches message content from ticket channels. The bot does not need the Server Members or Presence intents.
3. Navigate to **OAuth2 -> URL Generator**:
   - **Scopes**: `bot`, `applications.commands`
   - **Bot Permissions**:
     - `Manage Channels`
     - `Manage Roles` *(new: required to create the `available staff` role and add/remove it from members)*
     - `View Channels`
     - `Send Messages`
     - `Send Messages in Threads`
     - `Embed Links`
     - `Attach Files`
     - `Read Message History`
     - `Use External Emojis`
4. Copy the generated invite link and invite the bot to your server.

### Required Permissions Explained

| Permission | Used for |
| :--- | :--- |
| `Manage Channels` | Creating/deleting ticket channels, applying permission overwrites, and deploying the panel with `/setup-panel`. |
| `Manage Roles` | Creating the `available staff` role and adding/removing it when members use `/available` or `/unavailable`. |
| `View Channels` | Accessing ticket channels and the transcript archive channel. |
| `Send Messages` | Posting ticket summaries, welcome messages, and status updates. |
| `Send Messages in Threads` | Posting in ticket threads if threads are used. |
| `Embed Links` | Sending ticket summary and transcript embeds. |
| `Attach Files` | Uploading `.md` transcript files to the archive channel and user DMs. |
| `Read Message History` | Fetching ticket messages to build the transcript on closure. |
| `Use External Emojis` | Rendering the emojis used in the panel, buttons, and modals. |

> **Role hierarchy:** the bot's own role must be positioned **above** `available staff` to assign or remove it. When the bot creates the role it is placed at the bottom automatically, so no action is needed on first run. If the role already exists higher in the hierarchy, move the bot's role above it.
>
> If the bot cannot create the role, check that it has `Manage Roles` and that the server has not reached its role limit; the bot will log a warning and `/available` will report that the role is not ready.

---

## 🚀 Running the Bot

### Option 1: Run Locally with Go (v1.25+)

```bash
# 1. Download dependencies
go mod download

# 2. Run unit tests
go test ./...

# 3. Build executable
go build -o aerobot ./cmd/bot

# 4. Start bot
./aerobot -config config.json
```

### Option 2: Docker Compose (Recommended for VPS)

```bash
# 1. Ensure config.json is populated
cp config.example.json config.json
# Edit config.json with your Discord IDs

# 2. Launch container in background
docker compose up -d --build

# 3. View live logs
docker compose logs -f
```

---

## 💬 Commands & Usage

### Command Reference

| Command | Description | Who can use it |
| :--- | :--- | :--- |
| `/setup-panel` | Deploy the interactive ticket creation panel in the current channel. | Administrator / `Manage Channels` |
| `/close` | Request to close the current ticket (asks for confirmation, then archives a transcript). | Anyone in a ticket channel |
| `/ticket-info` | Show a summary of the current ticket. | Anyone in a ticket channel |
| `/add-user user:@member` | Give a member access to the current ticket channel. | Configured staff / Administrator / `Manage Channels` |
| `/available` | Self-assign the `available staff` role and receive pings for new tickets. | Members with any role in `staff_role_ids` |
| `/unavailable` | Remove the `available staff` role and stop receiving new ticket pings. | Members with any role in `staff_role_ids` |

### 1. Deploy Ticket Panel
In the channel where you want users to create tickets (e.g., `#support-tickets`), run:
```
/setup-panel
```
*(Requires Administrator or Manage Channels permission)*

### 2. User Ticket Flow
1. User clicks one of the six category buttons on the panel.
2. AeroBot opens the modal for that category with only the relevant fields.
3. After the form is submitted, AeroBot creates the private channel `ticket-<id>-<category>` (e.g. `ticket-0052-ship`) and saves the ticket as `RESOLVED`.
4. A summary card is posted in the new channel, pinging the `available staff` role and showing the **`🔴 Close Ticket`** control.

### 3. Closing a Ticket
1. Click **`🔴 Close Ticket`** or run `/close`.
2. Click **`⚠️ Confirm Closure`**.
3. AeroBot generates the `.md` transcript, saves it to `./transcripts/ticket-<zero-padded id>-<user>.md` (e.g. `ticket-0052-ShadowPilot_1234.md`), uploads the file to the transcript archive channel, and deletes the ticket channel.

### 4. Adding Someone to a Ticket
In a ticket channel, configured staff or members with Administrator / Manage Channels permission can run:
```
/add-user user:@member
```
The selected member receives permission to view and participate in that ticket.

### 5. Staff Availability
AeroBot creates and maintains the `available staff` role automatically. Members with any role listed in `staff_role_ids` can toggle their own availability:
```
/available    # Receive a ping whenever a new ticket is opened
/unavailable  # Stop receiving new ticket pings
```

---

## 📄 Transcript Output Sample

```markdown
# TICKET TRANSCRIPT: #ticket-shadowpilot
- **Ticket ID:** 1
- **User:** ShadowPilot#1234 (123456789012345678)
- **In-Game Nickname:** ShadowPilot
- **Category:** Ship Related
- **Ship Claim Name:** Dreadnought NX-01
- **Additional Info:** Claimed
- **Problem Description:** N/A
- **Closed At:** 2026-09-11 19:30:00 (UTC)

---
## MESSAGE HISTORY

[2026-09-11 19:25:01] AeroBot#0000: Hello @ShadowPilot, welcome to your support ticket (#1)...
[2026-09-11 19:25:20] ShadowPilot#1234: ShadowPilot
[2026-09-11 19:25:22] AeroBot#0000: Thank you, ShadowPilot! What is the ticket about?...
[2026-09-11 19:25:40] ShadowPilot#1234: Dreadnought NX-01 (Claimed)
[2026-09-11 19:26:05] ShadowPilot#1234: Warp core injector malfunctioned after undocking [Attachment: bug_screenshot.png (https://...)]
[2026-09-11 19:28:10] StaffMember#5678: Hello, I have reset your ship's coordinates. You can log back in.
```

---

## 🧪 Testing

Run all unit tests across configuration, database, and transcript formatting packages:

```bash
go test -v ./...
```

---

## 📜 License

MIT License. Designed and built with Go & `discordgo`.
