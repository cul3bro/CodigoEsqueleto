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
	log.Printf("GV----Recibido mensaje : %#v\n", m)

	switch x := m.(type) {
	case gvcomun.MsgLatido:
		sv.trataLatido(x)
	case gvcomun.MsgPeticionVistaValida:
		sv.tratarPetitionVistaValida(x)
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
	f, err := os.OpenFile("text.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	logger := log.New(f, "prefix", log.LstdFlags)
	switch x.NumVista {
	case -1:
		// EL NODO SIGUE VIVO
	case 0:
		//Puede haber habido perdida de datos
		// Estado inical inconsistente
		logger.Printf("Latido 0 de %v", x.Remitente)
		if sv.vistaValida.NumVista == 0 {
			sv.inicializarVista(x)
		} else {
			if _, esta := sv.servidores[x.Remitente]; esta {
				logger.Printf("Latido 0 de %v y estaba online", x.Remitente)

				sv.procesarServidorCaido(x.Remitente, esta)
			}
			sv.añadirServidorEspera(x)
		}

	case sv.vistaTentativa.NumVista:
		if !sv.esConsistente() {
			sv.confirmarVista(x)
		}
	}
	sv.comprobarEstadoVista()
	sv.servidores[x.Remitente] = 0
	sv.Send(x.Remitente, gvcomun.MsgVistaTentativa{Vista: sv.vistaTentativa})

}

func (sv *ServVistas) inicializarVista(x gvcomun.MsgLatido) {
	if sv.vistaTentativa.Primario == "" {
		sv.vistaTentativa.Primario = x.Remitente
	} else if sv.vistaTentativa.Copia == "" {
		sv.vistaTentativa.Copia = x.Remitente
	}
	sv.vistaTentativa.NumVista += 1
}

func (sv *ServVistas) trataPeticionPrimario(x msgsys.HostPuerto) {
	sv.Send(x, gvcomun.MsgPrimario(sv.vistaValida.Primario))
}

func (sv *ServVistas) tratarPetitionVistaValida(x gvcomun.MsgPeticionVistaValida) {
	sv.Send(x.Remitente, gvcomun.MsgVistaValida{sv.vistaValida})
}

// Si no recibe ningún Latido de alguno de los servidores c/v durante
// 	un nº @latidos_fallidos de @intervalo_latidos, los considera caídos.
func (sv *ServVistas) procesaSituacionReplicas() {

	f, err := os.OpenFile("text.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	logger := log.New(f, "prefix", log.LstdFlags)
	for servidor, retrasos := range sv.servidores {
		sv.servidores[servidor] = retrasos + 1
		logger.Printf("Nodo Retraso -> %s, %d", servidor, sv.servidores[servidor])
		// Ha fallado 4 latidos
		if retrasos == gvcomun.LATIDOSFALLIDOS {
			logger.Printf("Nodo Caido -> %s", servidor)
			sv.procesarServidorCaido(servidor, false)
		}
	}
	sv.comprobarEstadoVista()
	logger.Printf("Situacion tras procesar nodos \n%v", sv.vistaTentativa)
}

func (sv *ServVistas) comprobarEstadoVista() {
	f, err := os.OpenFile("text.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	logger := log.New(f, "prefix", log.LstdFlags)
	if sv.vistaTentativa.Primario == msgsys.HOSTINDEFINIDO {
		logger.Printf("Primario Indefinido")
		//sv.promocionarCopia()
	}
	if sv.vistaTentativa.Copia == msgsys.HOSTINDEFINIDO {
		logger.Printf("Copia indefinida")
		sv.nuevaVistaCopia()
	}
}

func (sv *ServVistas) procesarServidorCaido(
	servidor msgsys.HostPuerto, rearrancado bool) {
	f, err := os.OpenFile("text.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	logger := log.New(f, "prefix", log.LstdFlags)

	if !rearrancado {
		delete(sv.servidores, servidor)
	}
	switch servidor {
	case sv.vistaTentativa.Primario:
		logger.Printf("Fallo del primario")
		// Ha fallado el primario
		if sv.esConsistente() {
			logger.Printf("Es consistente")
			sv.promocionarCopia()
		} else {
			logger.Printf("Es no consistente")
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
	if nuevaCopia != msgsys.HOSTINDEFINIDO {
		sv.vistaTentativa = gvcomun.Vista{
			NumVista: sv.vistaTentativa.NumVista + 1,
			Primario: sv.vistaTentativa.Primario,
			Copia:    nuevaCopia,
		}
	}
}

func (sv *ServVistas) promocionarCopia() {

	nuevaCopia := sv.obtenerServidorEspera()
	sv.vistaTentativa = gvcomun.Vista{
		NumVista: sv.vistaTentativa.NumVista + 1,
		Primario: sv.vistaTentativa.Copia,
		Copia:    nuevaCopia,
	}
}

func (sv *ServVistas) obtenerServidorEspera() msgsys.HostPuerto {
	if len(sv.servidores) > 1 {
		for servidor, _ := range sv.servidores {
			if servidor != sv.vistaTentativa.Primario &&
				servidor != sv.vistaTentativa.Copia {
				return servidor
			}
		}
	}
	return msgsys.HOSTINDEFINIDO
}

func (sv *ServVistas) confirmarVista(x gvcomun.MsgLatido) {
	f, err := os.OpenFile("text.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	logger := log.New(f, "prefix", log.LstdFlags)
	if x.Remitente == sv.vistaTentativa.Primario {
		sv.vistaValida = sv.vistaTentativa
		logger.Printf("Vista confirmada %v\n", sv.vistaValida)
	}
}

func (sv *ServVistas) añadirServidorEspera(x gvcomun.MsgLatido) {
	f, err := os.OpenFile("text.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	logger := log.New(f, "prefix", log.LstdFlags)

	sv.servidores[x.Remitente] = 0
	logger.Printf("Servidor Añadido %v", x)
}

func (sv *ServVistas) esConsistente() bool {
	return sv.vistaTentativa.NumVista == sv.vistaValida.NumVista
}
