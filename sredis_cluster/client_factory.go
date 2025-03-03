package sredis_cluster

import (
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

type ClientFactory interface {
	GetRedisClusterClient() *redis.ClusterClient
}

type ClientFactoryImpl struct {
	redisClusterClient *redis.ClusterClient
}

type ClientFactoryOptions struct {
	URL     string
	Options *redis.ClusterOptions
}

var (
	clientInstance    ClientFactory
	clientFactoryLock sync.Mutex
)

func initializeRedisClusterClient(opt ClientFactoryOptions) error {
	clientFactoryLock.Lock()
	defer clientFactoryLock.Unlock()

	if clientInstance != nil {
		// client already initialized, skip...
		return nil
	}

	fmt.Println("MT TEST, REDIS CONNECTION URL: ", opt.URL)
	if len(opt.URL) > 0 {
		opt, err := redis.ParseClusterURL(opt.URL)
		if err != nil {
			return err
		}

		clientInstance = &ClientFactoryImpl{
			redisClusterClient: redis.NewClusterClient(opt),
		}

		return nil
	}

	clientInstance = &ClientFactoryImpl{
		redisClusterClient: redis.NewClusterClient(opt.Options),
	}

	return nil
}

func (s *ClientFactoryImpl) GetRedisClusterClient() *redis.ClusterClient {
	return s.redisClusterClient
}

func GetRedisClientFactory(redisConnectionURL string) (ClientFactory, error) {
	err := initializeRedisClusterClient(ClientFactoryOptions{URL: redisConnectionURL})

	return clientInstance, err
}
