/*
ESTE ES EL ÚNICO ARCHIVO QUE SE PUEDE MODIFICAR

RECOMENDACIÓN: Solo modificar a partir de la parte
                donde se encuentran la explicación
                de las otras variables.

*/

package main

import (
    "bytes"
    "fmt"
    "io"
    "log"
    "math/rand"
    "net"
    "strconv"
    "strings"
    "sync"
    "time"
)

var (
    buf    bytes.Buffer
    logger = log.New(&buf, "logger: ", log.Lshortfile)
    msg    string
)

const (
    maxAvionesEnCola = 10
    maxAviones       = 10 // Límite de aviones por categoría
)

// Avion simula la información de un avión.
type Avion struct {
    id           int
    categoria    string
    numPasajeros int
}

// Canales para cada etapa.
var (
    torreControl = make(chan Avion, maxAvionesEnCola)
    pistas       = make(chan Avion, maxAvionesEnCola)
    puertas      = make(chan Avion, maxAvionesEnCola)

    // Espera hasta que terminen las goroutines de creación de aviones.
    wg sync.WaitGroup
)

func main() {
    rand.Seed(time.Now().UnixNano())

    // Lanzamos goroutines “consumidoras” para leer de cada canal.
    go towerWorker()
    go pistaWorker()
    go puertaWorker()

    // Conexión al servidor.
    conn, err := net.Dial("tcp", "localhost:8000")
    if err != nil {
        logger.Fatal(err)
    }
    defer conn.Close()

    // Bucle que lee mensajes desde el servidor.
    buf := make([]byte, 512)
    for {
        n, err := conn.Read(buf)
        if err == io.EOF {
            break
        }
        if err != nil {
            fmt.Println(err)
            continue
        }
        if n > 0 {
            msg = string(buf[:n])
            fmt.Println("len: " + strconv.Itoa(n) + " msg: " + msg)
            handleOperation(msg)
        }
    }
}

// handleOperation interpreta el mensaje (si es numérico) y llama a simulateAirport.
func handleOperation(msg string) {
    msg = strings.TrimSpace(msg)

    // Ignoramos mensajes no numéricos.
    if _, err := strconv.Atoi(msg); err != nil {
        return
    }
    operation, err := strconv.Atoi(msg)
    if err != nil {
        fmt.Println("Error converting message to integer:", err)
        return
    }

    switch operation {
    case 0:
        fmt.Println("Aeropuerto Inactivo: No hay Aterrizajes")
    case 1:
        fmt.Println("Solo Categoría A: Solo podrán acceder los aviones de dicha categoría")
        simulateAirport("A")
    case 2:
        fmt.Println("Solo Categoría B: Solo podrán acceder los aviones de dicha categoría")
        simulateAirport("B")
    case 3:
        fmt.Println("Solo Categoría C: Solo podrán acceder los aviones de dicha categoría")
        simulateAirport("C")
    case 4:
        fmt.Println("Prioridad Categoría A: La prioridad del aterrizaje lo tienen los aviones de dicha categoría")
        simulateAirport("A")
    case 5:
        fmt.Println("Prioridad Categoría B: La prioridad del aterrizaje lo tienen los aviones de dicha categoría")
        simulateAirport("B")
    case 6:
        fmt.Println("Prioridad Categoría C: La prioridad del aterrizaje lo tienen los aviones de dicha categoría")
        simulateAirport("C")
    case 7, 8:
        fmt.Println("No definido: Se mantiene el estado anterior")
    case 9:
        fmt.Println("Aeropuerto Cerrado Temporal: No hay Aterrizajes")
    default:
        fmt.Println("Operación desconocida")
    }
}

// simulateAirport crea goroutines (una por cada avión) que envían datos a los canales.
func simulateAirport(categoria string) {
    for i := 0; i < maxAviones; i++ {
        numPasajeros := rand.Intn(150) + 1
        avion := Avion{
            id:           i,
            categoria:    categoria,
            numPasajeros: numPasajeros,
        }
        wg.Add(1)
        go handleAvion(avion)
    }
    // Esperamos a que terminen todas las goroutines que creamos.
    go func() {
        wg.Wait()
        // Si quisieras reiniciar canales o algo similar,
        // podrías hacerlo aquí, pero en este ejemplo se dejan abiertos.
    }()
}

// handleAvion simula los pasos del avión y envía sus datos a cada canal.
func handleAvion(avion Avion) {
    defer wg.Done()

    // Simula tiempo en torre de control.
    fmt.Printf("Avión %d de categoría %s con %d pasajeros está esperando en la torre de control\n",
        avion.id, avion.categoria, avion.numPasajeros)
    time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
    torreControl <- avion

    // Simula tiempo en pista.
    fmt.Printf("Avión %d de categoría %s está esperando en la pista\n",
        avion.id, avion.categoria)
    time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
    pistas <- avion

    // Simula tiempo en puerta.
    fmt.Printf("Avión %d de categoría %s está esperando en la puerta\n",
        avion.id, avion.categoria)
    time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
    puertas <- avion

    fmt.Printf("Avión %d de categoría %s ha completado el proceso\n",
        avion.id, avion.categoria)
}

// A continuación, definimos goroutines “consumidoras” que leen de cada canal.
// De este modo, los envíos a los canales no bloquearán definitivamente.
func towerWorker() {
    for avion := range torreControl {
        // Aquí podrías simular el tiempo que tarda la torre con este avión.
        fmt.Printf("[TorreControl] Procesando avión %d (cat. %s)\n", avion.id, avion.categoria)
        time.Sleep(1 * time.Second)
    }
}

func pistaWorker() {
    for avion := range pistas {
        // Simula el trabajo con avion.
        fmt.Printf("[Pista] Procesando avión %d (cat. %s)\n", avion.id, avion.categoria)
        time.Sleep(1 * time.Second)
    }
}

func puertaWorker() {
    for avion := range puertas {
        // Simula el trabajo con avion.
        fmt.Printf("[Puerta] Procesando avión %d (cat. %s)\n", avion.id, avion.categoria)
        time.Sleep(1 * time.Second)
    }
}

