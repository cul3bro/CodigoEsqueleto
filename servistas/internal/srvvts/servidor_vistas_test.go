package srvvts

import (
	"testing"
	"time"
)

func TestStartServidorVistas(t *testing.T) {

	sv := Make("127.0.0.1:20000")

	t.Run("Subtest Arraque GV",
		func(t *testing.T) { t.Parallel(); sv.Start() })

	t.Run("Subtest Parada GV", func(t *testing.T) {
		t.Parallel()
		time.Sleep(10 * time.Millisecond) //esperar a que e ponga en marcha GV
		sv.Stop()
	})
}
