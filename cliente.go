package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxAvionesEnCola = 10
)
var (
	torreControl = make(chan Avion, maxAvionesEnCola)
	pistas       = make(chan Avion, maxAvionesEnCola)
	puertas      = make(chan Avion, maxAvionesEnCola)

	wg sync.WaitGroup

	estadoActual  int
	limiteAviones = map[string]int{
		"A": 5, // Default values, can be changed
		"B": 5,
		"C": 5,
	}
	avionesRestantes = map[string]int{
		"A": 5,
		"B": 5,
		"C": 5,
	}
	mutex  sync.Mutex
	logger = log.Default()
)

type Avion struct {
	id           int
	categoria    string
	numPasajeros int
}
	
	
func main() {
	rand.Seed(time.Now().UnixNano())

	// Add initial setup for plane limits
	setupPlaneLimits()

	go towerWorker()
	go pistaWorker()
	go puertaWorker()

	conn, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		logger.Fatal(err)
	}
	defer conn.Close()

	buf := make([]byte, 512)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		if n > 0 {
			msg := string(buf[:n])
			handleOperation(strings.TrimSpace(msg))
		}
	}
}

func setupPlaneLimits() {
	fmt.Println("Configuración inicial de límites de aviones:")
	for categoria := range limiteAviones {
		fmt.Printf("Ingrese número de aviones para categoría %s: ", categoria)
		var limite int
		fmt.Scanf("%d", &limite)
		mutex.Lock()
		limiteAviones[categoria] = limite
		avionesRestantes[categoria] = limite
		mutex.Unlock()
	}
}


func handleOperation(msg string) {
	if num, err := strconv.Atoi(msg); err == nil {
		estadoActual = num
		fmt.Printf("Estado actualizado a %d\n", estadoActual)
		switch estadoActual {
		case 0:
			fmt.Println("Aeropuerto Inactivo: No hay Aterrizajes")
		case 1:
			simulateAirport("A")
		case 2:
			simulateAirport("B")
		case 3:
			simulateAirport("C")
		case 4, 5, 6:
			simulatePriority(estadoActual)
		case 7, 8:
			fmt.Println("Estado indefinido: Se mantiene el estado anterior")
		case 9:
			fmt.Println("Aeropuerto Cerrado Temporal: No hay Aterrizajes")
		default:
			fmt.Println("Operación desconocida")
		}
	}
}
func simulateAirport(categoria string) {
    mutex.Lock()
    if avionesRestantes[categoria] <= 0 {
        fmt.Printf("No quedan aviones disponibles de categoría %s\n", categoria)
        mutex.Unlock()
        return
    }
    mutex.Unlock()

    for i := 0; i < maxAvionesEnCola; i++ {
        numPasajeros := rand.Intn(150) + 1
        avion := Avion{
            id:           i,
            numPasajeros: numPasajeros,
            categoria:    determineCategory(numPasajeros),
        }

        mutex.Lock()
        availableSlots := avionesRestantes[avion.categoria] > 0
        isPriority := estadoActual >= 4 && estadoActual <= 6
        matchesCategory := (avion.categoria == categoria)
        
        if availableSlots && (isPriority || matchesCategory) {
            avionesRestantes[avion.categoria]--
            mutex.Unlock()
            wg.Add(1)
            go handleAvion(avion)
        } else {
            mutex.Unlock()
            fmt.Printf("Avión %d de categoría %s con %d pasajeros queda en espera\n",
                avion.id, avion.categoria, numPasajeros)
        }
    }
}



func simulatePriority(estado int) {

	switch estado {
	case 4:
		// Prioridad Categoría A
		simulateAirport("A")
	case 5:
		// Prioridad Categoría B
		simulateAirport("B")
	case 6:
		// Prioridad Categoría C
		simulateAirport("C")
	}
}

func handleAvion(avion Avion) {
	defer wg.Done()
	if isValidCategory(avion) {
		fmt.Printf("Avión %d de categoría %s con %d pasajeros está esperando en la torre de control\n",
			avion.id, avion.categoria, avion.numPasajeros)
		time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
		torreControl <- avion
	} else {
		// El avión no es válido para el estado actual, no se procesa.
		fmt.Printf("Avión %d de categoría %s no permitido en el estado actual, quedando en espera\n",
			avion.id, avion.categoria)
	}
}

func towerWorker() {
	for avion := range torreControl {
		// Procesamos aviones según el estado y la categoría
		if isValidCategory(avion) {
			// Si hay aviones de categoría C en espera, se procesan inmediatamente
			if avion.categoria == "C" || (avion.categoria != "C" && !isCategoryCInQueue()) {
				fmt.Printf("[TorreControl] Procesando avión %d (cat. %s)\n", avion.id, avion.categoria)
				time.Sleep(1 * time.Second)
				pistas <- avion
			} else {
				// Si no hay aviones de categoría C y el avión no es de esta categoría, se procesa
				fmt.Printf("Avión %d de categoría %s quedando en espera en la torre de control\n", avion.id, avion.categoria)
			}
		}
	}
}

func pistaWorker() {
	for avion := range pistas {
		if isValidCategory(avion) {
			fmt.Printf("[Pista] Procesando avión %d (cat. %s)\n", avion.id, avion.categoria)
			time.Sleep(1 * time.Second)
			puertas <- avion
		}
	}
}

func puertaWorker() {
	for avion := range puertas {
		if isValidCategory(avion) {
			fmt.Printf("[Puerta] Procesando avión %d (cat. %s)\n", avion.id, avion.categoria)
			time.Sleep(1 * time.Second)
			fmt.Printf("Avión %d de categoría %s ha completado el proceso\n", avion.id, avion.categoria)
		}
	}
}

func isValidCategory(avion Avion) bool {
    switch estadoActual {
    case 1, 4:
        return true // For priority A, all categories are valid but A has priority
    case 2, 5:
        return true // For priority B, all categories are valid but B has priority
    case 3, 6:
        return true // For priority C, all categories are valid but C has priority
    default:
        return false
    }
}


func determineCategory(numPasajeros int) string {
	switch {
	case numPasajeros > 100:
		return "A"
	case numPasajeros >= 50:
		return "B"
	default:
		return "C"
	}
}

func isCategoryCInQueue() bool {
    mutex.Lock()
    defer mutex.Unlock()
    return avionesRestantes["C"] > 0
}