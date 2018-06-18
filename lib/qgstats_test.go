package qgstats

import (
	// "database/sql"
	"github.com/achiku/qg"
	"log"
	"os"
	"testing"
)

var plugin QgPlugin

func init() {
	plugin.Host = os.Getenv("PGHOST")
	plugin.Port = os.Getenv("PGPORT")
	plugin.User = os.Getenv("PGUSER")
	plugin.Database = os.Getenv("PGDATABASE")
	plugin.Password = os.Getenv("PGPASSWORD")
	plugin.SSLMode = os.Getenv("PGSSLMODE")
	plugin.SSLKey = os.Getenv("PGSSLMODE")
	plugin.SSLCert = os.Getenv("PGSSLCERT")
	plugin.SSLRootCert = os.Getenv("PGSSLROOTCERT")
}

func TestMain(m *testing.M) {
	os.Exit(func() (status int) {
		var err error
		db, err := plugin.DB()
		if err != nil {
			log.Fatal("failed to connect to database")
			status = 1
			return

		}
		defer func() {
			_, err := db.Exec("TRUNCATE TABLE que_jobs")
			if err != nil {
				panic(err)
			}
			db.Close()
		}()
		status = m.Run()
		return
	}())
}

func TestIt(t *testing.T) {
	db, err := plugin.DB()
	if err != nil {
		t.Fatal(err)
	}
	client, err := qg.NewClient2(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(client.Enqueue(&qg.Job{Queue: "Q", Type: "test"}))

	var i int
	db.QueryRow("SELECT COUNT(*) FROM que_jobs").Scan(&i)
	t.Log(i)

	metrics, err := plugin.FetchMetrics()
	if err != nil {
		t.Fatal(err)
	}

	if 1 != metrics["count_total"].(int) {
		t.Logf(`1 != metrics["count_total"].(int) (got %v)`, metrics["count_total"].(int))
		t.Fail()
	}
}
