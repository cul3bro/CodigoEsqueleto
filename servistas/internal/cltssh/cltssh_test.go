package cltssh

import (
	"fmt"
	"servistas/internal/gvcomun"
	"servistas/internal/msgsys"
	"testing"
	"time"
)

func TestBasico(t *testing.T) {
	cmd :=
		"cd /home/tmp/servistas/v2/cmd/; go run cmdsrvvts/main.go 127.0.0.1:29001"
	r := make(chan string, 1000)
	ExecMutipleHosts(cmd, []string{"127.0.0.1"}, r, "/home/unai/.ssh/id_ed25519")

	time.Sleep(1000 * time.Millisecond)

	msgsys.Registrar([]msgsys.Message{gvcomun.MsgFin{}})
	//buzonReadTests, doneReceivers :=
	ms := msgsys.MakeMsgSys("127.0.0.1:29000")
	ms.Send("127.0.0.1:29001", gvcomun.MsgFin{})

	// esperar parada se servidor remoto el tiempo suficiente
	// para volcar salida de ejecuciones ssh en cmdOutput
	time.Sleep(100 * time.Millisecond)
	ms.CloseMessageSystem()
	close(r) //para que termine el bucle for siguiente, en lugar de bloquear

	for s := range r {
		fmt.Println(s)
	}

}
