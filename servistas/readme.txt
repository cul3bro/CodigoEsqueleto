>>> Contenido de la  version 2 del módulo "servistas" :
        * "g.mod": fichero de configuración básica de modulo
        * "cmd": directorio  ejecutables de servidor y cliente remoto mínimo
        * "internal": paquetes soporte para puesta en marcha de servicio de vistas
            - "integracion_test": paquete tests de integración servicio de vistas
            - "srvvts": paquete para código de funcionamiento de servidor
            - "msgsys": paquete código de mensajería
            - "cltssh": paquete cliente ssh con autentificación clave pública
        * "pkg" : librería cliente de servicio vistas exportada a exterior módulo
        * "vendor": código completo local de modulos externos necesarios (ssh..)

* Esta version v2 puede ser utilizada como ejemplo, consulta, guía o referencia
    - Para implementar la práctica 4 y práctica 5
    - Hay modificaciones en casi todos los ficheros
    - Se incluye paquete "cltssh" de ejecución remota ssh
        - Adaptado para ejecución con clave publica
    - Se ha modificaddo prácticamente todos los ficheros
    - Se ha incluido directorio "vendor" para este modulo
        - Con todo los modulos y paquetes necesarios para paquete externo "ssh"
    - Incluye tests de integración para las 3 primeras pruebas
    
* Para una versión funcional de la práctica 4,
    esta versión v2 del esqueleto solo requeriría
        - modificar el PATH completo del directorio del modulo "servistas"
            - en fichero "cltssh_test.go", línea 13
            - y fichero "integración _test.go", constante "PATH"
        - modificar PATH completo del fichero de clave privada para ssh
            - ubicado en fichero "integración _test.go", constante "PRIVKEYFILE"
            - y en fichero "cltssh_test.go", línea 15
        - y completar código : 
            - en el paquete "srvvts"
            - de pruebas que faltan en el paquete "integracion_test",
                - Están disponibles las 3 primeras pruebas completadas
                - y, finalmente,  los tests de este último paquete
        
* Probar todos los tests del modulo con : go test ./...
    - Su ejecución es concurrente, luego puede haber errores por interferencias
        - Si se reejecuta conserva en cache ejecuciones correctas de tests
            - Luego menos interferencias
            - y tiempos de puesta en marcha y comunicación van mejor
            - Borrado de cache : go clean -cache -modcache -i -r
            
* Los test de integración utilizan el mecanismo de "Subtests"
    - Documentación en sección "Subtests" de :
        - https://golang.org/pkg/testing/
        
* El modo depuración de tests se obtiene con :
        - o "go test -v"
        - o utilizando la opción "debug test" en vscode o editor similiar
    - con ello, además del depurador, visualiza salidas de "fmt" y "log"
    - SI EJECUCION ERRONEA de tests os pueden quedar PROCESOS SSH SIN TERMINAR
        - ELIMINARLOS con "pkill main"
            - En cada máquina distribuida si ejecución en distribuido
        
* Los tiempos de espera para :
        - arranque de servidores remotos con ssh en diferentes tests
        - utilización de funciones "SendReceive" y "ReceiveTimed"
    - Dependen de máquina o entorno distribuido de ejecución
        - Están ajustados para 2ª ejecución distribuida en el laboratorio 1.02
            - Primera ejecución es erronea porque busca ficheros codigo
                - en servidor remoto NFS (planta baja)
            - Segunda ejecución los obtiene de cache local del sistema de ficheros
                
* El ejecutable en el directorio "testcltvts" simula un cliente mínimo:
    - pero ejecutandose remotamente
        - unicamente reencamina mensajes entre :
            - nodo de tests
            - y nodo gestor de vistas
            
* Se utiliza en el código el concepto de "embedded structs"
        - https://golang.org/ref/spec#Struct_types
        - https://stackoverflow.com/questions/34079466/embedded-struct
    - Para ampliar de forma directa la funcionalidad :
        - de los servidores gestor de vistas, cliente de vistas, nodo de test
            - En los tipos de datos :
                - "srvvts.ServVistas",
                - "clienteTest" ubicado en "testcltvt/main.go"
                - "testServer" ubicado en "integracion_test/integracion_test.go"
            - en lo que respecta al sistema de mensajería para todos ellos
                - empotrando el tipo, campos y métodos de "msgsys. MsgSys"
                    - métodos Me, .Send, SendReceive, InternalSend, Receive, ReceiveTimed
            - y al cliente de vistas
                - empotrando tanto tipo, campos y métodos de "cltvts.CltVts"
                    - métodos Latido, PeticionPrimario, PeticionVistaValida
                    - pero solo en tipo "srvvts.ServVistas"