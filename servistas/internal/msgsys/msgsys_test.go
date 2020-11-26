package msgsys

import (
	"fmt"
	"testing"
	"time"
)

func TestNetworkMsgSys(t *testing.T) {
	server := HostPuerto("127.0.0.1:20000")

	ms := MakeMsgSys(server)

	defer ms.CloseMessageSystem()

	time.Sleep(10 * time.Microsecond) //Esperar un poco al servidor tcp

	ms.Send(server, 5)
	ms.Send(server, "texto")

	datoEsperado1 := 5
	datoObtenido1 := ms.Receive()
	datoEsperado2 := "texto"
	datoObtenido2 := ms.Receive()

	fmt.Printf("Dato obtenido1: %#v\n", datoObtenido1)
	fmt.Printf("Dato obtenido2: %#v\n", datoObtenido2)

	switch {
	case datoObtenido1 != datoEsperado1:
		t.Errorf("<- Buzon de red == %#v; esperaba MsgPeticionPrimario{}",
			datoObtenido1)

	case datoObtenido2 != datoEsperado2:
		t.Errorf("<- Buzon de red == %#v; esperaba MsgPeticionPrimario{}",
			datoObtenido2)
	}
}

func TestCorrectSendReceive(t *testing.T) {
	server := HostPuerto("127.0.0.1:20000")
	Registrar([]Message{HostPuerto("")})
	ms := MakeMsgSys(server)

	defer ms.CloseMessageSystem()

	go func() {
		m := ms.Receive()
		//fmt.Printf("Received: %#v\n", m)
		ms.Send(m.(HostPuerto), 27)
	}()

	time.Sleep(50 * time.Millisecond)

	m, _ := ms.SendReceive(server, server, 10*time.Millisecond)
	fmt.Printf("Received: %#v\n", m)

	if m != 27 {
		t.Errorf("SendReceive == %#v; esperaba 27}",
			m)
	}
}

func TestErrorSendReceiveWithTimeout(t *testing.T) {
	ms := MakeMsgSys("127.0.0.1:20000")

	go func() {
		_ = ms.Receive()
	}()

	time.Sleep(time.Millisecond)

	_, timeout := ms.SendReceive("127.0.0.1:20000",
		"127.0.0.1:20000", 10*time.Millisecond)

	if timeout {
		t.Error("SendReceive timeout =  false !!!")
	}
}
