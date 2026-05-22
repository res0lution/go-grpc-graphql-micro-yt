package database

import (
	"context"

	"iter"

	"portal-core/internal/config"
	"portal-core/internal/repository"
)

type Repositories struct {
	Jira     *repository.JiraRepository
	Feedback *repository.FeedbackRepository
	Notify   *repository.NotificationRepository
	SLA      *repository.SLARepository
	Metrics  *repository.MetricsRepository
	Mnemonic *repository.MnemonicRepository
	JiraSLA  *repository.JiraSLARepository
	Sessions *repository.SessionRepository
	Users    *repository.UserRepository
	Reasons  *repository.ReasonRepository
	Products *repository.ProductRepository
	Opers    *repository.ServiceOperationRepository
	Journal  *repository.JournalChangesRepository
	GitLinks *repository.GitLinksRepository
}

func (r Repositories) IterRepositories() iter.Seq[any] {
	repos := []any{
		r.Jira,
		r.Notify,
		r.Feedback,
		r.SLA,
		r.Metrics,
		r.Mnemonic,
		r.JiraSLA,
		r.Sessions,
		r.Users,
		r.Reasons,
		r.Products,
		r.Opers,
		r.Journal,
		r.GitLinks,
	}

	return func(yield func(any) bool) {
		for i := range repos {
			if !yield(repos[i]) {
				return
			}
		}
	}
}

// предполагаемая структура Db
type Db struct {
	inited   sync.Once
	pool     any
	logger   Logger
	Jira     *repository.JiraRepository
	Notify   *repository.NotificationRepository
	Feedback *repository.FeedbackRepository
	SLA      *repository.SLARepository
	Metrics  *repository.MetricsRepository
	Mnemonic *repository.MnemonicRepository
	JiraSLA  *repository.JiraSLARepository
	Sessions *repository.SessionRepository
	Users    *repository.UserRepository
	Reasons  *repository.ReasonRepository
	Products *repository.ProductRepository
	Opers    *repository.ServiceOperationRepository
	Journal  *repository.JournalChangesRepository
	GitLinks *repository.GitLinksRepository
}

func (db *Db) CreateRepositories(ctx context.Context, cfg *config.Config) {
	db.inited.Do(func() {
		db.Jira = repository.NewJiraRepository(db.pool)
		db.Notify = repository.NewNotificationRepository(db.pool)
		db.Feedback = repository.NewFeedbackRepository(db.pool)
		db.SLA = repository.NewSLARepository(db.pool)
		db.Metrics = repository.NewMetricsRepository(db.pool)
		db.Mnemonic = repository.NewMnemonicRepository(db.pool)

		db.JiraSLA = repository.NewJiraSLARepository(cfg.Jira.BaseURL, cfg.Jira.Token)

		db.Sessions = repository.NewSessionRepository(db.pool)
		db.Users = repository.NewUserRepository(db.pool)
		db.Reasons = repository.NewReasonRepository(db.pool)
		db.Products = repository.NewProductRepository(db.pool)
		db.Opers = repository.NewServiceOperationRepository(db.pool)

		db.Journal = repository.NewJournalChangesRepository(db.pool, 100)
		db.GitLinks = repository.NewGitLinksRepository(db.pool)
	})
}

type TableCreator interface {
	CreateTables(context.Context) error
}

func (db *Db) CreateTables(ctx context.Context) {
	for repo := range db.IterRepositories() {
		tc, ok := repo.(TableCreator)
		if !ok {
			continue
		}

		if err := tc.CreateTables(ctx); err != nil {
			db.logger.Error("create tables error", err)
			return
		}
	}

	db.logger.Debug("Table creation completed")
}

func (db *Db) InitTriggers(ctx context.Context) (err error) {
	tablesForTriggers := []string{
		"jira_tickets",
		"jira_ticket_reasons",
		"jira_ticket_reason_items",
		"jira_ticket_prerequisites",
	}

	if err = db.Journal.AddTriggerUpdateAtOnTable(ctx, "jira_tickets"); err != nil {
		return err
	}

	for _, table := range tablesForTriggers {
		if err = db.Journal.AddTriggerAuditOnTable(ctx, table); err != nil {
			return err
		}
	}

	return nil
}
