package appfx

import (
	"file-service/internal/domain"
	pgrepo "file-service/internal/infra/write/postgres"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"
)

// RepoModule provides concrete persistence adapters behind the repository
// interfaces used by the application layer.
var RepoModule = fx.Options(
	fx.Provide(ProvidePostgresFileRepository),
)

// ProvidePostgresFileRepository constructs the PostgreSQL-backed file repository.
func ProvidePostgresFileRepository(db *sqlx.DB, lg logging.Logger) (domain.FileRepository, error) {
	return pgrepo.New(db, lg)
}
