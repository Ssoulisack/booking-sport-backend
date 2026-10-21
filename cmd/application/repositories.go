package application

import (
	"backend/internal/adapters/secondary/mailer"
	"backend/internal/adapters/secondary/minio"
	"backend/internal/adapters/secondary/postgres"
	"backend/internal/adapters/secondary/redis"
	"backend/internal/adapters/secondary/storage"
	"backend/internal/config"
	"backend/internal/core/domain"
	"backend/internal/core/port/output"
	"log"
)

// Repositories holds all data storage interfaces
type AppRepositories struct {
	// System repositories
	// mysql    mysql.MySQLRepository
	redis    redis.CacheRepository
	minio    minio.MinIORepository
	mailer   output.Mailer
	postgres postgres.PostgresRepository
	storage  output.FileStorage
	// app settings the services need (links in emails)
	app config.AppConfig
	// Domain repositories
	user           output.UserRepository
	plans          output.PlanRepository
	businesses     output.BusinessRepository
	memberships    output.MembershipRepository
	invitations    output.InvitationRepository
	passwordResets output.PasswordResetRepository
	media          output.MediaRepository
	supportVisits  output.SupportVisitRepository
	sports         output.SportRepository
	resources      output.ResourceRepository
	coaches        output.CoachRepository
	openingHours   output.OpeningHourRepository
	closures       output.ClosureRepository
	prices         output.PriceRepository
	classSessions  output.ClassSessionRepository
	rules          output.RuleRepository
}

// setupRepositories initializes all data storage connections
func SetupRepositories(cfg *config.Config) *AppRepositories {
	// MySQL setup
	// mysqlRepo, err := mysql.NewMySQLRepository(cfg.Database)
	// if err != nil {
	// 	log.Fatalf("❌ Failed to initialize MySQL repository: %v", err)
	// }

	// PostgreSQL setup
	postgresRepo, err := postgres.NewPostgresRepository(cfg.Postgres)
	if err != nil {
		log.Fatalf("❌ Failed to initialize PostgreSQL repository: %v", err)
	}

	// Redis setup
	redisRepo, err := redis.NewRedisRepository(cfg.Redis)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Redis repository: %v", err)
	}

	// MinIO setup
	minioRepo, err := minio.NewMinIORepository(cfg.Minio)
	if err != nil {
		log.Fatalf("❌ Failed to initialize MinIO repository: %v", err)
	}

	// Domain repositories
	db := postgresRepo.GetDB()

	return &AppRepositories{
		// mysql:  mysqlRepo,
		redis:          redisRepo,
		minio:          minioRepo,
		mailer:         mailer.NewPlatformMailer(cfg.Mailer, cfg.App.Name),
		postgres:       postgresRepo,
		storage:        storage.NewSupabaseStorage(cfg.Storage),
		app:            cfg.App,
		user:           postgres.NewUserRepository(db),
		plans:          postgres.NewPlanRepository(db),
		businesses:     postgres.NewBusinessRepository(db),
		memberships:    postgres.NewMembershipRepository(db),
		invitations:    postgres.NewInvitationRepository(db),
		passwordResets: postgres.NewPasswordResetRepository(db),
		media:          postgres.NewMediaRepository(db),
		supportVisits:  postgres.NewSupportVisitRepository(db),
		sports:         postgres.NewScopedRepository[domain.Sport](db, "name"),
		resources:      postgres.NewScopedRepository[domain.Resource](db, "name"),
		coaches:        postgres.NewCoachRepository(db),
		openingHours:   postgres.NewOpeningHourRepository(db),
		closures:       postgres.NewScopedRepository[domain.Closure](db, ""),
		prices:         postgres.NewPriceRepository(db),
		classSessions:  postgres.NewClassSessionRepository(db),
		rules:          postgres.NewRuleRepository(db),
	}
}

// closeRepositories gracefully closes all connections
func CloseRepositories(repos *AppRepositories) {
	// if repos.mysql != nil {
	// 	repos.mysql.Close()
	// }
	if repos.redis != nil {
		repos.redis.Close()
	}
	if repos.postgres != nil {
		repos.postgres.Close()
	}
}
