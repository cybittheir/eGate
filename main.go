package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	server "github.com/emersion/go-smtp"

	"eGate/config"
	"eGate/internal/logging"
	"eGate/proxy"
	"eGate/version"
)

func main() {

	saveLogs := flag.Bool("l", false, "save logs to files (Logs/YYYYMMDD.log)")
	silent := flag.Bool("s", false, "silent mode (no logs to console)")
	debug := flag.Bool("d", false, "debug mode (print SMTP logs to console)")
	helpFlag := flag.Bool("h", false, "show help and exit")
	verFlag := flag.Bool("v", false, "show version and exit")
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Текст подсказки (без логов, просто stdout)
	helpText := fmt.Sprintf(
		"%s\nПараметры запуска:\n  -h  вывод этой подсказки\n  -l  сохранять логи в файлы (папка Logs, имя по дате)\n  -d выводить протокол SMTP на экран\n  -s  тихий режим (не выводить логи в консоль)\n",
		version.String(),
	)

	if *helpFlag {
		fmt.Print(helpText)
		return
	}

	if *verFlag {
		fmt.Print(version.String())
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
	logFile, err := logging.Setup(*saveLogs, *silent)
	if err != nil {
		// здесь уже можно использовать log, но он может быть disacrd; на всякий случай продублируем
		fmt.Println("Ошибка инициализации логирования:", err)
		log.Fatal(err)
	}
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Printf("Config error: %v", err)
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

	if logFile != nil && *debug {
		if w, ok := logFile.(io.Writer); ok {
			s.Debug = w
		} else {
			s.Debug = nil
		}

	} else {
		// например, в stdout или вообще отключить
		// s.Debug = os.Stdout
		if !*silent && *debug {
			s.Debug = os.Stdout
		} else {
			s.Debug = nil
		}
	}

	log.Printf("SMTP Proxy started on %s", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
