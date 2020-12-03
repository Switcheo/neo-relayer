package service

import (
	neoRpc "github.com/joeqian10/neo-gogogo/rpc"
	"github.com/joeqian10/neo-gogogo/wallet"
	"github.com/polynetwork/neo-relayer/config"
	"github.com/polynetwork/neo-relayer/db"
	"github.com/polynetwork/neo-relayer/log"
	rsdk "github.com/polynetwork/poly-go-sdk"
	"os"
)

// SyncService ...
type SyncService struct {
	relayAccount    *rsdk.Account
	relaySdk        *rsdk.PolySdk
	relaySyncHeight uint32

	neoAccount       *wallet.Account
	neoRpcClients    []*neoRpc.RpcClient
	neoSyncHeight    uint32
	neoNextConsensus string

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
}

// GetFirstNeoRpcClient returns the first rpc client initialized
func (this SyncService) GetFirstNeoRpcClient() *neoRpc.RpcClient {
	return this.neoRpcClients[0]
}

func checkIfExist(dir string) bool {
	_, err := os.Stat(dir)
	if err != nil && !os.IsExist(err) {
		return false
	}
	return true
}
