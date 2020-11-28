/*
   paquete cltvts donde se implementa el código de comunicación
   de clientes con el servidor de vistas
*/

package cltvts

import (
	"servistas/internal/gvcomun"
	"servistas/internal/msgsys"
)

type Remitente interface {
	Send(msgsys.HostPuerto, msgsys.Message)
	Me() msgsys.HostPuerto
}

type CltVts struct {
	Gv msgsys.HostPuerto // dirección completa de red de GV
}

func Make(me msgsys.HostPuerto, gv msgsys.HostPuerto) CltVts {
	gvcomun.ChangeLogPrefix()

	gvcomun.RegistrarTiposMensajesGV()

	return CltVts{}
}

func (cv CltVts) Latido(rt Remitente, numVista int) {

	rt.Send(cv.Gv, gvcomun.MsgLatido{numVista, rt.Me()})
}

func (cv CltVts) PeticionPrimario(rt Remitente) {
	//log.Printf("PeticionPrimarioen cltvts: GV = %#v, CLT = %#v\n ",
	//	servVistas, me)

	rt.Send(cv.Gv, gvcomun.MsgPeticionPrimario{rt.Me()})
}

// solo para depuración
func (cv CltVts) PeticionVistaValida(rt Remitente) {

	rt.Send(cv.Gv, gvcomun.MsgPeticionVistaValida{rt.Me()})
}
