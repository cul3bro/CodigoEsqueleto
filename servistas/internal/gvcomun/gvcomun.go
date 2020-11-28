/*
   paquete "comun" donde reside el código común compartido por
   los diferentes paquetes de este modulo
*/
package gvcomun

import (
	"log"
	"servistas/internal/msgsys"
)

type HostPuerto string // "nombredns:puerto" o "numIP:puerto"

const (
	LATIDOSFALLIDOS  = 4  // nº látidos fallidos para decidir fallo definitivo
	INTERVALOLATIDOS = 50 // en milisegundos
	ANSWERWAITTIME   = 35 // en milisegundos
)

type Vista struct {
	NumVista int
	Primario msgsys.HostPuerto
	Copia    msgsys.HostPuerto
}

// Tipos mensajes concretos para cliente y servidor:
//      Latido, VistaTentativa, VistaValida, DameVistaValida
// Mensajes de cliente
type MsgLatido struct {
	NumVista  int
	Remitente msgsys.HostPuerto
}

type MsgPeticionVistaValida struct {
	Remitente msgsys.HostPuerto
}

type MsgPeticionPrimario struct {
	Remitente msgsys.HostPuerto
}

type MsgFin struct{}

//Mensajes de Servidor
type MsgVistaTentativa struct {
	Vista Vista
}

type MsgVistaValida struct {
	Vista Vista
}

type MsgPrimario msgsys.HostPuerto

// Mensajes comunes
type MsgTickInterno struct{}

func RegistrarTiposMensajesGV() {
	// Registrar tipos mensajes de todos los tipos de servidores
	// para decodificación con Encode y Decode de red
	tiposMensaje := []msgsys.Message{
		MsgLatido{},              // lo que mandan los servidores c/v
		MsgVistaTentativa{},      //lo que devuelve el servidor en un latido
		MsgPeticionPrimario{},    //cliente de almacenamiento pide primario a GV
		MsgPrimario(""),          //GV devuelve primario valido
		MsgPeticionVistaValida{}, //depuracion
		MsgVistaValida{},
		MsgFin{}} //fin
	msgsys.Registrar(tiposMensaje)
}

// Funciones genericas

func CheckError(err error, comment string) {
	if err != nil {
		ChangeLogPrefix()
		log.Fatalf("Fatal error --- %s -- %s\n", err.Error(), comment)
	}
}

func ChangeLogPrefix() {
	// poner microsegundos y PATH completo de fichero codigo en logs
	log.SetFlags(log.Lmicroseconds | log.Llongfile)
}
