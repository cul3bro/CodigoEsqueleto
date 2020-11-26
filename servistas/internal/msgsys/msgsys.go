package msgsys

import (
	"encoding/gob"
	"log"
	"net"
	"time"
)

type HostPuerto string // "nombredns:puerto" o "numIP:puerto"

type Message interface{} //tipo generico, incluye todos los tipos existentes

const (
	// Dato no conocido
	HOSTINDEFINIDO = "HOST INDEFINIDO"
	// Máximo nº de mensajes en mailbox
	MAXMESSAGES = 10000
)

type MsgSys struct {
	me       HostPuerto
	listener net.Listener
	buzon    chan Message
	done     chan struct{}
	tmr      *time.Timer
}

func checkError(err error, comment string) {
	if err != nil {
		log.SetFlags(log.Lmicroseconds)
		log.Fatalf("Fatal error --- %s -- %s\n", err.Error(), comment)
	}
}

func MakeMsgSys(me HostPuerto) (ms MsgSys) {
	ms = MsgSys{me: me,
		buzon: make(chan Message, MAXMESSAGES),
		done:  make(chan struct{}),
		tmr:   time.NewTimer(0)}
	// detener y consumir, eventualmente el timer inicializado
	if !ms.tmr.Stop() {
		<-ms.tmr.C
	}

	var err error
	ms.listener, err = net.Listen("tcp", string(ms.me))
	checkError(err, "Problema aceptación en networkReceiver  ")

	log.Println("Process listening at ", string(ms.me))

	// concurrent network listener to this MailBoxRead
	go ms.networkReceiver()

	return ms
}

// Registrar una lista de tipos de mensaje
func Registrar(tiposMensaje []Message) {
	for _, t := range tiposMensaje {
		gob.Register(t)
	}
}

func (ms MsgSys) Me() HostPuerto {
	return ms.me
}

// Close message system and all its goroutines
func (ms *MsgSys) CloseMessageSystem() {
	//notificar terminacion a la goroutine de escucha networkReceiver
	close(ms.done)

	err := ms.listener.Close() // Cerrar el Accept de networkReceiver
	checkError(err, "Problema cierre de Listener en CloseMessageSystem")

	close(ms.buzon)

	// Wait a milisecond for goroutine to die
	time.Sleep(time.Millisecond)
}

// network listener to a Channel
func (ms *MsgSys) networkReceiver() {
	for {
		conn, err := ms.listener.Accept() // bloqueo en recepción de red
		if err != nil {                   // en caso de error
			select {
			// puede ser normalpor listener.close() de CloseMessageSystem
			case <-ms.done:
				log.Println("STOPPED listening for messages at", string(ms.me))
				return // Escucha terminada correctamente para MsgSys !!!!

			// o un error de estableceimineto de conexion a notificar
			default:
				checkError(err, "Problema aceptación en New")
			}
		}

		decoder := gob.NewDecoder(conn)
		var msg Message
		err = decoder.Decode(&msg)
		checkError(err, "Problema Decode en mailbox.Make")

		conn.Close()

		ms.buzon <- msg
	}
}

func (ms MsgSys) Send(destinatario HostPuerto, msg Message) {
	conn, err := net.Dial("tcp", string(destinatario))
	checkError(err, "Problema con DialTCP en Send")

	// fmt.Printf("Message for encoder: %#v \n", msg)
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(&msg)
	checkError(err, "Problema con Encode en Send")

	conn.Close()
}

// Send sincrono con respuesta y timeout, true si recibe respuesta, false sino
// timeout en microsegundos
func (ms *MsgSys) SendReceive(destinatario HostPuerto,
	msg Message, timeout time.Duration) (res Message, ok bool) {

	log.SetFlags(log.Lmicroseconds)
	//log.Println("Comienzo SendReceive :", destinatario, msg)

	ms.Send(destinatario, msg)

	ms.tmr.Reset(timeout)

	// recibe mensaje o timeout
	select {
	case res = <-ms.buzon:
		//fmt.Println("SendReceive before timer.stop")
		if !ms.tmr.Stop() {
			<-ms.tmr.C
		}
		//fmt.Println("SendReceive AFTER timer.stop")
		return res, true

	case <-ms.tmr.C:
		//log.Println("SendReceive timeout! ", destinatario, msg)

		return struct{}{}, false
	}
}

func (ms *MsgSys) InternalSend(msg Message) {
	ms.buzon <- msg
}

// Recepción bloqueante
func (ms *MsgSys) Receive() Message {
	return <-ms.buzon
}

// Recepción con timeout, true si recibe mensaje, false sino
func (ms *MsgSys) ReceiveTimed(timeout time.Duration) (m Message, ok bool) {
	ms.tmr.Reset(timeout)

	select {
	case m = <-ms.buzon:
		if !ms.tmr.Stop() {
			<-ms.tmr.C
		}
		return m, true

	case <-ms.tmr.C:
		return struct{}{}, false
	}
}

func (ms *MsgSys) ProcessAllMsg(procesaMensaje func(m Message)) {
	for m := range ms.buzon {
		procesaMensaje(m)
	}
}
