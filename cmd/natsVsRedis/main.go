package main

import (
	"context"
	"fmt"
	"github.com/mmtretiak/natsVsRedis/pkg/nats"
	"github.com/mmtretiak/natsVsRedis/pkg/redis"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

var (
	operationsCount int
	valueLength     int
	numberOfClients int
	rateLimit       int
	timeout         time.Duration

	redisConnectionUrl     = "redis://:@redis-cluster-test-headless.data.svc.cluster.local:6379"
	natsConnectionUrl      = "nats://nats-cluster-test-0.nats-cluster-test-headless.data.svc.cluster.local:4222,nats://nats-cluster-test-1.nats-cluster-test-headless.data.svc.cluster.local:4222,nats://nats-cluster-test-2.nats-cluster-test-headless.data.svc.cluster.local:4222"
	dragonFlyConnectionUrl = "redis://:@dragonfly-test.data.svc.cluster.local:6379"
	natsBucketName         = "perfomance_test_3"
)

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type KVClient interface {
	Set(ctx context.Context, key, value string) error
	Get(ctx context.Context, key string) (string, error)
	FlushAll(ctx context.Context) error
	DeleteByPrefix(ctx context.Context, prefix string) error
}

func parseEnv() {
	var err error

	operationsCount64, err := strconv.ParseInt(os.Getenv("OPERATIONS_COUNT"), 10, 64)
	if err != nil {
		panic(err)
	}
	operationsCount = int(operationsCount64)

	rateLimit64, err := strconv.ParseInt(os.Getenv("RATE_LIMIT"), 10, 64)
	if err != nil {
		panic(err)
	}
	rateLimit = int(rateLimit64)

	numberOfClients64, err := strconv.ParseInt(os.Getenv("CLIENT_COUNT"), 10, 64)
	if err != nil {
		panic(err)
	}
	numberOfClients = int(numberOfClients64)

	valueLength64, err := strconv.ParseInt(os.Getenv("VALUE_LENGTH"), 10, 64)
	if err != nil {
		panic(err)
	}
	valueLength = int(valueLength64)

	timeout = time.Second / time.Duration(rateLimit)
}

func main() {
	parseEnv()

	testKv(numberOfClients, createRedisClient, "Redis")
	testKv(numberOfClients, createNatsClient, "NatsClient")
	testKv(numberOfClients, createDragonflyClient, "Dragonfly")

	for {
		time.Sleep(5 * time.Second)
	}
}

func testKv(numberOfClients int, newKv func() (KVClient, error), kvType string) {
	wg := &sync.WaitGroup{}

	setOpsPerSecondChan := make(chan float64, numberOfClients)
	getOpsPerSecondChan := make(chan float64, numberOfClients)

	for i := 0; i < numberOfClients; i++ {
		kvClient, err := newKv()
		if err != nil {
			panic(err)
		}

		kvTester := kvTester{
			kvClient:  kvClient,
			kvType:    kvType,
			keyPrefix: randStringRunes(10),
		}

		wg.Add(1)
		go func() {
			kvTester.setAndGetBench(operationsCount, valueLength, setOpsPerSecondChan, getOpsPerSecondChan)
			wg.Done()
		}()
	}

	var setOps float64 = 0
	var getOps float64 = 0

	for i := 0; i < numberOfClients; i++ {
		setOps += <-setOpsPerSecondChan
	}

	for i := 0; i < numberOfClients; i++ {
		getOps += <-getOpsPerSecondChan
	}

	fmt.Println("Average set ops per second: ", setOps/float64(numberOfClients))
	fmt.Println("Average get ops per second: ", getOps/float64(numberOfClients))

	wg.Wait()

}

func (k kvTester) setAndGetBench(operationsCount, valueLength int, setOpsPerSecondChan chan float64, getOpsPerSecondChan chan float64) {
	err := k.setNTimes(operationsCount, valueLength, setOpsPerSecondChan)
	if err != nil {
		panic(err)
	}

	err = k.getNTimes(operationsCount, valueLength, getOpsPerSecondChan)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 15)
	err = k.kvClient.DeleteByPrefix(context.Background(), k.keyPrefix)
	if err != nil {
		panic(err)
	}
}

type kvTester struct {
	kvClient  KVClient
	kvType    string
	keyPrefix string
}

func (k *kvTester) setNTimes(setCount int, stringLength int, setOpsPerSecondChan chan<- float64) error {
	randString := randStringRunes(stringLength)

	startTime := time.Now()

	//ticker := time.NewTicker(timeout)
	//defer ticker.Stop()

	for i := 0; i < setCount; i++ {
		if err := k.kvClient.Set(context.Background(), fmt.Sprintf("%s-key-%d", k.keyPrefix, i), randString); err != nil {
			return err
		}
		//<-ticker.C
	}

	timeTaken := time.Since(startTime)
	totalTimeInSeconds := timeTaken.Seconds()
	totalTimeInMicro := timeTaken.Nanoseconds()

	opsPerSecond := float64(setCount) / totalTimeInSeconds
	takenPerOperation := time.Duration(float64(totalTimeInMicro) / float64(setCount))
	setOpsPerSecondChan <- opsPerSecond

	fmt.Printf("kvType: %v, setCount: %v, stringLength: %v, timeTaken: %v, rateLimit: %v, timeoutTime: %v, ops: %v, timeTakenPerOperation: %v\n", k.kvType, setCount, stringLength, timeTaken, rateLimit, timeout, opsPerSecond, takenPerOperation)
	return nil
}

func (k *kvTester) getNTimes(getCount int, stringLength int, getOpsPerSecondChan chan<- float64) error {
	startTime := time.Now()

	//ticker := time.NewTicker(timeout)
	//defer ticker.Stop()

	for i := 0; i < getCount; i++ {
		if _, err := k.kvClient.Get(context.Background(), fmt.Sprintf("%s-key-%d", k.keyPrefix, i)); err != nil {
			return err
		}
		//<-ticker.C
	}

	timeTaken := time.Since(startTime)
	totalTimeInSeconds := timeTaken.Seconds()
	totalTimeInMicro := timeTaken.Nanoseconds()

	opsPerSecond := float64(getCount) / totalTimeInSeconds
	takenPerOperation := time.Duration(float64(totalTimeInMicro) / float64(getCount))
	getOpsPerSecondChan <- opsPerSecond

	fmt.Printf("kvType: %v, getCount: %v, stringLength: %v, timeTaken: %v, rateLimit: %v, timeoutTime: %v, ops: %v, timeTakenPerOperation: %v\n", k.kvType, getCount, stringLength, timeTaken, rateLimit, timeout, opsPerSecond, takenPerOperation)
	return nil
}

func createRedisClient() (KVClient, error) {
	return redis.NewClusterClient(redisConnectionUrl)
}

func createDragonflyClient() (KVClient, error) {
	return redis.NewClient(dragonFlyConnectionUrl)
}

func createNatsClient() (KVClient, error) {
	return nats.NewClient(natsConnectionUrl, natsBucketName)
}
