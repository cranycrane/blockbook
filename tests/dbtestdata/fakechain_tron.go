package dbtestdata

import (
	"strconv"

	"github.com/trezor/blockbook/bchain"
)

// fakeBlockChainTronType
type fakeBlockChainTronType struct {
	*fakeBlockChainEthereumType
}

// NewFakeBlockChainTronType
func NewFakeBlockChainTronType(parser bchain.BlockChainParser) (bchain.BlockChain, error) {
	return &fakeBlockChainTronType{
		fakeBlockChainEthereumType: &fakeBlockChainEthereumType{&fakeBlockChain{&bchain.BaseChain{Parser: parser}}},
	}, nil
}

// GetChainInfo
func (c *fakeBlockChainTronType) GetChainInfo() (*bchain.ChainInfo, error) {
	return &bchain.ChainInfo{
		Chain:         c.GetNetworkName(),
		Blocks:        1,
		Headers:       1,
		Bestblockhash: GetTestTronBlock1(c.Parser).BlockHeader.Hash,
		Version:       "tron_test_1.0",
		Subversion:    "MockTron",
	}, nil
}

// GetBestBlockHash
func (c *fakeBlockChainTronType) GetBestBlockHash() (string, error) {
	return GetTestTronBlock1(c.Parser).BlockHeader.Hash, nil
}

// GetBestBlockHeight
func (c *fakeBlockChainTronType) GetBestBlockHeight() (uint32, error) {
	return GetTestTronBlock1(c.Parser).BlockHeader.Height, nil
}

// GetBlockHash
func (c *fakeBlockChainTronType) GetBlockHash(height uint32) (string, error) {
	b := GetTestTronBlock1(c.Parser)
	if height == b.BlockHeader.Height {
		return b.BlockHeader.Hash, nil
	}
	return "", bchain.ErrBlockNotFound
}

// GetBlockHeader
func (c *fakeBlockChainTronType) GetBlockHeader(hash string) (*bchain.BlockHeader, error) {
	b := GetTestTronBlock1(c.Parser)
	if hash == b.BlockHeader.Hash {
		return &b.BlockHeader, nil
	}
	return nil, bchain.ErrBlockNotFound
}

// GetBlock
func (c *fakeBlockChainTronType) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	b1 := GetTestTronBlock0(c.Parser)
	if hash == b1.BlockHeader.Hash || height == b1.BlockHeader.Height {
		return b1, nil
	}
	b2 := GetTestTronBlock1(c.Parser)
	if hash == b2.BlockHeader.Hash || height == b2.BlockHeader.Height {
		return b2, nil
	}
	return nil, bchain.ErrBlockNotFound
}

func (c *fakeBlockChainTronType) GetBlockInfo(hash string) (*bchain.BlockInfo, error) {
	b := GetTestTronBlock1(c.Parser)
	if hash == b.BlockHeader.Hash {
		return getBlockInfo(b), nil
	}
	return nil, bchain.ErrBlockNotFound
}

// GetTransaction
func (c *fakeBlockChainTronType) GetTransaction(txid string) (*bchain.Tx, error) {
	blk := GetTestTronBlock1(c.Parser)
	t := getTxInBlock(blk, txid)
	if t == nil {
		return nil, bchain.ErrTxNotFound
	}
	return t, nil
}

func (c *fakeBlockChainTronType) GetContractInfo(contractDesc bchain.AddressDescriptor) (*bchain.ContractInfo, error) {
	addresses, _, _ := c.Parser.GetAddressesFromAddrDesc(contractDesc)
	return &bchain.ContractInfo{
		Type:           bchain.UnknownTokenType,
		Contract:       addresses[0],
		Name:           "TronTestContract" + strconv.Itoa(int(contractDesc[0])),
		Symbol:         "TRC" + strconv.Itoa(int(contractDesc[0])),
		Decimals:       6,
		CreatedInBlock: 1000,
	}, nil
}
