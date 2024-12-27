package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Estructura de avión con información de vuelo
type Aircraft struct {
	id       int
	passengers int
	category string
	arrival  time.Time
}

// Pistas y puertas del aeropuerto
type Runway struct {
	id int
}

type Gate struct {
	id int
}

// Cola con prioridad de aterrizajes
type PriorityQueue []Aircraft

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	priorityMap := map[string]int{"A": 1, "B": 2, "C": 3}
	if pq[i].category != pq[j].category {
		return priorityMap[pq[i].category] < priorityMap[pq[j].category]
	}
	return pq[i].arrival.Before(pq[j].arrival)
}

func (pq PriorityQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

// Aeropuerto con toda su configuración
type Airport struct {
	mu              sync.Mutex
	runways         chan Runway
	gates           chan Gate
	landingQueue    PriorityQueue
	currentPriority string
	state           int
	maxQueueSize    int
	activeLandings  int
	activeGates     int
	totalPlanes     int
	landedPlanes    int
	done            chan bool
}

var (
	logBuffer  bytes.Buffer
	logger     = log.New(&logBuffer, "logger: ", log.Lshortfile)
	wg         sync.WaitGroup
	numRunways = 3
	numGates   = 10
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Configuración inicial de aviones
	numCategoryA := 10
	numCategoryB := 10
	numCategoryC := 10
	totalPlanes := numCategoryA + numCategoryB + numCategoryC

	conn, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		logger.Fatal(err)
	}
	defer conn.Close()

	airport := &Airport{
		runways:      make(chan Runway, numRunways),
		gates:        make(chan Gate, numGates),
		landingQueue: PriorityQueue{},
		state:        0,
		maxQueueSize: 50,
		done:         make(chan bool),
		totalPlanes:  totalPlanes,
	}

	// Inicializando pistas y puertas
	for i := 1; i <= numRunways; i++ {
		airport.runways <- Runway{id: i}
	}

	for i := 1; i <= numGates; i++ {
		airport.gates <- Gate{id: i}
	}

	// Generar aviones al azar
	go generateAirplanes(airport, numCategoryA, numCategoryB, numCategoryC)

	// Gestionar comunicación con el cliente
	go handleMessages(conn, airport)

	// Manejo de aterrizajes
	go manageLandings(airport)

	// Monitorear la simulación
	go monitorSimulation(airport)

	// Mantener la ejecución de la simulación
	wg.Add(1)
	<-airport.done
	wg.Done()
	wg.Wait()
}

func generateAirplanes(airport *Airport, numA, numB, numC int) {
	id := 1
	aircraftList := make([]Aircraft, 0, numA+numB+numC)

	// Generación de aviones de cada categoría
	for i := 0; i < numA; i++ {
		aircraftList = append(aircraftList, Aircraft{
			id:        id,
			category:  "A",
			passengers: rand.Intn(100) + 101,
			arrival:   time.Now(),
		})
		id++
	}
	for i := 0; i < numB; i++ {
		aircraftList = append(aircraftList, Aircraft{
			id:        id,
			category:  "B",
			passengers: rand.Intn(51) + 50,
			arrival:   time.Now(),
		})
		id++
	}
	for i := 0; i < numC; i++ {
		aircraftList = append(aircraftList, Aircraft{
			id:        id,
			category:  "C",
			passengers: rand.Intn(50),
			arrival:   time.Now(),
		})
		id++
	}

	// Aleatorizar la lista de aviones
	rand.Shuffle(len(aircraftList), func(i, j int) {
		aircraftList[i], aircraftList[j] = aircraftList[j], aircraftList[i]
	})

	// Encolar los aviones
	for _, aircraft := range aircraftList {
		if airport.enqueueAircraft(aircraft) {
			// Successfully enqueued aircraft
		}
	}
}

// Encolar un avión en la cola de aterrizajes
func (airport *Airport) enqueueAircraft(aircraft Aircraft) bool {
	airport.mu.Lock()
	defer airport.mu.Unlock()

	if len(airport.landingQueue) >= airport.maxQueueSize {
		fmt.Printf("Queue for Category %s is full. Aircraft %d rejected.\n", aircraft.category, aircraft.id)
		return false
	}

	airport.landingQueue = append(airport.landingQueue, aircraft)
	fmt.Printf("Aircraft %d from Category %s queued for landing.\n", aircraft.id, aircraft.category)
	return true
}

// Procesar mensajes del servidor
func handleMessages(conn net.Conn, airport *Airport) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := strings.TrimSpace(scanner.Text())
		processServerMessage(msg, airport)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Connection error:", err)
	}
	airport.mu.Lock()
	airport.state = 9
	airport.currentPriority = ""
	airport.mu.Unlock()
}

