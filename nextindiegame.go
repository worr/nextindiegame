package main

import (
	"bytes"
	"code.google.com/p/gcfg"
	"database/sql"
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

type GameOutput struct {
	Genre   string
	Emotion string
	Fantasy string
	Link    string
}

// A helper function to fetch a random val from a table
func getRandomVal(db *sql.DB, table string) (string, error) {
	var ret string

	rows, err := db.Query(fmt.Sprintf("select value from %s order by random() limit 1", table))
	if err != nil {
		return "", err
	}

	rows.Next()
	if err = rows.Scan(&ret); err != nil {
		return "", err
	}

	return ret, nil
}

// The main route for the site
func getRandom(db *sql.DB, templates map[string]*template.Template, params martini.Params) string {
	var genre, emotion, fantasy string
	var err error

	if genre, err = getRandomVal(db, GENRE_TABLE); err != nil {
		return fmt.Sprintf("%v", err)
	}

	if emotion, err = getRandomVal(db, EMOTION_TABLE); err != nil {
		return fmt.Sprintf("%v", err)
	}

	if fantasy, err = getRandomVal(db, FANTASY_TABLE); err != nil {
		return fmt.Sprintf("%v", err)
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	if err = templates["main.html"].Execute(buf, &GameOutput{genre, emotion, fantasy, ""}); err != nil {
		return fmt.Sprintf("%v", err)
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

	templates[name], err = template.New(name).ParseFiles(bPath, kPath)
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

	m.Get("/", getRandom)

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
