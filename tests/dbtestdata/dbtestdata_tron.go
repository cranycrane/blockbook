package dbtestdata

import (
	"github.com/trezor/blockbook/bchain/coins/tron"
	"math/big"

	"github.com/trezor/blockbook/bchain"
)

// Addresses
const (
	TronAddrZero       = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb"
	TronAddrTJ         = "TJngGWiRMLgNFScEybQxLEKQMNdB4nR6Vx" // 0x60bb513e91aa723a10a4020ae6fcce39bce7e240
	TronAddrTX         = "TXncUDXYkRCmwhFikxYMutwAy93fbhPbbv" // 0xef51c82ea6336ba1544c4a182a7368e9fbe28274
	TronAddrContractTR = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // TRC20 (USDT)
	TronAddrContractTV = "TVj7RNVHy6thbM7BWdSe9G6gXwKhjhdNZS" // TRC20 (KLV)
	TronAddrContractTU = "TU2MJ5Veik1LRAgjeSzEdvmDYx7mefJZvd" // non TRC20
	TronAddrContractTA = "TQEepeTijBFcWjnwF7N6THWEYpxJjpwqdd" // TRC721
	TronAddrContractTX = "TXWLT4N9vDcmNHDnSuKv2odhBtizYuEMKJ" // TRC1155

)

const (
	// TronAddrTJ -> TronAddrTX, value 257
	TronTx1Id     = "0xc92919ad24ffd58f760b18df7949f06e1190cf54a50a0e3745a385608ed3cbf2"
	TronTx1Packed = "08a08d0610d0b2efa7061abf01120201a418dc69220201012a6423b872dd00000000000000000000000034627862d50389c8d7a1ab5ef074b84ab4ddb9e90000000000000000000000000cecca0e53477d2b6c562ab68c3452fc99f7817e000000000000000000000000000000000000000000000000000000000000067f3220c92919ad24ffd58f760b18df7949f06e1190cf54a50a0e3745a385608ed3cbf23a1460bb513e91aa723a10a4020ae6fcce39bce7e2404214ef51c82ea6336ba1544c4a182a7368e9fbe2827422060a0142120101"
)

var TronTx1InternalData = &bchain.EthereumInternalData{
	Transfers: []bchain.EthereumInternalTransfer{
		{
			Type:  bchain.CALL,
			From:  TronAddrTX,
			To:    TronAddrTJ,
			Value: *big.NewInt(999999),
		},
	},
	Error: "",
}

var TronBlock1SpecificData = &bchain.EthereumBlockSpecificData{
	Contracts: []bchain.ContractInfo{
		{
			Contract:       TronAddrContractTR,
			Type:           tron.TRC20TokenType,
			Name:           "USD Token",
			Symbol:         "USDT",
			Decimals:       12,
			CreatedInBlock: 99999,
		},
	},
}

func GetTestTronBlock0(parser bchain.BlockChainParser) *bchain.Block {
	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Height:        99999,
			Hash:          "0x0000000000000000000000000000000000000000000000000000000000000000",
			Time:          1694226700,
			Confirmations: 2,
		},
		Txs: []bchain.Tx{},
	}
}

func GetTestTronBlock1(parser bchain.BlockChainParser) *bchain.Block {
	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Height:        100000,
			Hash:          "0x11223344556677889900aabbccddeeff11223344556677889900aabbccddeeff",
			Size:          12345,
			Time:          1677700000,
			Confirmations: 99,
		},
		Txs: unpackTxs([]packedAndInternal{{
			packed: TronTx1Packed,
		}}, parser),
		CoinSpecificData: TronBlock1SpecificData,
	}
}
