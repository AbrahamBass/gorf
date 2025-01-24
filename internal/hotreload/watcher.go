package hotreload

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	// "syscall"
	"net"
	"time"

	"github.com/fsnotify/fsnotify"
)

func HotReload(mainFile string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Error al crear watcher:", err)
	}
	defer watcher.Close()

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error al obtener directorio actual:", err)
	}

	filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if err := watcher.Add(path); err != nil {
				log.Println("Error agregando", path, ":", err)
			}
		}

		return nil
	})

	var (
		cmd       *exec.Cmd
		restartCh = make(chan struct{}, 1) // Canal para debounce
	)

	startProcess := func() {
		maxRetries := 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if isPortAvailable("8080") {
				cmd = exec.Command("go", "run", mainFile)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Start(); err != nil {
					log.Println("Error al iniciar proceso:", err)
					return
				}
				log.Println("Servidor iniciado con PID:", cmd.Process.Pid)
				return
			}
			log.Printf("Intento %d: puerto 8080 ocupado\n", attempt)
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		log.Fatal("No se pudo iniciar el servidor después de 3 intentos")
	}

	stopProcess := func() {
		if cmd != nil && cmd.Process != nil {
			log.Println("Deteniendo servidor con PID:", cmd.Process.Pid)

			// Enviar SIGTERM
			// if err := cmd.Process.Kill(); err != nil {
			// 	log.Println("Error al detener el proceso:", err)
			// }

			// time.Sleep(1 * time.Second)

			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				log.Println("Error enviando SIGINT:", err)
				// // Si falla, intenta con Kill()
				// if err := cmd.Process.Kill(); err != nil {
				// 	log.Println("Error al matar proceso:", err)
				// }
			}

			// Esperar a que el proceso termine
			if err := cmd.Wait(); err != nil {
				if exiterr, ok := err.(*exec.ExitError); ok {
					log.Printf("Proceso terminado (código %d)\n", exiterr.ExitCode())
				} else {
					log.Println("Error esperando proceso:", err)
				}
			}

			log.Println("Servidor detenido.")
		}
	}

	startProcess()

	go func() {
		var (
			debounceTimer    *time.Timer
			debounceDuration = 1500 * time.Millisecond // Tiempo de debounce
			isRestarting     bool
		)

		for {
			<-restartCh

			if isRestarting {
				continue
			}

			isRestarting = true

			// Si ya hay un temporizador en marcha, detenerlo
			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			// Iniciar un nuevo temporizador
			debounceTimer = time.AfterFunc(debounceDuration, func() {
				stopProcess()
				time.Sleep(3 * time.Second) // Esperar liberación de puerto
				startProcess()
				isRestarting = false
			})
		}
	}()

	for {
		select {
		case event := <-watcher.Events:
			// Solo reiniciar en cambios de escritura o creación
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				log.Println("Cambios detectados en:", event.Name)
				select {
				case restartCh <- struct{}{}:
				default: // Evita bloqueo si el canal está lleno
				}
			}

		case err := <-watcher.Errors:
			log.Println("Error del watcher:", err)
		}
	}

}

func isPortAvailable(port string) bool {
	// Intentar abrir el puerto directamente
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}
