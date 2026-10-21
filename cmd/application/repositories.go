package application

import (
	"backend/internal/adapters/secondary/mailer"
	"backend/internal/adapters/secondary/minio"
	"backend/internal/adapters/secondary/postgres"
	"backend/internal/adapters/secondary/redis"
	"backend/internal/config"
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
	// Domain repositories
	user output.UserRepository
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
	userRepo := postgres.NewUserRepository(postgresRepo.GetDB())

	return &AppRepositories{
		// mysql:  mysqlRepo,
		redis:    redisRepo,
		minio:    minioRepo,
		user:     userRepo,
		mailer:   mailerRepo,
		postgres: postgresRepo,
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
