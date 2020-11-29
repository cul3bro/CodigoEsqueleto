package integracion_test

import (
	"encoding/gob"
	"fmt"
	"net"
	"servistas/internal/cltssh"
	"servistas/internal/gvcomun"
	"servistas/internal/msgsys"
	"testing"
	"time"
)

const (
	//hosts
	MAQUINA1 = "127.0.0.1"
	MAQUINA2 = "127.0.0.1"
	MAQUINA3 = "127.0.0.1"
	MAQUINA4 = "127.0.0.1"

	//puertos
	PUERTO0 = "29000"
	PUERTO1 = "29001"
	PUERTO2 = "29002"
	PUERTO3 = "29003"
	PUERTO4 = "29004"

	//nodos
	NODOTEST     = MAQUINA1 + ":" + PUERTO0
	NODOGV       = MAQUINA2 + ":" + PUERTO1
	NODOCLIENTE1 = MAQUINA3 + ":" + PUERTO2
	NODOCLIENTE2 = MAQUINA4 + ":" + PUERTO3
	NODOCLIENTE3 = MAQUINA1 + ":" + PUERTO4

	// PATH de los (plural) ejecutables de modulo golang de servicio de vistas
	PATH = "/home/fuster/Descargas/Practica4/CodigoEsqueleto/servistas/cmd/"
	// GO = /usr/local/go/bin/go
	// fichero main de ejecutables relativos a PATH previo
	EXECGV      = "cmdsrvvts/main.go " + NODOGV // un parámetro
	EXECTESTCLT = "testcltvts/main.go "         // 2 parámetros en llamadas ssh

	// comandos completo a ejecutar en máquinas remota con ssh
	// cd /home/tmp/servistas/v2/cmd/; go run cmdsrvvts/main.go 127.0.0.1:29001
	SRVVTSCMD = "cd " + PATH + "; go run " + EXECGV
	// cd /home/tmp/servistas/v2/cmd/;
	// go run testcltvts/main.go 127.0.0.1:29003 127.0.0.1:29001 127.0.0.1:29000
	CLTVTSCMD = "cd " + PATH + "; go run " + EXECTESTCLT

	// Ubicar, en esta constante, el PATH completo a vuestra clave privada local
	// emparejada con la clave pública en authorized_keys de máquinas remotas

	PRIVKEYFILE = "/home/fuster/.ssh/id_rsa"
)

// TEST primer rango
func TestPrimerasPruebas(t *testing.T) { // (m *testing.M) {
	// <setup code>
	// Crear servidor de test y
	//procesos en maquinas remotas : servidor de vistas
	ts := startTestServer(NODOTEST)
	ts.startDistributedProcesses(t)

	// Run test sequence

	// Test1 : No debería haber ningun primario, si SV no ha recibido aún latidos
	t.Run("P=0:T1",
		func(t *testing.T) { ts.ningunPrimarioTest1(t) })

	// Test2: tenemos el primer primario correcto
	t.Run("P=1:T2",
		func(t *testing.T) { ts.primerPrimarioTest2(t) })

	// Test3: Primer nodo copia
	t.Run("C=1:T3",
		func(t *testing.T) { ts.PrimerCopiaTest3(t) })

	// Test4: Copia toma relevo
	t.Run("CopiaPromociona:T4",
		func(t *testing.T) { ts.CopiaTomaRelevoTest4(t) })

	// Test5: Servidor rearranca y es copia
	t.Run("ServidorRearrancadoCopia:T5",
		func(t *testing.T) { ts.ServidorRearrancadoEsCopiaTest5(t) })

	// Test6: Servidor en espera se convierte en copia
	t.Run("PromocionEsperaCopia:T6",
		func(t *testing.T) { ts.PromocionEsperaCopiaTest6(t) })

	// Test7: Primario manda latido 0 y es puesto en nodos de espera
	t.Run("PrimarioRearrancaEspera:T7",
		func(t *testing.T) {
			ts.PrimarioAEsperaTest7(t)
		})

	// Test8: GV espera a que primario confirme la vista
	t.Run("ServidorEsperaConfirmacion:T8",
		func(t *testing.T) {
			ts.EsperaConfirmacionTest8(t)
		})

	// Test9: Un nuevo servidor sin inicializar no puede ser primario
	t.Run("ServidorSinInicializarNoEsPrimario:T9",
		func(t *testing.T) {
			ts.ServidorSinInicializarNoEsPrimarioTest9(t)
		})

	// tear down code
	// eliminar procesos en máquinas remotas
	ts.stopDistributedProcessesNoErr(t)

	ts.stop()
}

