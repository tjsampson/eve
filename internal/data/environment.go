package data

import (
	"context"

	"gitlab.unanet.io/devops/eve/pkg/errors"
	"gitlab.unanet.io/devops/eve/pkg/json"
)

type Environment struct {
	ID          int       `db:"id"`
	Name        string    `db:"name"`
	Alias       string    `db:"alias"`
	Description string    `db:"description"`
	Metadata    json.Text `db:"metadata"`
}

type Environments []Environment

func (r *Repo) EnvironmentByName(ctx context.Context, name string) (*Environment, error) {
	var environment Environment

	row := r.db.QueryRowxContext(ctx, "select * from environment where name = $1", name)
	err := row.StructScan(&environment)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, NotFoundErrorf("environment with name: %s, not found", name)
		}
		return nil, errors.Wrap(err)
	}

	return &environment, nil
}

func (r *Repo) EnvironmentByID(ctx context.Context, id int) (*Environment, error) {
	var environment Environment

	row := r.db.QueryRowxContext(ctx, "select * from environment where id = $1", id)
	err := row.StructScan(&environment)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, NotFoundErrorf("environment with id: %d, not found", id)
		}
		return nil, errors.Wrap(err)
	}

	return &environment, nil
}

func (r *Repo) Environments(ctx context.Context) (Environments, error) {
	rows, err := r.db.QueryxContext(ctx, "select * from environment")
	if err != nil {
		return nil, errors.Wrap(err)
	}
	var environments []Environment
	for rows.Next() {
		var environment Environment
		err = rows.StructScan(&environment)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		environments = append(environments, environment)
	}

	return environments, nil
}
