package main

import (
	"bytes"
	"code.google.com/p/gcfg"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/go-martini/martini"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"path"
	"strconv"
)

const GENRE_TABLE = "genre"
const EMOTION_TABLE = "emotion"
const FANTASY_TABLE = "fantasy"

const CONFIG_FILE = "nextindiegame.ini"

type Config struct {
	Server struct {
		Hostname  string
		Port      string
		Templates string
	}

	Database struct {
		Location string
	}
}

type Game struct {
	Genre   string
	Emotion string
	Fantasy string
	Link    string
}

type ApplicationError struct {
	Error string
}

// A helper function to fetch a random val from a table
func getRandomVal(db *sql.DB, table string) (int, string, error) {
	var ret struct {
		Id    int
		Value string
	}

	row := db.QueryRow(fmt.Sprintf("select id, value from %s order by random() limit 1", table))
	if err := row.Scan(&ret.Id, &ret.Value); err != nil {
		return -1, "", err
	}

	return ret.Id, ret.Value, nil
}

func getVal(db *sql.DB, table string, id int) (string, error) {
	var ret string

	row := db.QueryRow(fmt.Sprintf("select value from %s where id = ?", table), id)
	if err := row.Scan(&ret); err != nil {
		return "", err
	}

	return ret, nil
}

func getLink(genreId, emotionId, fantasyId int) string {
	return fmt.Sprintf("/l/%02x%02x%02x",
		genreId,
		emotionId,
		fantasyId)
}

func NewRandomGame(db *sql.DB) (*Game, error) {
	var genre, emotion, fantasy string
	var genreId, emotionId, fantasyId int
	var err error

	if genreId, genre, err = getRandomVal(db, GENRE_TABLE); err != nil {
		return nil, err
	}

	if emotionId, emotion, err = getRandomVal(db, EMOTION_TABLE); err != nil {
		return nil, err
	}

	if fantasyId, fantasy, err = getRandomVal(db, FANTASY_TABLE); err != nil {
		return nil, err
	}

	return &Game{genre, emotion, fantasy, getLink(genreId, emotionId, fantasyId)}, nil
}

func NewLinkGame(db *sql.DB, link string) (*Game, error) {
	var genreId, emotionId, fantasyId int64
	var genre, emotion, fantasy string
	var err error

	// Link UNACCEPTABLE
	if len(link) != 6 {
		return nil, errors.New("UNACCEPTABLE")
	}

	if genreId, err = strconv.ParseInt(link[0:2], 16, 0); err != nil {
		return nil, err
	}

	if emotionId, err = strconv.ParseInt(link[2:4], 16, 0); err != nil {
		return nil, err
	}

	if fantasyId, err = strconv.ParseInt(link[4:6], 16, 0); err != nil {
		return nil, err
	}

	if genre, err = getVal(db, "genre", int(genreId)); err != nil {
		return nil, err
	}

	if emotion, err = getVal(db, "emotion", int(emotionId)); err != nil {
		return nil, err
	}

	if fantasy, err = getVal(db, "fantasy", int(fantasyId)); err != nil {
		return nil, err
	}

	return &Game{genre, emotion, fantasy, fmt.Sprintf("/l/%s", link)}, nil
}

func logError(logger *syslog.Writer, err error) {
	msg := fmt.Sprintf("%v", err)
	logger.Emerg(msg)
	log.Print(msg)
}

// Routes
// API routes
func randGame(db *sql.DB, templates map[string]*template.Template, logger *syslog.Writer, params martini.Params) string {
	var game *Game
	var err error

	if game, err = NewRandomGame(db); err != nil {
		logError(logger, err)
		return fmt.Sprintf("error getting game data")
	}
	data, _ := json.Marshal(game)
	return bytes.NewBuffer(data).String()
}

// HTML routes
func index(db *sql.DB, templates map[string]*template.Template, logger *syslog.Writer, params martini.Params) string {
	buf := bytes.NewBuffer(make([]byte, 0))
	var game *Game
	var err error

	for k, v := range params {
		if k == "link" {
			if game, err = NewLinkGame(db, v); err != nil {
				logError(logger, err)
				return fmt.Sprintf("error getting game data")
			}
		}
	}

	if game == nil {
		if game, err = NewRandomGame(db); err != nil {
			logError(logger, err)
			return fmt.Sprintf("error getting game data")
		}
	}

	if err = templates["main.html"].Execute(buf, game); err != nil {
		logError(logger, err)
		return fmt.Sprintf("error getting game data")
	}

	return buf.String()
}

// Add a template that inherits from the base template (everything)
func addTemplate(cfg *Config, templates map[string]*template.Template, name string) error {
	var err error

	kPath := path.Join(cfg.Server.Templates, name)
	bPath := path.Join(cfg.Server.Templates, "base.html")
	if _, err = os.Stat(kPath); os.IsNotExist(err) {
		return err
	}

	if _, err = os.Stat(bPath); os.IsNotExist(err) {
		return err
	}

	templates[name], err = template.New("main").ParseFiles(kPath, bPath)
	return err
}

func start(context *cli.Context) {
	var cfg Config
	templates := make(map[string]*template.Template)

	if err := gcfg.ReadFileInto(&cfg, CONFIG_FILE); err != nil {
		log.Fatalf("Could not open file %s: %v", CONFIG_FILE, err)
		return
	}

	db, err := sql.Open("sqlite3", cfg.Database.Location)
	if err != nil {
		log.Fatalf("Could not open database %s: %v", cfg.Database.Location, err)
	}
	defer db.Close() // Will never close lololol

	if err = addTemplate(&cfg, templates, "main.html"); err != nil {
		log.Fatalf("Could not compile templates: %v", err)
	}

	var logger *syslog.Writer
	if logger, err = syslog.New(syslog.LOG_LOCAL0|syslog.LOG_INFO, "nextindiega.me"); err != nil {
		log.Fatalf("Could not initialize logger: %v", err)
	}

	m := martini.Classic()

	// Load my templates and my db into my context for my route handlers
	m.Map(templates)
	m.Map(db)
	m.Map(logger)

	m.Get("/", index)
	m.Get("/l/:link", index)
	m.Get("/api/game/", randGame)

	m.Use(martini.Static("static"))

	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", cfg.Server.Hostname, cfg.Server.Port), m))
}

func main() {
	app := cli.NewApp()
	app.Name = "nextindiegame"
	app.Usage = "Randomly generate terrible indie game titles"
	app.Flags = []cli.Flag{
		cli.StringFlag{"config, c", CONFIG_FILE, "Location of config file"},
	}

	app.Action = start

	app.Run(os.Args)
}
