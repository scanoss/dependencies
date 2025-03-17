package example

import (
	"fmt"
	"sync"
	"time"
)

// Trabajo simple
type Job struct {
	ID            int
	GenerateMore  bool // Si este trabajo generará más trabajos
	NumToGenerate int  // Cuántos trabajos adicionales generará
	Depth         int  // Profundidad restante
}

// Resultado de un trabajo
type Result struct {
	JobID   int
	NewJobs []Job
	Depth   int
}

func main() {
	// Crear canales de input y output
	input := make(chan Job, 10)
	output := make(chan Result, 10)

	// Mutex para proteger el contador de trabajos
	var mu sync.Mutex
	pendingJobs := 2 // Tenemos 2 trabajos iniciales

	// Grupo de espera para los workers
	var wg sync.WaitGroup

	// Iniciar 3 workers
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go worker(i, input, output, &wg)
	}

	// Goroutine para procesar resultados y realimentar el canal de input
	go func() {
		for result := range output {
			fmt.Printf("Procesando resultado del trabajo %d\n", result.JobID)

			// Contar nuevos trabajos a generar
			newJobsCount := 0
			if result.Depth > 0 {
				newJobsCount = len(result.NewJobs)
			}

			// Incrementar contador ANTES de añadir nuevos trabajos
			if newJobsCount > 0 {
				mu.Lock()
				pendingJobs += newJobsCount
				fmt.Printf("  [+] Incrementando contador a %d por %d nuevos trabajos\n",
					pendingJobs, newJobsCount)
				mu.Unlock()
			}

			// Añadir nuevos trabajos al canal de input
			if result.Depth > 0 {
				for _, newJob := range result.NewJobs {
					fmt.Printf("  Realimentando nuevo trabajo: %d (desde trabajo %d)\n",
						newJob.ID, result.JobID)
					input <- newJob
				}
			}

			// Decrementar contador para el trabajo completado
			mu.Lock()
			pendingJobs--
			fmt.Printf("  [-] Decrementando contador a %d (trabajo %d completado)\n",
				pendingJobs, result.JobID)

			// Si no quedan trabajos pendientes, cerrar el canal de input
			if pendingJobs == 0 {
				fmt.Println("  [!] No quedan trabajos pendientes, cerrando canal de input")
				close(input)
			}
			mu.Unlock()
		}
	}()

	// Agregar trabajos iniciales al canal de input
	initialJobs := []Job{
		{ID: 1, GenerateMore: true, NumToGenerate: 2, Depth: 2},
		{ID: 2, GenerateMore: false, Depth: 0},
	}

	for _, job := range initialJobs {
		fmt.Printf("Añadiendo trabajo inicial: %d\n", job.ID)
		input <- job
	}

	// Esperar a que terminen todos los workers
	wg.Wait()
	fmt.Println("Todos los workers han terminado, cerrando canal de output")

	// Cerrar canal de output
	close(output)

	fmt.Println("Proceso terminado correctamente")
}

// Función worker que procesa trabajos
func worker(id int, input <-chan Job, output chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range input {
		fmt.Printf("Worker %d: Procesando trabajo %d\n", id, job.ID)

		// Simular trabajo
		time.Sleep(50 * time.Millisecond)

		// Preparar nuevos trabajos si es necesario
		var newJobs []Job
		if job.GenerateMore && job.Depth > 0 {
			for i := 0; i < job.NumToGenerate; i++ {
				newJobID := job.ID*10 + i + 1
				newJobs = append(newJobs, Job{
					ID:            newJobID,
					GenerateMore:  true,
					NumToGenerate: 1,
					Depth:         job.Depth - 1,
				})
			}
		}

		// Enviar resultado al canal de output
		output <- Result{
			JobID:   job.ID,
			NewJobs: newJobs,
			Depth:   job.Depth - 1,
		}
	}

	fmt.Printf("Worker %d: Terminando\n", id)
}
