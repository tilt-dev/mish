package tracing

import (
	"context"
	"database/sql"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func Query(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	span, ctx := StartSystemSpanFromContext(ctx, "db/sql", opentracing.Tags{
		string(ext.Component): "db",
		query: query,
	})
	defer span.Finish()
	return db.QueryContext(ctx, query, args...)
}

func Exec(ctx context.Context, db *sql.DB, query string, args ...interface{}) (sql.Result, error) {
	span, ctx := StartSystemSpanFromContext(ctx, "db/sql", opentracing.Tags{
		string(ext.Component): "db",
		query: query,
	})
	defer span.Finish()
	return db.ExecContext(ctx, query, args...)
}