// ---------------------------------------------------------------------
// Servidor de test

type testServer struct {
	msgsys.MsgSys
	// Canal de resultados de ejecución de comadnos ssh remotos
	cmdOutput chan string
}

func startTestServer(me msgsys.HostPuerto) (ts testServer) {
	// Registrar tipos de mensaje de gestión d vistas
	gvcomun.RegistrarTiposMensajesGV()

	ts = testServer{
		MsgSys:    msgsys.MakeMsgSys(me),
		cmdOutput: make(chan string, 1000),
	}

	return ts
}

func (ts *testServer) stop() {
	ts.CloseMessageSystem()
	close(ts.cmdOutput)

	// Visulaizar salidas obtenidos de los ssh ejecutados por servidor de tests
	for s := range ts.cmdOutput {
		fmt.Println(s)
	}
}

func (ts *testServer) startDistributedProcesses(t *testing.T) {
	t.Helper()
	// Poner en marcha servidor/gestor de vistas y 3 clientes
	// en 4 máquinas remota con ssh
	cltssh.ExecMutipleHosts(SRVVTSCMD,
		[]string{MAQUINA2}, ts.cmdOutput, PRIVKEYFILE)
	cltssh.ExecMutipleHosts(CLTVTSCMD+NODOCLIENTE1+" "+NODOGV+" "+NODOTEST,
		[]string{MAQUINA3}, ts.cmdOutput, PRIVKEYFILE)
	cltssh.ExecMutipleHosts(CLTVTSCMD+NODOCLIENTE2+" "+NODOGV+" "+NODOTEST,
		[]string{MAQUINA4}, ts.cmdOutput, PRIVKEYFILE)
	cltssh.ExecMutipleHosts(CLTVTSCMD+NODOCLIENTE3+" "+NODOGV+" "+NODOTEST,
		[]string{MAQUINA1}, ts.cmdOutput, PRIVKEYFILE)

	// ajustar si necesario para esperar al
	// tiempo de establecimiento de sesión de ssh
	time.Sleep(4000 * time.Millisecond)
}

func (ts *testServer) stopDistributedProcesses() {
	// Parar procesos distribuidos con ssh
	// una opción :
	ts.Send(NODOGV, gvcomun.MsgFin{})
	ts.Send(NODOCLIENTE1, gvcomun.MsgFin{})
	ts.Send(NODOCLIENTE2, gvcomun.MsgFin{})
	ts.Send(NODOCLIENTE3, gvcomun.MsgFin{})

	// esperar parada se servidores remotos el tiempo suficiente
	// para volcar salida de ejecuciones ssh en cmdOutput
	time.Sleep(100 * time.Millisecond)
}

func (ts *testServer) stopDistributedProcessesNoErr(t *testing.T) {
	t.Helper()
	// Parar procesos distribuidos con ssh
	// una opción :
	// Se ignoran los posibles errores en el send
	_ = ts.SendConError(NODOGV, gvcomun.MsgFin{})
	_ = ts.SendConError(NODOCLIENTE1, gvcomun.MsgFin{})
	_ = ts.SendConError(NODOCLIENTE2, gvcomun.MsgFin{})
	_ = ts.SendConError(NODOCLIENTE3, gvcomun.MsgFin{})

	// esperar parada se servidores remotos el tiempo suficiente
	// para volcar salida de ejecuciones ssh en cmdOutput
	time.Sleep(100 * time.Millisecond)
}

// --------------------------------------------------------------------------
// FUNCIONES DE SUBTESTS

// No debería haber primario
func (ts *testServer) ningunPrimarioTest1(t *testing.T) {
	fmt.Println(t.Name(), ".....................")

	// obten la respuesta a  la petición de primario
	p, ok := ts.SendReceive(NODOCLIENTE1,
		gvcomun.MsgPeticionPrimario{Remitente: NODOCLIENTE1},
		gvcomun.ANSWERWAITTIME*time.Millisecond)

	if !ok {
		fmt.Printf(
			"TIMEOUT SENDRECEIVE NINGUN PRIMERO TEST: %s", NODOCLIENTE1)
		t.Fatalf(
			"Ha saltado timeout esperando respuesta de Gestor de Vistas %#v",
			t.Name())
	}

	if p != gvcomun.MsgPrimario(msgsys.HOSTINDEFINIDO) {
		t.Fatalf("Primario = %#v; DESEABLE DESCONOCIDO = %#v",
			p, msgsys.HOSTINDEFINIDO)
	}

	fmt.Println(".............", t.Name(), "Superado")
}

