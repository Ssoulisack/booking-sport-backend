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
	mailer   mailer.MailerRepository
	postgres postgres.PostgresRepository
	storage  output.FileStorage
	// Domain repositories
	user          output.UserRepository
	sports        output.SportRepository
	resources     output.ResourceRepository
	coaches       output.CoachRepository
	openingHours  output.OpeningHourRepository
	closures      output.ClosureRepository
	prices        output.PriceRepository
	classSessions output.ClassSessionRepository
	rules         output.RuleRepository
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
	// Mailer setup
	mailerRepo := mailer.NewMailerRepository(cfg.Mailer)

	// Domain repositories
	db := postgresRepo.GetDB()
	userRepo := postgres.NewUserRepository(db)

	return &AppRepositories{
		// mysql:  mysqlRepo,
		redis:         redisRepo,
		minio:         minioRepo,
		storage:       storage.NewSupabaseStorage(cfg.Storage),
		user:          userRepo,
		mailer:        mailerRepo,
		postgres:      postgresRepo,
		sports:        postgres.NewCRUDRepository[domain.Sport](db, "name"),
		resources:     postgres.NewCRUDRepository[domain.Resource](db, "name"),
		coaches:       postgres.NewCoachRepository(db),
		openingHours:  postgres.NewOpeningHourRepository(db),
		closures:      postgres.NewCRUDRepository[domain.Closure](db, ""),
		prices:        postgres.NewPriceRepository(db),
		classSessions: postgres.NewCRUDRepository[domain.ClassSession](db, "title"),
		rules:         postgres.NewRuleRepository(db),
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
