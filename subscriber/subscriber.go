package main

import (
	"encoding/json"
	"os"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

type User struct {
	Name  string
	Email string
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

func main() {
	Logger := logrus.New()
	Logger.SetFormatter(&logrus.JSONFormatter{})
	Logger.SetOutput(os.Stdout)

	rClient := redis.NewClient(&redis.Options{
		Addr:     "redis:6379", // We connect to host redis, thats what the hostname of the redis service is set to in the docker-compose
		Password: "redis",      // The password IF set in the redis Config file
		DB:       0,
	})

	var client = RedisClient{
		Redis:  rClient,
		Logger: Logger,
	}

	err := client.Redis.Ping().Err()
	if err != nil {
		panic(err)
	}

	// ctx := context.Background()
	topicName := "users"

	// suscribe to topic
	sub := client.Redis.Subscribe(topicName)
	if err != nil {
		panic(err)
	}

	defer sub.Close()
	//create chan
	channel := sub.Channel()
	for msg := range channel {
		user := &User{}
		err := user.UnmarshalBinary([]byte(msg.Payload))
		if err != nil {
			panic(err)
		}
		client.Logger.Info(msg)
	}
}