// No debería haber primario
func (ts *testServer) primerPrimarioTest2(t *testing.T) {
	// t.Skip("SKIPPED primerPrimarioTest2")

	fmt.Println(t.Name(), ".....................")

	// Primer cliente por primera vez :
	// 		latido 0 y vista tentativa por respuesta en tiempo razonable
	vTentativa := ts.clienteLatido0(t, NODOCLIENTE1)

	// Preparar las vistas a comparar entre recibida y vista esperada
	vac := vistasAcomparar{t: t,
		recibido: vTentativa,
		referencia: gvcomun.Vista{Primario: NODOCLIENTE1,
			Copia:    msgsys.HOSTINDEFINIDO,
			NumVista: 1},
	}

	// Comprobar vista tentativa recibida
	vac.comprobar()

	fmt.Println(".............", t.Name(), "Superado")
}

// Hace a nodo 2 la copia
func (ts *testServer) PrimerCopiaTest3(t *testing.T) {
	//t.Skip("SKIPPED PrimerCopiaTest3")

	// solo nos interesa la vista tentativa devuelta por latido a Gestor Vistas
	_, ok := ts.SendReceive(NODOCLIENTE1,
		gvcomun.MsgLatido{-1, NODOCLIENTE1},
		gvcomun.ANSWERWAITTIME*time.Millisecond,
	)
	if !ok {
		t.Fatal(
			"Salta timeout esperando respuesta a latido -1 de cliente")
	}

	// Segundo cliente por primera vez:
	// 		latido 0 y vista tentativa por respuesta en tiempo razonable
	vTentativa := ts.clienteLatido0(t, NODOCLIENTE2)

	// Preparar las vistas a comparar entre recibida y vista esperada
	vac := vistasAcomparar{t: t,
		recibido: vTentativa,
		referencia: gvcomun.Vista{Primario: NODOCLIENTE1,
			Copia:    NODOCLIENTE2,
			NumVista: 2},
	}

	// Comprobar vista tentativa recibida
	vac.comprobar()

	fmt.Println(".............", t.Name(), "Superado")
}

// Copia toma relevo si primario falla
func (ts *testServer) CopiaTomaRelevoTest4(t *testing.T) {
	var vTentativa gvcomun.Vista

	// Confirmamos la primera vista válida (2, Nodo1, Nodo2)
	vTentativa = ts.clienteLatido(t, NODOCLIENTE1, 2)

	for i := 0; i <= gvcomun.LATIDOSFALLIDOS; i += 1 {
		// Mandar latidos de NODO2 durante 250 milis
		// hasta que el GV de por muerto a primario -> NODO1

		time.Sleep(time.Millisecond * gvcomun.INTERVALOLATIDOS)
		vTentativa = ts.clienteLatido(t, NODOCLIENTE2, vTentativa.NumVista)
	}
	fmt.Printf("--------------------- %v\n", vTentativa)

	// Preparar las vistas a comparar entre recibida y vista esperada
	vac := vistasAcomparar{t: t,
		recibido: vTentativa,
		referencia: gvcomun.Vista{Primario: NODOCLIENTE2,
			Copia:    msgsys.HOSTINDEFINIDO,
			NumVista: 3},
	}

	// Comprobar vista tentativa recibida
	vac.comprobar()
	fmt.Println(".............", t.Name(), "Superado")
}

// Servidor rearrancado se convierte en copia
func (ts *testServer) ServidorRearrancadoEsCopiaTest5(t *testing.T) {
	vista := ts.clienteLatido0(t, NODOCLIENTE1)
	// Preparar las vistas a comparar entre recibida y vista esperada
	vac := vistasAcomparar{t: t,
		recibido: vista,
		referencia: gvcomun.Vista{Primario: NODOCLIENTE2,
			Copia:    NODOCLIENTE1,
			NumVista: 4},
	}
	fmt.Printf("Vistas %v\n", vac)
	vac.comprobar()
	fmt.Println(".............", t.Name(), "Superado")
}

