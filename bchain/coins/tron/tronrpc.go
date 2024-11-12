package tron

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/juju/errors"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/eth"
)

type Network uint32

const (
	// MainNet is production network
	MainNet     eth.Network = 11111
	TestNetNile Network     = 201910292
)

type TronRPC struct {
	*eth.EthereumRPC
	Parser *TronParser // Vlastní pole Parser s typem *TronParser
}

func NewTronRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	c, err := eth.NewEthereumRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	var cfg eth.Configuration
	err = json.Unmarshal(config, &cfg)
	if err != nil {
		return nil, errors.Annotatef(err, "Invalid configuration file")
	}

	s := &TronRPC{
		EthereumRPC: c.(*eth.EthereumRPC),
		Parser:      NewTronParser(cfg.BlockAddressesToKeep, cfg.AddressAliases),
	}

	return s, nil
}

// OpenRPC opens an RPC connection to the Tron backend
var OpenRPC = func(url string) (bchain.EVMRPCClient, bchain.EVMClient, error) {
	opts := []rpc.ClientOption{}
	opts = append(opts, rpc.WithWebsocketMessageSizeLimit(0))

	r, err := rpc.DialOptions(context.Background(), url, opts...)
	if err != nil {
		return nil, nil, err
	}

	rpcClient := &TronRPCClient{Client: r}
	ethClient := ethclient.NewClient(r) // Ethereum client for compatibility
	tc := &TronClient{
		Client:    ethClient,
		rpcClient: rpcClient,
	}

	return rpcClient, tc, nil
}

// Initialize Tron RPC
func (b *TronRPC) Initialize() error {
	b.OpenRPC = OpenRPC

	rc, ec, err := b.OpenRPC(b.ChainConfig.RPCURL)
	if err != nil {
		return err
	}

	b.Client = ec
	b.RPC = rc
	b.MainNetChainID = MainNet

	b.NewBlock = &TronNewBlock{channel: make(chan *types.Header)}
	b.NewTx = &TronNewTx{channel: make(chan common.Hash)}

	b.Testnet = false
	b.Network = "tron-mainnet"

	log.Info("TronRPC: initialized Tron blockchain: ", b.Network)
	return nil
}

// GetChainParser returns Tron-specific BlockChainParser
func (b *TronRPC) GetChainParser() bchain.BlockChainParser {
	return b.Parser
}
