/*
   Package srvvts implementa servicio de vistas para gestión de réplicas
*/
package srvvts

import (
	"log"
	"os"
	"servistas/internal/gvcomun"
	"servistas/internal/msgsys"
	"time"
	// paquetes adicionales si necesario
)

type Retrasos int

type ServVistas struct {
	msgsys.MsgSys
	doneTicker chan struct{}

	// COMPLETAR CON ESTRUCTURA DATOS de ESTADO de GESTOR DE VISTA ????????
	vistaValida    gvcomun.Vista
	vistaTentativa gvcomun.Vista
	servidores     map[msgsys.HostPuerto]Retrasos
}

func Make(me msgsys.HostPuerto) ServVistas {
	// poner microsegundos y PATH completo de fichero codigo en logs
	gvcomun.ChangeLogPrefix()

	//log.Println("Puesta en marcha de servidor_gv")

	// Registrar tipos de mensaje de gestión d vistas
	gvcomun.RegistrarTiposMensajesGV()

	// crear estructura datos estado del servidor
	return ServVistas{
		// Crear channel msgsys
		MsgSys:     msgsys.MakeMsgSys(me),
		doneTicker: make(chan struct{}),
		servidores: make(map[msgsys.HostPuerto]Retrasos),
		vistaValida: gvcomun.Vista{
			NumVista: 0,
			Primario: "",
			Copia:    "",
		},
		vistaTentativa: gvcomun.Vista{
			NumVista: 0,
			Primario: "",
			Copia:    "",
		},
		//CAMPOS ADICIONALES DEL LITERAL DE ESTRUCTURA DATOS DEL GV ???????
	}
}

// Poner en marcha el servidor de vistas/gestor de vistas
func (sv *ServVistas) Start() {
	// crea generador de eventos ticker en periodos de envio de latido
	go sv.ticker()

	// Tratar mensajes de clientes y ticks periodicos de procesado de situacion
	// Termina ejecución cuando recibe mensaje tipo MsgFin.
	sv.ProcessAllMsg(sv.procesaMensaje) // Este metodo esta en paquete "msgsys"
}

func (sv *ServVistas) Stop() {
	// primero cerrar terminación (done) para que ticker
	// pare de enviar MsgTickInterno a buzon
	close(sv.doneTicker)

	// Esperar que goroutina ticker llegue a done por el time.Sleep
	time.Sleep(2 * gvcomun.INTERVALOLATIDOS * time.Millisecond)

	//
	sv.CloseMessageSystem()
}

// ticker to run through all the execution
func (sv *ServVistas) ticker() {
iteracion:
	for {
		select {
		case <-sv.doneTicker:
			//log.Println("Done with receiving Messages from TICKER!!!")
			break iteracion // Deja ya de generar  Ticks !!!

		default:
			time.Sleep(gvcomun.INTERVALOLATIDOS * time.Millisecond)
			sv.InternalSend(gvcomun.MsgTickInterno{})
		}
	}
}

func (sv *ServVistas) procesaMensaje(m msgsys.Message) {
	//log.Printf("----Recibido mensaje : %#v\n", m)

	switch x := m.(type) {
	case gvcomun.MsgLatido:
		sv.trataLatido(x)
	case gvcomun.MsgPeticionVistaValida:
		// falta la funcion de devolver la vista valida
		log.Println("Recibido peticion vista valida")
	case gvcomun.MsgPeticionPrimario:
		sv.trataPeticionPrimario(x.Remitente)
	case gvcomun.MsgTickInterno:
		sv.procesaSituacionReplicas()
	case gvcomun.MsgFin:
		log.Println("Recibido FIN")
		sv.Stop() // Eliminar el servidor de vistas !!
		os.Exit(0)
	default:
		log.Printf(
			"Llega mensaje TIPO DESCONOCIDO en servidor_vistas.go!! %#v\n", m)
	}
}

/// RESTO DE FUNCIONES DEL GESTOR DE VISTAS A IMPLEMENTAR ?????????


