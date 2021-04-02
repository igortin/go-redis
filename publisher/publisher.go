package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type User struct {
	Name  string `json:"name"`
	Email string `json: "email"`
}

func (u *User) MarshalBinary() ([]byte, error) {
	b, err := json.Marshal(u)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func (u *User) UnmarshalBinary(b []byte) error {
	err := json.Unmarshal(b, u)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) String() string {
	return "User: " + u.Name + " registered with Email: " + u.Email
}

type RedisClient struct {
	Redis  *redis.Client
	Logger *logrus.Logger
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%+v", `{"status": ok}`)
}

func loggingMiddleware(next http.Handler, client *RedisClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		req := fmt.Sprintf("%s %s", r.Method, r.URL)

		client.Logger.Info(req)
		next.ServeHTTP(w, r)
		client.Logger.Info(req, "completed in", time.Since(start))
	})
}

func redisHandler(c *RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				err := r.(error)
				panic(err)
			}
		}()
		//Parse Query parameters good
		var user User
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		topic := "users"
		msg, err := user.MarshalBinary()
		if err != nil {
			panic(err)
		}

		err = c.Redis.Publish(topic, msg).Err()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Request successfully proceed: %v", user)
	}
}

func main() {
	Logger := logrus.New()
	Logger.SetFormatter(&logrus.JSONFormatter{})
	Logger.SetOutput(os.Stdout)

	Client := redis.NewClient(&redis.Options{
		Addr:     "redis:6379", // We connect to host redis, thats what the hostname of the redis service is set to in the docker-compose
		Password: "redis",
		DB:       0,
	})

	var client = RedisClient{
		Redis:  Client,
		Logger: Logger,
	}

	// Check connectivity to Redis
	err := client.Redis.Ping().Err()
	if err != nil {
		panic(err)
	}

	router := mux.NewRouter()

	handler := loggingMiddleware(router, &client)

	router.HandleFunc("/health", healthCheck).Methods("GET").Schemes("http")

	// Main logic to store k:v to redis
	router.HandleFunc("/", redisHandler(&client)).Methods("POST").Schemes("http")

	srv := &http.Server{
		Handler:      handler,
		Addr:         ":8081",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	client.Logger.Fatal(srv.ListenAndServe())
}