// Servidor en espera se convierte en copia si primario falla
func (ts *testServer) PromocionEsperaCopiaTest6(t *testing.T) {

	// NODO 2 confirma vista
	ts.clienteLatido(t, NODOCLIENTE2, 4)
	// Arranca NODO 3
	vistaN3 := ts.clienteLatido0(t, NODOCLIENTE3)
	// Latido de

	// Configuración -> {numVista = 4, Prim: NODO2, Cop: NODO1}
	for i := 0; i <= gvcomun.LATIDOSFALLIDOS; i += 1 {
		// Mandar latidos de N1 y N3 durante 250 milis
		// hasta que el GV de por muerto a primario -> NODO2

		time.Sleep(time.Millisecond * gvcomun.INTERVALOLATIDOS)
		ts.clienteLatido(t, NODOCLIENTE1, 4)
		vistaN3 = ts.clienteLatido(t, NODOCLIENTE3, 4)
	}
	// Configuración Deseada-> {numVista = 5, Prim: NODO1, Cop: NODO3}
	vac := vistasAcomparar{t: t,
		recibido: vistaN3,
		referencia: gvcomun.Vista{Primario: NODOCLIENTE1,
			Copia:    NODOCLIENTE3,
			NumVista: 5},
	}
	fmt.Printf("Vistas %v\n", vac)
	vac.comprobar()
	fmt.Println(".............", t.Name(), "Superado")
}

// Servidor en espera se convierte en copia si primario falla
func (ts *testServer) PrimarioAEsperaTest7(t *testing.T) {

	// Primario confirma vista
	ts.clienteLatido(t, NODOCLIENTE1, 5)
	// Añadir N2 a espera
	ts.clienteLatido0(t, NODOCLIENTE2)
	// Rearranca N1
	vistaN1 := ts.clienteLatido0(t, NODOCLIENTE1)

	// Configuración Deseada-> {numVista = 5, Prim: NODO1, Cop: NODO3}
	vac := vistasAcomparar{t: t,
		recibido: vistaN1,
		referencia: gvcomun.Vista{
			Primario: NODOCLIENTE3,
			Copia:    NODOCLIENTE2,
			NumVista: 6},
	}
	fmt.Printf("Vistas %v\n", vac)
	vac.comprobar()
	fmt.Println(".............", t.Name(), "Superado")
}

// Nuevo servidor primario no confirma la vista
func (ts *testServer) EsperaConfirmacionTest8(t *testing.T) {
	// Partimos de la configuración del test 7
	// {numVista = 6, Prim: NODO3, Cop: NODO2} <- Tentativa
	// {numVista = 5, Prim: NODO1, Cop: NODO3} <- Valida

	// Hasta que Nodo3 no mande latido con 6, no se confirma
	if ts.VistasSonConsistentes(t, 5) {
		t.Fatal("Vista valida y tentativa son iguales antes de la confirmacion")
	}

	// Comprobar que se mantiene el estado en el tiempo
	for i := 0; i <= gvcomun.LATIDOSFALLIDOS; i += 1 {
		// Mandar latidos de N1 y N3 durante 250 milis hasta que el GV de por muerto a primario

		time.Sleep(time.Millisecond * gvcomun.INTERVALOLATIDOS)
		ts.clienteLatido(t, NODOCLIENTE3, 5) // <-- Primario no confirma vista
		ts.clienteLatido(t, NODOCLIENTE2, 6) // <-- Copia envía numVistaTentativa
	}

	if ts.VistasSonConsistentes(t, 5) {
		t.Fatal("Vista valida y tentativa son iguales antes de la confirmacion")
	}
	// PRIMARIO CONFIRMA VISTA
	ts.clienteLatido(t, NODOCLIENTE3, 6)

	// Comprobar que se ha confirmado
	if !ts.VistasSonConsistentes(t, 6) {
		t.Fatal("Vista valida y tentativa son distintas después de la confirmacion")
	}
	fmt.Println(".............", t.Name(), "Superado")
}

// Comprobar que la vista tentativa y la valida son distintas
func (ts *testServer) VistasSonConsistentes(t *testing.T, numVistaAnt int) bool {
	// Lanzamos una peticion de vista valida y tentativa
	vistaTentativa := ts.clienteLatido(t, NODOCLIENTE3, numVistaAnt)
	msgVistaValida, ok := ts.SendReceive(NODOCLIENTE3,
		gvcomun.MsgPeticionVistaValida{NODOCLIENTE3},
		gvcomun.ANSWERWAITTIME*time.Millisecond,
	)
	if !ok {
		t.Fatal("Salta timeout esperando haciendo peticion de vista valida")
	}
	vistaValida, ok2 := msgVistaValida.(gvcomun.MsgVistaValida)
	if !ok2 {
		t.Fatal("No es un mensaje de vista valida")
	}

	// Comprobamos que no tengan el mismo número de vista
	if vistaTentativa.NumVista == vistaValida.Vista.NumVista {
		return true
	}
	return false
}