func (sv *ServVistas) trataLatido(x gvcomun.MsgLatido) {
	switch x.NumVista {
	case -1:
		// EL NODO SIGUE VIVO
	case 0:
		//Puede haber habido perdida de datos
		// Estado inical inconsistente
		if sv.vistaValida.NumVista == 0 {
			sv.inicializarVista(x)
		} else {
			// Cambiar porsiacaso
			sv.añadirServidorEspera(x)
		}

	case sv.vistaTentativa.NumVista:

		sv.confirmarVista(x)
	}

	sv.servidores[x.Remitente] = 0
	sv.MsgSys.Send(x.Remitente, gvcomun.MsgVistaTentativa{Vista: sv.vistaTentativa})

}

func (sv *ServVistas) inicializarVista(x gvcomun.MsgLatido) {
	if sv.vistaTentativa.Primario == "" {
		sv.vistaTentativa.Primario = x.Remitente
	} else if sv.vistaTentativa.Copia == "" {
		sv.vistaTentativa.Copia = x.Remitente
	}
	sv.vistaTentativa.NumVista += 1
}

func (sv *ServVistas) trataPeticionPrimario(x msgsys.HostPuerto)  {
	//return primario valido
}

// Si no recibe ningún Latido de alguno de los servidores c/v durante
// 	un nº @latidos_fallidos de @intervalo_latidos, los considera caídos.
func (sv *ServVistas) procesaSituacionReplicas() {

	for servidor, retrasos := range sv.servidores{
		retrasos += 1
		// Ha fallado 4 latidos
		if retrasos == gvcomun.LATIDOSFALLIDOS {
			sv.procesarServidorCaido(servidor)
		}
	}
}

func (sv *ServVistas) procesarServidorCaido(servidor msgsys.HostPuerto) {
	delete(sv.servidores, servidor)
	switch servidor {
	case sv.vistaTentativa.Primario:
		// Ha fallado el primario
		if sv.esConsistente() {
			sv.promocionarCopia()
		} else {
			log.Fatalf("Fallo de consistencia %v", sv.vistaTentativa)
		}
	case sv.vistaTentativa.Copia:
		// Ha fallado la copia
		sv.nuevaVistaCopia()
	}
}

//El gestor de vistas crea una nueva vista si :
//	En la inicialización, cuando se incorporan los primeros servidores como primario y
//		copia.
//	No ha recibido un latido del primario o de la copia durante un no @latidos_fallidos
//		de @intervalo_latidos.
//	Ha caido el primario o la copia, y se han perdido sus datos. Quizás, ha rearrancado
//		a continuación, sin que el gestor de vistas haya detectado su fallo.
//	Solo está vivo el primario y aparece un nuevo nodo.
func (sv *ServVistas) nuevaVistaCopia() {
	nuevaCopia := sv.obtenerServidorEspera()
	sv.vistaTentativa = gvcomun.Vista{
		NumVista: sv.vistaTentativa.NumVista + 1,
		Primario: sv.vistaTentativa.Primario,
		Copia: nuevaCopia,
	}
}

func (sv *ServVistas) promocionarCopia() {
	nuevaCopia := sv.obtenerServidorEspera()

	sv.vistaTentativa = gvcomun.Vista{
		NumVista: sv.vistaTentativa.NumVista + 1,
		Primario: sv.vistaTentativa.Copia,
		Copia: nuevaCopia ,
	}
}

func (sv *ServVistas) obtenerServidorEspera() (msgsys.HostPuerto) {
	if len(sv.servidores) > 2 {
		for servidor, _ := range sv.servidores {
			if servidor != sv.vistaTentativa.Primario {
				return servidor
			}
		}
	}
	return ""
}

func (sv *ServVistas) confirmarVista(x gvcomun.MsgLatido) {
	if x.Remitente == sv.vistaTentativa.Primario {
		sv.vistaValida = sv.vistaTentativa
	}
}

func (sv *ServVistas) añadirServidorEspera(x gvcomun.MsgLatido) {
	sv.servidores[x.Remitente] = 0
}

func (sv *ServVistas) esConsistente() bool {
	return sv.vistaTentativa.NumVista == sv.vistaValida.NumVista
}