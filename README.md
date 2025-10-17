migration steps
run river migrations
- go install github.com/riverqueue/river/cmd/river@latest
- river migrate-up --database-url "postgresql://postgres:postgres2025@localhost:5432/yantra"

run gorm migrations
- setup .env DATABASE_URL
- go run cmd/migrate/main.go