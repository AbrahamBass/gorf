package hotreload

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
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
		cmd = exec.Command("go", "run", mainFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			log.Println("Error al iniciar proceso:", err)
		}
	}

	stopProcess := func() {
		if cmd != nil && cmd.Process != nil {
			// Enviar SIGTERM y esperar
			cmd.Process.Signal(syscall.SIGTERM)
			cmd.Wait()
		}
	}

	startProcess()

	go func() {
		for {
			<-restartCh
			stopProcess()
			startProcess()
			time.Sleep(500 * time.Millisecond) // Tiempo de espera entre reinicios
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
