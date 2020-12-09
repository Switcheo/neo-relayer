package service

import (
	neoRpc "github.com/joeqian10/neo-gogogo/rpc"
	"github.com/joeqian10/neo-gogogo/wallet"
	"github.com/polynetwork/neo-relayer/config"
	"github.com/polynetwork/neo-relayer/db"
	"github.com/polynetwork/neo-relayer/log"
	rsdk "github.com/polynetwork/poly-go-sdk"
	"math/rand"
	"os"
	"time"
)

// SyncService ...
type SyncService struct {
	relayAccount    *rsdk.Account
	relaySdk        *rsdk.PolySdk
	relaySyncHeight uint32

	neoAccount          *wallet.Account
	neoRpcClients       []*neoRpc.RpcClient
	lastLivelyRpcClient *neoRpc.RpcClient
	neoSyncHeight       uint32
	neoNextConsensus    string

	db     *db.BoltDB
	config *config.Config
}

// NewSyncService ...
func NewSyncService(acct *rsdk.Account, relaySdk *rsdk.PolySdk, neoAccount *wallet.Account, neoRpcClients []*neoRpc.RpcClient) *SyncService {
	if !checkIfExist(config.DefConfig.DBPath) {
		os.Mkdir(config.DefConfig.DBPath, os.ModePerm)
	}
	boltDB, err := db.NewBoltDB(config.DefConfig.DBPath)
	if err != nil {
		log.Errorf("db.NewWaitingDB error:%s", err)
		os.Exit(1)
	}
	syncSvr := &SyncService{
		relayAccount: acct,
		relaySdk:     relaySdk,

		neoAccount:    neoAccount,
		neoRpcClients: neoRpcClients,
		db:            boltDB,
		config:        config.DefConfig,
	}
	return syncSvr
}

// Run ...
func (this *SyncService) Run() {
	go this.RelayToNeo()
	go this.RelayToNeoRetry()
	go this.NeoToRelay()
	go this.NeoToRelayCheckAndRetry()
	go this.SyncLivelyNeoClient()
}

// SyncLivelyNeoClient go routine to try to find and set the last active neo client
func (this *SyncService) SyncLivelyNeoClient() {
	// relaySyncHeight

	// Check that the current lively client is still lively
	currentClient := this.GetLivelyNeoRpcClient()
	res := currentClient.GetBlockCount()

	// For debugging:
	//log.Info("[SetLivelyNeoClient] currentClient: ", currentClient.Endpoint.String())
	//log.Info("[SetLivelyNeoClient] blockHeight: ", res.Result)
	//log.Info("[SetLivelyNeoClient] relaySyncHeight: ", this.relaySyncHeight)

	// if there's an error, try to find a new active client
	if res.HasError() || res.Result < int(this.relaySyncHeight) {
		log.Info("[SetLivelyNeoClient] Endpoint is no longer lively: ", currentClient.Endpoint.String())
		log.Info("[SetLivelyNeoClient] Current relay sync height: ", this.relaySyncHeight)
		for _, client := range this.neoRpcClients {
			response := client.GetBlockCount()
			if !response.HasError() && response.Result >= int(this.relaySyncHeight) {
				log.Info("[SetLivelyNeoClient] Found a new reliable endpoint: ", client.Endpoint.String())
				this.lastLivelyRpcClient = client
				break
			}
		}
	}

	// Check every 2 seconds
	time.Sleep(time.Duration(this.config.ScanInterval) * time.Second)
	go this.SyncLivelyNeoClient()
}

// GetLivelyNeoRpcClient returns the first Neo rpc client that is lively or the first one in the list
func (this SyncService) GetLivelyNeoRpcClient() *neoRpc.RpcClient {
	if this.lastLivelyRpcClient != nil {
		return this.lastLivelyRpcClient
	}
	return this.neoRpcClients[0]
}

// GetRandomNeoRpcClients returns a set of random neo rpc clients
func (this SyncService) GetRandomNeoRpcClients(max int) []*neoRpc.RpcClient {
	// create a copy of the slice header
	c := this.neoRpcClients

	n := min(max, len(this.neoRpcClients))
	samples := make([]*neoRpc.RpcClient, n)

	for i := 0; i < n; i++ {
		length := len(c)
		r := rand.Intn(length-1)
		samples[i] = c[r]

		// remove the sample from the copy slice
		c[r], c[length-1] = c[length-1], c[r]
		c = c[:length-1]
	}
	return samples
}

func checkIfExist(dir string) bool {
	_, err := os.Stat(dir)
	if err != nil && !os.IsExist(err) {
		return false
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}