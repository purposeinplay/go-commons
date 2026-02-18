package leaderelection

import (
	"log"
	"testing"

	"github.com/purposeinplay/go-commons/psqldocker"
	"github.com/purposeinplay/go-commons/psqltest"
	"github.com/purposeinplay/go-commons/psqlutil"
)

var (
	schema  string
	TestDSN string
)

func TestMain(m *testing.M) {
	const (
		user          = "test"
		password      = "pass"
		dbName        = "test"
		containerName = "win_quests_discord_tests"
	)

	sch, err := psqlutil.ReadSchema()
	if err != nil {
		log.Println("err while reading schema:", err)
	}

	schema = sch

	psqlContainer := psqldocker.NewContainer(
		user,
		password,
		dbName,
		psqldocker.WithImageTag("alpine"),
		psqldocker.WithContainerName(containerName),
		psqldocker.WithExpiration(20),
	)

	err = psqlContainer.Start()
	if err != nil {
		log.Println("err while starting container:", err)
	}

	defer func() {
		if err := psqlContainer.Close(); err != nil {
			log.Println("err while closing container:", err)
		}
	}()

	TestDSN = psqlutil.ConnectionConfig{
		Host:     "localhost",
		Port:     psqlContainer.Port(),
		User:     user,
		Password: password,
		DBName:   dbName,
		SSLMode:  "disable",
	}.DSN()

	psqltest.Register(TestDSN)

	m.Run()
}
