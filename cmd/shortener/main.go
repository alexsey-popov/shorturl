package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexsey-popov/shorturl/internal/api"
	"github.com/alexsey-popov/shorturl/internal/config"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	// Создаём логгер
	l, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("ошибка при создании логгера: %v", err)
	}
	defer l.Sync()
	sugar := l.Sugar()

	sugar.Infof("Build version: %s", buildVersion)
	sugar.Infof("Build date: %s", buildDate)
	sugar.Infof("Build commit: %s", buildCommit)

	// Парсим конфиг значениями из флагов и переменных окружения
	cfg, err := config.NewParsed()
	if err != nil {
		sugar.Fatalf("ошибка при пирсинге конфига: %v", err.Error())
	}

	// Создаём объект приложения (обвёртка над REST и grpcsrv эндпоинтами)
	app, err := api.NewFromConfig(cfg, sugar)
	if err != nil {
		sugar.Fatalf("ошибка при пирсинге конфига: %v", err)
	}

	// Канал для перехвата системных сигналов завершения
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Канал для ошибок запуска сервера
	srvErr := make(chan error, 1)

	// Запускаем сервер в отдельной горутине
	go func() {
		err := app.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
		}
	}()

	// Ожидаем сигнала или ошибки сервера для graceful shutdown
	select {
	case sig := <-stop:
		sugar.Infof("Получен сигнал остановки (%s), завершаем работу сервера...", sig)
	case err := <-srvErr:
		sugar.Errorf("ошибка в работе сервера: %v", err)
	}

	// Останавливаем сервис (соединение с бд и т.п.)
	if err := app.Close(); err != nil {
		sugar.Errorf("ошибка при остановке сервиса: %v", err)
	}

	sugar.Infoln("Сервер(REST) успешно остановлен")
}
