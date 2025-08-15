package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database used for local data storage
type Store struct {
	db *sql.DB
}

// Open opens (and creates/migrates) the database at the given path
func Open(ctx context.Context, dbPath string) (*Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, fmt.Errorf("empty database path")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, fmt.Errorf("create database dir: %w", err)
	}
	// Ensure file exists with strict perms
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		f, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, fmt.Errorf("create database file: %w", err)
		}
		f.Close()
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// Pragmas
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set WAL: %w", err)
	}
	_, _ = db.ExecContext(ctx, "PRAGMA foreign_keys=ON;")
	_, _ = db.ExecContext(ctx, "PRAGMA busy_timeout=5000;")
	_, _ = db.ExecContext(ctx, "PRAGMA synchronous=NORMAL;")

	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	// user_version based migrations
	var ver int
	_ = s.db.QueryRowContext(ctx, "PRAGMA user_version;").Scan(&ver)

	// v1: ai_summaries table
	if ver == 0 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS ai_summaries (
  account_email TEXT NOT NULL,
  message_id    TEXT NOT NULL,
  summary       TEXT NOT NULL,
  updated_at    INTEGER NOT NULL,
  PRIMARY KEY (account_email, message_id)
);
`)
		if err == nil {
			_, err = tx.ExecContext(ctx, "PRAGMA user_version=1;")
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate v1: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ver = 1
	}

	// v2: placeholder migration for existing v2 databases
	if ver == 1 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "PRAGMA user_version=2;")
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate v2: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ver = 2
	}

	// v3: prompt templates and results
	if ver == 2 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS prompt_templates (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  name          TEXT NOT NULL UNIQUE,
  description   TEXT,
  prompt_text   TEXT NOT NULL,
  category      TEXT NOT NULL DEFAULT 'summary',
  created_at    INTEGER NOT NULL,
  is_favorite   BOOLEAN DEFAULT FALSE,
  usage_count   INTEGER DEFAULT 0
);
`)
		if err == nil {
			_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS prompt_results (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  account_email TEXT NOT NULL,
  message_id    TEXT NOT NULL,
  prompt_id     INTEGER NOT NULL,
  result_text   TEXT NOT NULL,
  created_at    INTEGER NOT NULL,
  FOREIGN KEY (prompt_id) REFERENCES prompt_templates(id)
);
`)
		}
		if err == nil {
			// Insert default prompts
			_, err = tx.ExecContext(ctx, `
INSERT INTO prompt_templates (name, description, prompt_text, category, created_at, is_favorite) VALUES
('Quick Summary', 'Brief 2-3 bullet point summary', 'Summarize this email in 2-3 bullet points:\n\n{{body}}', 'summary', ?, TRUE),
('Action Items', 'Extract specific action items and deadlines', 'Extract specific action items and deadlines from this email:\n\n{{body}}', 'analysis', ?, TRUE),
('Key Decisions', 'Identify key decisions or conclusions', 'What key decisions or conclusions are mentioned in this email?\n\n{{body}}', 'analysis', ?, FALSE),
('Meeting Summary', 'Summarize meeting details', 'Summarize the meeting details, attendees, and key points from this email:\n\n{{body}}', 'summary', ?, FALSE);
`, time.Now().Unix(), time.Now().Unix(), time.Now().Unix(), time.Now().Unix())
		}
		if err == nil {
			_, err = tx.ExecContext(ctx, "PRAGMA user_version=3;")
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate v3: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ver = 3
	}

	// v4: bulk analysis prompts
	if ver == 3 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		// Insert bulk analysis prompts one by one to avoid SQL formatting issues
		bulkPrompts := []struct {
			name, description, promptText string
		}{
			{
				"Cloud Product Analysis",
				"Analyze cloud product updates and extract relevant information about specific services",
				"You are analyzing a collection of cloud product update emails. Focus on extracting and summarizing information about cloud services, new features, and product announcements.\n\nEmails to analyze:\n{{messages}}\n\nPlease provide a comprehensive analysis including:\n1. **New Product Features**: List any new features or capabilities mentioned\n2. **Service Updates**: Document any service improvements or changes\n3. **AI/ML Services**: Highlight any updates related to AI, machine learning, or Bedrock services\n4. **Pricing Changes**: Note any pricing updates or new pricing models\n5. **Regional Availability**: Document any new region launches or availability changes\n6. **Integration Updates**: List any new integrations or API changes\n7. **Security & Compliance**: Note any security enhancements or compliance updates\n\nFormat your response clearly with bullet points and sections.",
			},
			{
				"Newsletter Digest",
				"Create a concise digest summarizing the key points from multiple newsletter emails",
				"You are creating a digest from multiple newsletter emails. Extract the most important information and create a concise summary.\n\nEmails to analyze:\n{{messages}}\n\nPlease create a digest with:\n1. **Top Headlines**: 3-5 most important stories or announcements\n2. **Key Updates**: Significant changes or new information\n3. **Action Items**: Any items requiring attention or follow-up\n4. **Trends**: Patterns or recurring themes across the emails\n5. **Summary**: 2-3 sentence executive summary\n\nKeep the digest concise and actionable.",
			},
			{
				"Technical Updates Summary",
				"Summarize technical updates and changes from multiple technical emails",
				"You are analyzing technical update emails to extract key technical changes and improvements.\n\nEmails to analyze:\n{{messages}}\n\nPlease provide a technical summary including:\n1. **API Changes**: Any new endpoints, deprecations, or breaking changes\n2. **Performance Improvements**: Speed, efficiency, or scalability enhancements\n3. **New Integrations**: Third-party service connections or partnerships\n4. **Security Updates**: Security patches, authentication changes, or compliance updates\n5. **Developer Experience**: Tools, SDKs, or development workflow improvements\n6. **Infrastructure Changes**: Platform updates, deployment changes, or architecture improvements\n7. **Migration Notes**: Any required actions for existing users\n\nFormat with clear technical details and impact assessment.",
			},
			{
				"Business Intelligence Report",
				"Extract business insights and strategic information from multiple business emails",
				"You are analyzing business emails to extract strategic insights and business intelligence.\n\nEmails to analyze:\n{{messages}}\n\nPlease provide a business intelligence report including:\n1. **Market Trends**: Industry developments or market shifts\n2. **Competitive Intelligence**: Competitor activities or positioning\n3. **Strategic Initiatives**: New business directions or partnerships\n4. **Financial Updates**: Revenue, investment, or cost information\n5. **Customer Insights**: User feedback, adoption metrics, or satisfaction data\n6. **Risk Factors**: Potential challenges or concerns\n7. **Opportunities**: New market opportunities or growth areas\n8. **Recommendations**: Strategic actions or next steps\n\nFormat as a business report with clear insights and actionable recommendations.",
			},
			{
				"Event & Conference Summary",
				"Summarize information from multiple event-related emails",
				"You are analyzing event and conference emails to create a comprehensive summary.\n\nEmails to analyze:\n{{messages}}\n\nPlease provide an event summary including:\n1. **Upcoming Events**: Dates, locations, and key details\n2. **Registration Deadlines**: Important dates and requirements\n3. **Featured Speakers**: Key presenters and their topics\n4. **Session Highlights**: Notable sessions, workshops, or tracks\n5. **Networking Opportunities**: Meetups, social events, or community activities\n6. **Costs & Discounts**: Pricing, early bird offers, or special rates\n7. **Travel Information**: Venue details, accommodation, or transportation\n8. **Action Items**: Registration tasks, preparation requirements, or follow-ups\n\nFormat with clear event details and next steps.",
			},
		}

		for _, prompt := range bulkPrompts {
			_, err = tx.ExecContext(ctx, `
INSERT OR IGNORE INTO prompt_templates (name, description, prompt_text, category, created_at, is_favorite) 
VALUES (?, ?, ?, 'bulk_analysis', ?, TRUE)`,
				prompt.name, prompt.description, prompt.promptText, time.Now().Unix())
			if err != nil {
				break
			}
		}

		if err == nil {
			_, err = tx.ExecContext(ctx, "PRAGMA user_version=4;")
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate v4: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ver = 4
	}

	return nil
}

// Close closes the underlying database
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB returns the underlying sql.DB for use by domain stores
func (s *Store) DB() *sql.DB {
	return s.db
}