// Procesar los mensajes recibidos del servidor
func processServerMessage(msg string, airport *Airport) {
	code, err := strconv.Atoi(msg)
	if err != nil {
		logger.Println("Invalid message:", msg)
		return
	}

	airport.mu.Lock()
	defer airport.mu.Unlock()

	switch code {
	case 0:
		airport.state = 0
		airport.currentPriority = ""
		fmt.Println("Airport Inactive: No landings allowed.")
	case 1:
		airport.state = 1
		airport.currentPriority = "A"
		fmt.Println("Only Category A can land.")
	case 2:
		airport.state = 2
		airport.currentPriority = "B"
		fmt.Println("Only Category B can land.")
	case 3:
		airport.state = 3
		airport.currentPriority = "C"
		fmt.Println("Only Category C can land.")
	case 4:
		airport.state = 4
		airport.currentPriority = "A"
		fmt.Println("Priority landing for Category A.")
	case 5:
		airport.state = 5
		airport.currentPriority = "B"
		fmt.Println("Priority landing for Category B.")
	case 6:
		airport.state = 6
		airport.currentPriority = "C"
		fmt.Println("Priority landing for Category C.")
	case 7, 8:
		fmt.Println("State unchanged.")
	case 9:
		airport.state = 9
		airport.currentPriority = ""
		fmt.Println("Airport temporarily closed.")
	default:
		fmt.Println("Unrecognized code.")
	}
}

// Gestionar aterrizajes de aviones
func manageLandings(airport *Airport) {
	for {
		airport.mu.Lock()
		if airport.state == 0 || airport.state == 9 {
			airport.mu.Unlock()
			time.Sleep(1 * time.Second)
			continue
		}

		// Ordenar la cola de aterrizajes
		sort.Sort(airport.landingQueue)

		var aircraft *Aircraft
		var aircraftIndex int

		// Buscar el avión adecuado para aterrizar según el estado actual
		switch airport.state {
		case 1, 2, 3:
			for i, a := range airport.landingQueue {
				if a.category == airport.currentPriority {
					aircraft = &a
					aircraftIndex = i
					break
				}
			}
		case 4, 5, 6:
			priorityFound := false
			for i, a := range airport.landingQueue {
				if a.category == airport.currentPriority {
					aircraft = &a
					aircraftIndex = i
					priorityFound = true
					break
				}
			}
			if !priorityFound && len(airport.landingQueue) > 0 {
				aircraft = &airport.landingQueue[0]
				aircraftIndex = 0
			}
		default:
			if len(airport.landingQueue) > 0 {
				aircraft = &airport.landingQueue[0]
				aircraftIndex = 0
			}
		}

		// Aterrizar el avión si se cumplen las condiciones
		if aircraft != nil && len(airport.runways) > 0 {
			airport.landingQueue = append(airport.landingQueue[:aircraftIndex], airport.landingQueue[aircraftIndex+1:]...)
			airport.activeLandings++

			runway := <-airport.runways
			airport.mu.Unlock()

			go landAircraft(airport, *aircraft, runway)
		} else {
			airport.mu.Unlock()
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// Manejar el aterrizaje de un avión
func landAircraft(airport *Airport, aircraft Aircraft, runway Runway) {
	fmt.Printf("Aircraft %d from Category %s is landing on Runway %d.\n", aircraft.id, aircraft.category, runway.id)
	landingTime := time.Duration(rand.Intn(3)+1) * time.Second
	time.Sleep(landingTime)
	fmt.Printf("Aircraft %d landed and moving to gate.\n", aircraft.id)

	// Asignar puerta
	select {
	case gate := <-airport.gates:
		fmt.Printf("Gate %d: Aircraft %d assigned.\n", gate.id, aircraft.id)
		go handleGate(airport, aircraft, gate)
	default:
		fmt.Printf("No gates available for Aircraft %d. Waiting...\n", aircraft.id)
		time.Sleep(1 * time.Second)
		airport.mu.Lock()
		airport.landingQueue = append([]Aircraft{aircraft}, airport.landingQueue...)
		airport.mu.Unlock()
	}

	// Liberar la pista
	airport.runways <- runway
	airport.mu.Lock()
	airport.activeLandings--
	airport.landedPlanes++
	airport.mu.Unlock()
}

// Manejar el desembarque del avión en la puerta
func handleGate(airport *Airport, aircraft Aircraft, gate Gate) {
	fmt.Printf("Gate %d: Aircraft %d starting disembarkation.\n", gate.id, aircraft.id)
	disembarkTime := time.Duration(rand.Intn(3)+1) * time.Second
	time.Sleep(disembarkTime)
	fmt.Printf("Gate %d: Aircraft %d disembarked.\n", gate.id, aircraft.id)
	airport.mu.Lock()
	airport.activeGates--
	airport.mu.Unlock()
	airport.gates <- gate
}

// Monitorear la finalización de la simulación
func monitorSimulation(airport *Airport) {
	for {
		airport.mu.Lock()
		complete := airport.landedPlanes >= airport.totalPlanes && len(airport.landingQueue) == 0 &&
			airport.activeLandings == 0 && airport.activeGates == 0
		airport.mu.Unlock()

		if complete && airport.state != 9 {
			airport.done <- true
			return
		}
		time.Sleep(1 * time.Second)
	}
}
