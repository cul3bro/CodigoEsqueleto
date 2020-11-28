/*
   paquete cltvts donde se implementa el código de comunicación
   de clientes con el servidor de vistas
*/

package cltvts

import (
	"log"
	"os"
	"servistas/internal/gvcomun"
	"servistas/internal/msgsys"
	"time"
)

type Remitente interface {
	Send(msgsys.HostPuerto, msgsys.Message)
	Me() msgsys.HostPuerto
}

type CltVts struct {
	msgsys.MsgSys
	doneTicker chan struct{}

	Gv    msgsys.HostPuerto // dirección completa de red de GV
	Vista gvcomun.Vista
}

func Make(me msgsys.HostPuerto, gv msgsys.HostPuerto) CltVts {
	gvcomun.ChangeLogPrefix()

	gvcomun.RegistrarTiposMensajesGV()

	return CltVts{
		MsgSys:     msgsys.MakeMsgSys(me),
		doneTicker: make(chan struct{}),
		Gv:         gv,
		Vista: gvcomun.Vista{
			NumVista: 0,
			Primario: "",
			Copia:    "",
		},
	}
}

func (cv *CltVts) Start() {
	go cv.ticker()

	cv.ProcessAllMsg(cv.procesaMensaje)
}

func (cv *CltVts) Stop() {
	close(cv.doneTicker)

	time.Sleep(2 * gvcomun.INTERVALOLATIDOS * time.Millisecond)

	cv.CloseMessageSystem()
}

func (cv *CltVts) ticker() {
iteracion:
	for {
		select {
		case <-cv.doneTicker:
			break iteracion

		default:
			time.Sleep(gvcomun.INTERVALOLATIDOS * time.Millisecond)
			cv.enviarLatido()
		}
	}
}

func (cv *CltVts) procesaMensaje(m msgsys.Message) {
	switch x := m.(type) {
	case gvcomun.MsgPeticionVistaValida:
		// falta la funcion de devolver la vista valida
		log.Println("Recibido peticion vista valida")
	case gvcomun.MsgVistaTentativa:
		cv.procesarVistaTentativa(x)
	case gvcomun.MsgPrimario:
		cv.simularCopiaEstado(x)
	case gvcomun.MsgFin:
		log.Println("Recibido FIN")
		cv.Stop() // Eliminar el servidor de vistas !!
		os.Exit(0)
	default:
		log.Printf(
			"Llega mensaje TIPO DESCONOCIDO en cliente_vistas.go!! %#v\n", m)
	}
}

func (cv *CltVts) enviarLatido() {
	cv.Send(cv.Gv, gvcomun.MsgLatido{cv.Vista.NumVista,
		cv.Me()})
}

func (cv *CltVts) PeticionPrimario(rt Remitente) {
	//log.Printf("PeticionPrimarioen cltvts: GV = %#v, CLT = %#v\n ",
	//	servVistas, me)

	rt.Send(cv.Gv, gvcomun.MsgPeticionPrimario{rt.Me()})
}

// solo para depuración
func (cv *CltVts) PeticionVistaValida(rt Remitente) {

	rt.Send(cv.Gv, gvcomun.MsgPeticionVistaValida{rt.Me()})
}

func (cv *CltVts) procesarVistaTentativa(x gvcomun.MsgVistaTentativa) {
	if x.Vista.NumVista == 1 { //situacion especial (primera vista)
		cv.Vista.NumVista = -1
	} else if x.Vista.NumVista != cv.Vista.NumVista {
		if x.Vista.Primario == cv.Me() &&
			x.Vista.Copia != "" { //Si soy primario y hay copia
			cv.copiarEstado(x)
		} else {
			cv.Vista = x.Vista
		}
	}
}

func (cv *CltVts) copiarEstado(x gvcomun.MsgVistaTentativa) {
	msg, ok := cv.SendReceive(x.Vista.Copia, gvcomun.MsgPrimario(cv.Me()),
		gvcomun.ANSWERWAITTIME)
	if ok {
		res, isType := msg.(gvcomun.MsgPrimario)
		if (isType) && msgsys.HostPuerto(res) == x.Vista.Copia {
			cv.Vista = x.Vista
		}
	}
}

func (cv *CltVts) simularCopiaEstado(x gvcomun.MsgPrimario) {
	//time.Sleep(gvcomun.ANSWERWAITTIME * time.Millisecond)

	cv.Send(msgsys.HostPuerto(x), gvcomun.MsgPrimario(cv.Me()))
}
