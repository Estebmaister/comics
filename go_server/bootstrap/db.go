package bootstrap

import (
	"context"
	"errors"

	"comics/internal/repo/mongo"
)

const (
	mongoUserRepoType = iota
	sqliteUserRepoType
	mongoComicRepoType
	sqliteComicRepoType
	postgresUserRepoType
	postgresComicRepoType
)

// newRepo creates a new UserRepo instance
func newRepo(ctx context.Context, env *Env, repoType int) (ClosableUserStore, error) {
	switch repoType {
	case mongoUserRepoType:
		return mongo.NewUserRepo(ctx, env.DBConfig, env.TracerConfig)
	// case sqliteUserRepoType:
	// 	return sqlite.NewUserRepo(ctx, env.DBConfig, env.TracerConfig)
	// case mongoComicRepoType:
	// 	return mongo.NewComicRepo(ctx, env.DBConfig, env.TracerConfig)
	// case sqliteComicRepoType:
	// 	return sqlite.NewComicRepo(ctx, env.DBConfig, env.TracerConfig)
	// case postgresUserRepoType:
	// 	return postgres.NewUserRepo(ctx, env.DBConfig, env.TracerConfig)
	// case postgresComicRepoType:
	// 	return postgres.NewComicRepo(ctx, env.DBConfig, env.TracerConfig)
	default:
		return nil, errors.New("unknown repo type")
	}
}
