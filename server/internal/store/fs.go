package store

import (
	"io/fs"
)

func migrationsFS() fs.FS {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		panic(err) // the embed path is fixed at compile time
	}
	return sub
}
