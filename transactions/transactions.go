package transactions

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/IBM/sarama"
	"github.com/nats-io/nats.go"

	L "userTransactions/logging"
)

type Service struct {
	mux           *http.ServeMux
	repo          Repo
	nc            *nats.Conn
	groupConsumer sarama.ConsumerGroup
}

// Init configures and initializes the service
func Init(dbHost, dbPort, dbUser, dbPassword, dbName, subject, broker, group, topic string) *Service {
	L.Logger.Info("Service is being initialized")

	var srv Service

	// initialize REST service
	srv.muxSetup()

	// initialiye DB connection
	srv.dbSetup(dbHost, dbPort, dbUser, dbPassword, dbName)

	// initialize NATS connection
	srv.natsSetup(subject)

	// initialize kafka connection
	srv.kafkaSetup(broker, group, topic)

	L.Logger.Info("Service initialized")
	return &srv
}

func (srv *Service) muxSetup() {
	mux := http.NewServeMux()

	mux.Handle("/", http.NotFoundHandler())
	mux.Handle("/deposit", http.HandlerFunc(srv.depositHandler))
	mux.Handle("/transfer", http.HandlerFunc(srv.transferHnadler))

	srv.mux = mux
}

func (srv *Service) dbSetup(dbHost, dbPort, dbUser, dbPassword, dbName string) {
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		L.Logger.Fatalf("Failed initializin DB connection. Connection string: %s. Error: %s", connectionString, err.Error())
	}
	srv.repo.db = db

	err = srv.repo.db.Ping()
	if err != nil {
		L.Logger.Fatalf("Failed to ping the DB. Connection string: %s. Error: %s", connectionString, err.Error())
	} else {
		L.Logger.Info("DB connection established")
	}
}

func (srv *Service) natsSetup(subject string) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		L.Logger.Fatal("Failed connecting to NATS: ", err)
	}

	srv.nc = nc

	nc.Subscribe(subject, srv.getBalanceNATS)
}

func (srv *Service) getBalanceNATS(msg *nats.Msg) {
	idstr := strings.Split(msg.Subject, ".")[2] // subject: get.balance.ID

	id, err := strconv.Atoi(idstr)
	if err != nil {
		L.Logger.Error("Failed to convert id to integer: ", err)
		// TODO: find a way to return the error
		return
	}

	b := srv.repo.getBalance(id)
	balance := BalanceNATSResponse{Balance: b}

	resp, err := json.Marshal(balance)
	if err != nil {
		L.Logger.Error("Failed to marshal the response: ", err)
		// TODO: find a way to return the error
		return
	}

	msg.Respond(resp)
}

// Run is used to start service
func (srv *Service) Run(port string) {
	L.Logger.Info("Running the service")
	go func() {
		err := http.ListenAndServe(port, srv.mux)
		if err != nil {
			L.Logger.Fatalf("failed startign the service on port %s :%s", port, err.Error())
		} else {
			L.Logger.Infof("The service is up and running on port %s", port)
		}
	}()
}

// Stop is stopping the service and closes all connections
func (srv *Service) Stop() {
	L.Logger.Info("Closing DB connection...")
	srv.repo.db.Close()

	L.Logger.Info("Closing NATS connection...")
	srv.nc.Close()

	L.Logger.Info("Closing Kafka consumer...")
	if err := srv.groupConsumer.Close(); err != nil {
		L.Logger.Error("Error closing partition consumer:", err)
	}
}
