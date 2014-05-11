package main

import (
	"code.google.com/p/gcfg"
	"database/sql"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/go-martini/martini"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
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

func getTuple(params martini.Params) string {
	return ""
}

func getRandom() string {
	return ""
}

func insertVal(valType string, val string) error {
	return nil
}

func start(context *cli.Context) {
	var cfg Config

	if err := gcfg.ReadFileInto(&cfg, CONFIG_FILE); err != nil {
		log.Fatalf("Could not open file %s: %v", CONFIG_FILE, err)
		return
	}

	db, err := sql.Open("sqlite3", cfg.Database.Location)
	if err != nil {
		log.Fatalf("Could not open database %s: %v", cfg.Database.Location, err)
	}
	defer db.Close() // Will never close lololol

	m := martini.Classic()

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
