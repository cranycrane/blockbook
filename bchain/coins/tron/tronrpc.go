package tron

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/juju/errors"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/eth"
	"math/big"
)

// todo look at GetContractInfo
// todo look at EthereumTypeGetErc20ContractBalance
// todo look at GetTokenURI

const (
	// MainNet is production network
	MainNet     eth.Network = 11111
	TestNetNile eth.Network = 201910292

	TRC10TokenType   bchain.TokenTypeName = "TRC10"
	TRC20TokenType   bchain.TokenTypeName = "TRC20"
	TRC721TokenType  bchain.TokenTypeName = "TRC721"
	TRC1155TokenType bchain.TokenTypeName = "TRC1155"
)

type TronConfiguration struct {
	eth.Configuration
	MessageQueueBinding string `json:"message_queue_binding"`
}

type TronRPC struct {
	*eth.EthereumRPC
	Parser      *TronParser
	ChainConfig *TronConfiguration
	mq          *bchain.MQ
}

func NewTronRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	c, err := eth.NewEthereumRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	var cfg TronConfiguration
	err = json.Unmarshal(config, &cfg)
	if err != nil {
		return nil, errors.Annotatef(err, "Invalid Tron configuration file")
	}

	bchain.EthereumTokenTypeMap = []bchain.TokenTypeName{TRC20TokenType, TRC721TokenType, TRC1155TokenType}

	s := &TronRPC{
		EthereumRPC: c.(*eth.EthereumRPC),
		Parser:      NewTronParser(cfg.BlockAddressesToKeep, cfg.AddressAliases),
	}

	s.EthereumRPC.Parser = s.Parser
	s.ChainConfig = &cfg
	s.PushHandler = pushHandler

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

	ctx, cancel := context.WithTimeout(context.Background(), b.Timeout)
	defer cancel()

	id, err := b.Client.NetworkID(ctx)
	if err != nil {
		return err
	}

	// parameters for getInfo request
	switch eth.Network(id.Uint64()) {
	case MainNet:
		b.Testnet = false
		b.Network = "mainnet"
	case TestNetNile:
		b.Testnet = true
		b.Network = "nile"
	default:
		return errors.Errorf("Unknown network id %v", id)
	}

	log.Info("TronRPC: initialized Tron blockchain: ", b.Network)
	return nil
}

// GetBestBlockHash returns hash of the tip of the best-block-chain
// need to overwrite this because the getBestHeader method in EthRpc is
// relying on the subscription
// known bug: the networkId does not get updated, because it is also
// relying on the getBestHeader method.
func (b *TronRPC) GetBestBlockHash() (string, error) {
	var err error
	var header bchain.EVMHeader

	header, err = b.getBestHeader()
	if err != nil {
		return "", err
	}

	return header.Hash(), nil
}

// GetBestBlockHeight returns height of the tip of the best-block-chain
func (b *TronRPC) GetBestBlockHeight() (uint32, error) {
	var err error
	var header bchain.EVMHeader

	header, err = b.getBestHeader()
	if err != nil {
		return 0, err
	}

	return uint32(header.Number().Uint64()), nil
}

func (b *TronRPC) getBestHeader() (bchain.EVMHeader, error) {
	var err error
	var header bchain.EVMHeader
	ctx, cancel := context.WithTimeout(context.Background(), b.Timeout)
	defer cancel()
	header, err = b.Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	b.UpdateBestHeader(header)
	return header, nil
}

// GetChainParser returns Tron-specific BlockChainParser
func (b *TronRPC) GetChainParser() bchain.BlockChainParser {
	return b.Parser
}

func (b *TronRPC) CreateMempool(chain bchain.BlockChain) (bchain.Mempool, error) {
	if b.Mempool == nil {
		b.Mempool = bchain.NewMempoolEthereumType(chain, b.ChainConfig.MempoolTxTimeoutHours, b.ChainConfig.QueryBackendOnMempoolResync)
	}
	return b.Mempool, nil
}

func (b *TronRPC) InitializeMempool(addrDescForOutpoint bchain.AddrDescForOutpointFunc, onNewTxAddr bchain.OnNewTxAddrFunc, onNewTx bchain.OnNewTxFunc) error {
	if b.Mempool == nil {
		return errors.New("Tron Mempool not created")
	}
	b.Mempool.OnNewTxAddr = onNewTxAddr
	b.Mempool.OnNewTx = onNewTx

	if b.mq == nil {
		mq, err := bchain.NewMQ(b.ChainConfig.MessageQueueBinding, b.PushHandler)
		if err != nil {
			return err
		}
		b.mq = mq
	}

	return nil
}

func (b *TronRPC) Shutdown(ctx context.Context) error {
	if b.mq != nil {
		if err := b.mq.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Tron does not have any method for getting mempool transactions (does not support parameter 'pending' in eth_getBlockByNumber)
// https://developers.tron.network/reference/eth_getblockbynumber
func (b *TronRPC) GetMempoolTransactions() ([]string, error) {
	return nil, nil
}

func (b *TronRPC) EthereumTypeGetBalance(addrDesc bchain.AddressDescriptor) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.Timeout)
	defer cancel()
	fmt.Printf("Original addrDesc bytes: %x\n", addrDesc)
	fmt.Printf("Original addrDesc length: %d\n", len(addrDesc))

	return b.Client.BalanceAt(ctx, addrDesc, nil)
}

// EthereumTypeGetNonce returns current balance of an address
func (b *TronRPC) EthereumTypeGetNonce(addrDesc bchain.AddressDescriptor) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.Timeout)
	defer cancel()
	return b.Client.NonceAt(ctx, addrDesc, nil)
}
