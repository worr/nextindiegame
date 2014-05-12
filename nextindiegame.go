package main

import (
	"bytes"
	"code.google.com/p/gcfg"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/go-martini/martini"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
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
	var ret struct { Id int; Value string }

	rows, err := db.Query(fmt.Sprintf("select id, value from %s order by random() limit 1", table))
	if err != nil {
		return -1, "", err
	}

	rows.Next()
	if err = rows.Scan(&ret.Id, &ret.Value); err != nil {
		return -1, "", err
	}

	return ret.Id, ret.Value, nil
}

func getLink(genreId, emotionId, fantasyId int) string {
	return fmt.Sprintf("/game/?genre=%d&emotion=%d&fantasy=%d",
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

// Routes
// API routes
func randGame(db *sql.DB, templates map[string]*template.Template, params martini.Params) string {
	var game *Game
	var err error

	if game, err = NewRandomGame(db); err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Printf("%v\n", game)
	data, _ := json.Marshal(game)
	return bytes.NewBuffer(data).String()
}

// HTML routes
func index(db *sql.DB, templates map[string]*template.Template, params martini.Params) string {
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := templates["main.html"].Execute(buf, nil); err != nil {
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

	m := martini.Classic()

	// Load my templates and my db into my context for my route handlers
	m.Map(templates)
	m.Map(db)

	m.Get("/", index)
	m.Get("/api/game/", randGame)

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
