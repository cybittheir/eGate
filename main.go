package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	server "github.com/emersion/go-smtp"

	"eGate/config"
	"eGate/proxy"
	"eGate/version"
)

func main() {

	saveLogs := flag.Bool("l", false, "save logs to files (Logs/YYYYMMDD.log)")
	silent := flag.Bool("s", false, "silent mode (no logs to console)")
	helpFlag := flag.Bool("h", false, "show help and exit")

	flag.Parse()

	// Текст подсказки (без логов, просто stdout)
	helpText := fmt.Sprintf(
		"%s\nПараметры запуска:\n  -h  вывод этой подсказки\n  -l  сохранять логи в файлы (папка Logs, имя по дате)\n  -s  тихий режим (не выводить логи в консоль)\n",
		version.String(),
	)

	if *helpFlag {
		fmt.Print(helpText)
		return
	}

	// Информация о программе и о вызове — всегда, без префикса времени
	fmt.Println(version.String())
	fmt.Println("-h - подсказки параметров запуска")
	if *saveLogs {
		fmt.Println("Режим: запись логов в файлы (Logs/YYYYMMDD.log)")
	}
	if *silent {
		fmt.Println("Режим: тихий (логи в консоль не выводятся)")
	}

	// Теперь настраиваем логирование
	logFile, err := setupLogging(*saveLogs, *silent)
	if err != nil {
		// здесь уже можно использовать log, но он может быть disacrd; на всякий случай продублируем
		fmt.Println("Ошибка инициализации логирования:", err)
		log.Fatal(err)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	wl, err := config.LoadWhitelist(cfg.WhitelistPath)
	if err != nil {
		log.Fatalf("Whitelist error: %v", err)
	}

	be := &proxy.Backend{Cfg: cfg, Whitelist: wl}

	// Background reloader
	go func() {
		for {
			time.Sleep(30 * time.Second)
			newWl, err := config.LoadWhitelist(cfg.WhitelistPath)
			if err != nil {
				log.Printf("Whitelist reload failed: %v", err)
				continue
			}
			be.UpdateWhitelist(newWl)
		}
	}()

	s := server.NewServer(be)
	s.Addr = cfg.Local.Addr
	s.Domain = cfg.Local.Domain
	s.AllowInsecureAuth = true
	s.Debug = os.Stdout

	log.Printf("SMTP Proxy started on %s", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func setupLogging(saveToFile, silent bool) (*os.File, error) {
	var writers []io.Writer

	if saveToFile {
		if err := os.MkdirAll("Logs", 0755); err != nil {
			return nil, fmt.Errorf("не удалось создать папку Logs: %w", err)
		}
		ts := time.Now().Format("20060102")
		path := filepath.Join("Logs", ts+".log")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("не удалось открыть лог-файл %s: %w", path, err)
		}

		if silent {
			log.SetOutput(f)
			return f, nil
		}

		writers = append(writers, f, os.Stdout)
		log.SetOutput(io.MultiWriter(writers...))
		return f, nil
	}

	if silent {
		log.SetOutput(io.Discard)
	} else {
		log.SetOutput(os.Stdout)
	}
	return nil, nil
}
