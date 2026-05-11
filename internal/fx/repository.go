package appfx

import (
	"file-service/internal/domain"
	writerepo "file-service/internal/infra/write/scylla"
	"github.com/ofm-microservices/ofm-common/pkg/logging"

	"github.com/gocql/gocql"
	"go.uber.org/fx"
)

// RepoModule provides concrete persistence adapters behind the repository
// interfaces used by the application layer.
var RepoModule = fx.Options(
	fx.Provide(ProvideFileRepository),
)

// ProvideFileRepository constructs the Scylla-backed file repository.
func ProvideFileRepository(db *gocql.Session, lg logging.Logger) (domain.FileRepository, error) {
	return writerepo.New(db, lg)
}
