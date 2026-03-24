// Package tu is the test utility
package tu

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"

	"github.com/dnjooiopa/phone-charging-locker/pkg/dbctx"
	"github.com/dnjooiopa/phone-charging-locker/schema"
)

// Context is the test context contains test's dependencies
type Context struct {
	tmpDir       string
	cleanupHooks []func()

	DB *sql.DB
}

func (ctx *Context) setup() {
	var err error

	defer func() {
		if err != nil {
			panic(err)
		}
	}()

	ctx.tmpDir, err = os.MkdirTemp("", "pcl-tu-*")
	if err != nil {
		panic(err)
	}

	dbPath := filepath.Join(ctx.tmpDir, "test.db")
	ctx.DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}

	_, err = ctx.DB.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		panic(err)
	}
	_, err = ctx.DB.Exec("PRAGMA foreign_keys=ON")
	if err != nil {
		panic(err)
	}

	// prepare schema
	err = schema.Migrate(context.Background(), ctx.DB)
	if err != nil {
		panic(err)
	}
}

func (ctx *Context) Teardown() {
	for _, f := range ctx.cleanupHooks {
		f()
	}

	if ctx.DB != nil {
		ctx.DB.Close()
	}

	if ctx.tmpDir != "" {
		os.RemoveAll(ctx.tmpDir)
	}
}

func (ctx *Context) Ctx() context.Context {
	c := context.Background()
	c = dbctx.NewContext(c, ctx.DB)
	return c
}

// Setup setups test dependencies
func Setup() *Context {
	ctx := &Context{}
	ctx.setup()
	return ctx
}

var (
	inTest     bool
	inTestOnce sync.Once
)

func InTest() bool {
	inTestOnce.Do(func() {
		inTest = flag.Lookup("test.v") != nil
	})
	return inTest
}
