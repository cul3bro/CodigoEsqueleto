/*
Ejecutable testcltvts solo para pruebas, únicamente en esta prática,
   de un cliente mínimio del servico de vistas
*/
package main

import (
	"fmt"
	"log"
	"os"
	"servistas/internal/gvcomun"
	"servistas/internal/msgsys"
	"servistas/pkg/cltvts"
)

func main() {
	// obtener host:puerto de este clientes desde argumentos de llamada
	me := os.Args[1]
	// obtener host:puerto del gestor de vistas desde argumentos de llamada
	gv := os.Args[2]
	// obtener host:puerto del nodo de tests desde argumentos de llamada
	nodotest := os.Args[3]

	// Start a test client and get its mailbox channel
	StartClienteVistas(msgsys.HostPuerto(me),
		msgsys.HostPuerto(gv), msgsys.HostPuerto(nodotest))
}

type clienteTest struct {
	cltvts.CltVts // tipo abstracto de cliente de gestor de vistas
	msgsys.MsgSys
	nodotest msgsys.HostPuerto
}

func StartClienteVistas(me, gv, nodotest msgsys.HostPuerto) {

	log.Println("Puesta en marcha de ClienteVistas")

	// Registrar tipos de mensaje de gestión d vistas
	gvcomun.RegistrarTiposMensajesGV()

	// Crear el cliente de test
	clt := clienteTest{
		CltVts:   cltvts.CltVts{Gv: gv},
		MsgSys:   msgsys.MakeMsgSys(me),
		nodotest: nodotest,
	}

	// Tratar mensajes de clientes para reencaminar a Servidor de Vistas
	// Termina ejecución cuando recibe mensaje tipo MsgFin
	clt.ProcessAllMsg(clt.procesaMensaje)
}

func (cl *clienteTest) stop() {
	cl.CloseMessageSystem()
}

func (cl *clienteTest) procesaMensaje(m msgsys.Message) {
	//fmt.Printf("----Recibido mensaje : %#v\n", m)

	switch x := m.(type) {
	case gvcomun.MsgLatido:
		//fmt.Println("Recibido latido %v", x)
		cl.Latido(*cl, x.NumVista)

	case gvcomun.MsgVistaTentativa:
		cl.Send(cl.nodotest, gvcomun.MsgVistaTentativa{x.Vista})

	case gvcomun.MsgPeticionVistaValida:
		//fmt.Println("Recibido peticion vista")
		cl.PeticionVistaValida(cl)

	case gvcomun.MsgVistaValida:
		cl.Send(cl.nodotest, gvcomun.MsgVistaValida{x.Vista})

	case gvcomun.MsgPeticionPrimario:
		//fmt.Println("Recibido peticion PRIMARIO")
		cl.PeticionPrimario(*cl)

	case gvcomun.MsgPrimario:
		cl.Send(cl.nodotest, x)

	case gvcomun.MsgFin:
		//log.Println("Recibido FIN")
		cl.stop()
		os.Exit(0) // Aquí termina  la ejecución del servidor

	default:
		fmt.Printf(
			"Recibido mensaje TIPO DESCONOCIDO en servidor_vistas.go!! %#v\n", m)
	}
}
