package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
	"github.com/star-39/moe-sticker-bot/pkg/config"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

func Init() {
	initLogrus()
	b = initBot()
	initWorkspace(b)
	initWorkersPool()
	go initGoCron()
	if config.Config.WebApp {
		InitWebAppServer()
	} else {
		log.Info("WebApp not enabled.")
	}

	log.WithFields(log.Fields{"botName": botName, "dataDir": dataDir}).Info("Bot OK.")

	// complies to telebot v3.1
	b.Use(middleware.Recover())

	b.Handle("/quit", cmdQuit)
	b.Handle("/cancel", cmdQuit)
	b.Handle("/exit", cmdQuit)
	b.Handle("/help", cmdStart, checkState)
	b.Handle("/about", cmdAbout, checkState)
	b.Handle("/faq", cmdFAQ, checkState)
	b.Handle("/import", cmdImport, checkState)
	b.Handle("/download", cmdDownload, checkState)
	b.Handle("/create", cmdCreate, checkState)
	b.Handle("/manage", cmdManage, checkState)
	b.Handle("/search", cmdSearch, checkState)

	b.Handle("/register", cmdRegister, checkState)
	b.Handle("/statrep", cmdStatRep, checkState)

	b.Handle("/start", cmdStart, checkState)

	b.Handle(tele.OnText, handleMessage)
	b.Handle(tele.OnVideo, handleMessage)
	b.Handle(tele.OnAnimation, handleMessage)
	b.Handle(tele.OnSticker, handleMessage)
	b.Handle(tele.OnDocument, handleMessage)
	b.Handle(tele.OnPhoto, handleMessage)
	b.Handle(tele.OnCallback, handleMessage, autoRespond, sanitizeCallback)

	b.Start()
}

// This one never say goodbye.
func endSession(c tele.Context) {
	cleanUserDataAndDir(c.Sender().ID)
}

// This one will say goodbye.
func terminateSession(c tele.Context) {
	cleanUserDataAndDir(c.Sender().ID)
	c.Send("Bye. /start")
}

func endManageSession(c tele.Context) {
	ud, exist := users.data[c.Sender().ID]
	if !exist {
		return
	}
	if ud.stickerData.id == "" {
		return
	}
	path := filepath.Join(config.Config.WebappDataDir, ud.stickerData.id)
	os.RemoveAll(path)
}

func onError(err error, c tele.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorln("Recovered from onError!!", r)
		}
	}()
	sendFatalError(err, c)
	cleanUserDataAndDir(c.Sender().ID)
}

func initBot() *tele.Bot {
	pref := tele.Settings{
		Token:       config.Config.BotToken,
		Poller:      &tele.LongPoller{Timeout: 10 * time.Second},
		Synchronous: false,
		// Genrally, issues are tackled inside each state, only fatal error should be returned to framework.
		// onError will terminate current session and log to terminal.
		OnError: onError,
	}
	log.WithField("token", config.Config.BotToken).Info("Attempting to initialize...")
	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func initWorkspace(b *tele.Bot) {
	botName = b.Me.Username
	dataDir = botName + "_data"
	users = Users{data: make(map[int64]*UserData)}
	downloadQueue = DownloadQueue{ss: make(map[string]bool)}
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	if config.Config.UseDB {
		dbName := botName + "_db"
		err = initDB(dbName)
		if err != nil {
			log.Fatalln("Error initializing database!!", err)
		}
	} else {
		log.Warn("Not using database because --use_db is not set.")
	}

	if runtime.GOOS == "linux" {
		BSDTAR_BIN = "bsdtar"
		CONVERT_BIN = "convert"
	} else {
		BSDTAR_BIN = "tar"
		CONVERT_BIN = "magick"
		CONVERT_ARGS = []string{"convert"}
	}
}

func initGoCron() {
	time.Sleep(15 * time.Second)
	cronScheduler = gocron.NewScheduler(time.UTC)
	cronScheduler.Every(2).Days().Do(purgeOutdatedUserData)
	cronScheduler.Every(1).Weeks().Do(curateDatabase)
	cronScheduler.StartAsync()
}

func initLogrus() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:            true,
		DisableLevelTruncation: true,
	})

	level := strings.ToUpper(config.Config.LogLevel)
	switch level {
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARNING":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.TraceLevel)
	}
	log.Debug("Warning: Log level below DEBUG might print sensitive information, including passwords.")
}
