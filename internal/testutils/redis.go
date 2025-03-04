package testutils

import (
	"fmt"
	"sync"

	"github.com/alicebob/miniredis"
)

var (
	miniRdb     *miniredis.Miniredis
	miniRdbOnce sync.Once
)

func PrepareRedis() (dsn string, shutdown func()) {
	miniRdbOnce.Do(func() {
		miniRdb = miniredis.NewMiniRedis()
		if err := miniRdb.Start(); err != nil {
			panic(err)
		}
	})

	dsn = fmt.Sprintf("127.0.0.1:%s", miniRdb.Port())
	shutdown = miniRdb.Close
	return
}