// Si anteriores servidores caen, un nuevo servidor no
func (ts *testServer) ServidorSinInicializarNoEsPrimarioTest9(t *testing.T) {
	// NODO1 se añade a espera
	vista := ts.clienteLatido0(t, NODOCLIENTE1)
	// Aseguramos que estamos en la vista {6, P:NODO3, C:NODO2}
	vistasAcomparar{
		recibido: vista,
		referencia: gvcomun.Vista{
			NumVista: 6,
			Primario: NODOCLIENTE3,
			Copia:    NODOCLIENTE2,
		},
		t: t,
	}.comprobar()
	var err error
	for i := 0; i <= gvcomun.LATIDOSFALLIDOS; i += 1 {
		// Mandar latidos de N1 hasta que PRIMARIO Y COPIA fallen
		// Y por tanto el GV tambien caiga

		time.Sleep(time.Millisecond * gvcomun.INTERVALOLATIDOS)
		// Puede fallar cualquier latido, pero lo comprobamos tras el tiempo

		err = ts.SendConError(NODOGV, gvcomun.MsgLatido{
			NumVista:  vista.NumVista,
			Remitente: NODOCLIENTE1,
		})
		if err != nil {
			t.Logf("%v", err)
			break
		}
	}
	// Si no devuelve error el dialTCP, fallamos
	if err != nil {
		t.Log("Send ha dado error")
	} else {
		t.Fatal("El gestor de vistas ha contestado y debería estar caído")
	}
	fmt.Println(".............", t.Name(), "Superado")
}

func (ts *testServer) clienteLatido0(t *testing.T,
	nodoCliente msgsys.HostPuerto) gvcomun.Vista {

	return ts.clienteLatido(t, nodoCliente, 0)
}

func (ts *testServer) clienteLatido(t *testing.T,
	nodoCliente msgsys.HostPuerto, numVista int) gvcomun.Vista {

	// solo nos interesa la vista tentativa devuelta por latido a Gestor Vistas
	m, ok := ts.SendReceive(nodoCliente,
		gvcomun.MsgLatido{numVista, nodoCliente},
		gvcomun.ANSWERWAITTIME*time.Millisecond,
	)
	if !ok {
		t.Fatal("Salta timeout esperando latido 0 de cliente")
	}

	switch x := m.(type) {
	case gvcomun.MsgVistaTentativa:
		return x.Vista // salida correcta
	default:
		t.Fatalf(t.Name(),
			"Mensaje recibido INCORRECTO en primerPrimario: %#v", x)
	}

	// no debería llegar a ejecutarse, pero se pone por error compilacion
	return gvcomun.Vista{}
}

// --------------------------------------------------------------------------
// FUNCIONES DE APOYO

type vistasAcomparar struct {
	recibido   gvcomun.Vista
	referencia gvcomun.Vista
	t          *testing.T
}

func (vs vistasAcomparar) comprobar() {
	if vs.recibido.Primario != vs.referencia.Primario {
		vs.t.Fatalf(
			"%s : PRIMARIO recibido (%s) y de referencia (%s) no coinciden",
			vs.t.Name(), vs.recibido.Primario, vs.referencia.Primario)
	}
	if vs.recibido.Copia != vs.referencia.Copia {
		vs.t.Fatalf("%s : COPIA recibido (%s) y de referencia (%s) no coinciden",
			vs.t.Name(), vs.recibido.Copia, vs.referencia.Copia)
	}
	if vs.recibido.NumVista != vs.referencia.NumVista {
		vs.t.Fatalf(
			"%s : NUM VISTA recibido (%d) y de referencia (%d) no coinciden",
			vs.t.Name(), vs.recibido.NumVista, vs.referencia.NumVista)
	}
}

func (ts *testServer) lanzarNodo1() {
	cltssh.ExecMutipleHosts(CLTVTSCMD+NODOCLIENTE1+" "+NODOGV+" "+NODOTEST,
		[]string{MAQUINA3}, ts.cmdOutput, PRIVKEYFILE)
}

// Hace un send devolviendo un error en vez de hacer log.Fatal
func (ts *testServer) SendConError(destinatario msgsys.HostPuerto, msg msgsys.Message) error {
	conn, err := net.Dial("tcp", string(destinatario))
	if err != nil {
		return err
	}
	// fmt.Printf("Message for encoder: %#v \n", msg)
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(&msg)

	_ = conn.Close()
	return nil
}
