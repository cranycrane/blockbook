package tron

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/decred/base58"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/eth"
	"strings"
)

// TronTypeAddressDescriptorLen - the AddressDescriptor of TronType has fixed length
// prefix - 0x41, sha256 checksum - 4 bytes
const TronTypeAddressDescriptorLen = 1 + 20 + 4

const TronTypeAddressHexLen = 20

// TronAmountDecimalPoint defines number of decimal points in Tron amounts
// base unit is 'SUN', 1 TRX = 1,000,000 SUN
const TronAmountDecimalPoint = 6

// TronParser handle
type TronParser struct {
	*eth.EthereumParser
}

// NewTronParser vrací novou instanci TronParser
func NewTronParser(b int, addressAliases bool) *TronParser {
	ethParser := eth.NewEthereumParser(b, addressAliases)
	ethParser.AmountDecimalPoint = TronAmountDecimalPoint
	return &TronParser{
		EthereumParser: ethParser,
	}
}

// GetAddrDescFromVout returns internal address representation of given transaction output
func (p *TronParser) GetAddrDescFromVout(output *bchain.Vout) (bchain.AddressDescriptor, error) {
	if len(output.ScriptPubKey.Addresses) != 1 {
		return nil, bchain.ErrAddressMissing
	}
	return p.GetAddrDescFromAddress(output.ScriptPubKey.Addresses[0])
}

func (p *TronParser) GetAddrDescFromAddress(address string) (bchain.AddressDescriptor, error) {
	if strings.HasPrefix(address, "T") {
		decoded := base58.Decode(address)
		if len(decoded) == 0 {
			return nil, errors.New("invalid Base58 Tron address")
		}
		return decoded, nil
	}

	if strings.HasPrefix(address, "0x") {
		address = address[2:]
		address = "41" + address
	}

	decoded, err := hex.DecodeString(address)
	if err != nil {
		return nil, err
	}

	firstSHA := sha256.Sum256(decoded)
	secondSHA := sha256.Sum256(firstSHA[:])
	checksum := secondSHA[:4]

	fullAddress := append(decoded, checksum...)

	return fullAddress, nil
}

// GetAddressesFromAddrDesc checks len and prefix and converts to base58
func (p *TronParser) GetAddressesFromAddrDesc(desc bchain.AddressDescriptor) ([]string, bool, error) {
	if !p.IsTronAddress(desc) {
		return nil, false, errors.New("invalid Tron address: must start with '41' and have correct len")
	}

	base58Addr := base58.Encode(desc)
	return []string{base58Addr}, true, nil
}

func (p *TronParser) TxToTx(tx *bchain.RpcTransaction, receipt *bchain.RpcReceipt, internalData *bchain.EthereumInternalData, blocktime int64, confirmations uint32, fixEIP55 bool) (*bchain.Tx, error) {
	txid := tx.Hash
	var (
		fa, ta []string
		err    error
	)
	if len(tx.From) > 2 {
		tx.From, err = p.ConvertToBase58Descriptor(tx.From)
		if err != nil {
			return nil, err
		}
		fa = []string{tx.From}
	}
	if len(tx.To) > 2 {
		tx.To, err = p.ConvertToBase58Descriptor(tx.To)
		if err != nil {
			return nil, err
		}
		ta = []string{tx.To}
	}
	if receipt != nil && receipt.Logs != nil {
		for _, l := range receipt.Logs {
			if len(l.Address) > 2 {
				l.Address, err = p.ConvertToBase58Descriptor(l.Address)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if internalData != nil {
		// ignore empty internal data
		if internalData.Type == bchain.CALL && len(internalData.Transfers) == 0 && len(internalData.Error) == 0 {
			internalData = nil
		} else {
			for i := range internalData.Transfers {
				it := &internalData.Transfers[i]
				it.From, err = p.ConvertToBase58Descriptor(it.From)
				if err != nil {
					return nil, err
				}
				it.To, err = p.ConvertToBase58Descriptor(it.To)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	ct := bchain.EthereumSpecificData{
		Tx:           tx,
		InternalData: internalData,
		Receipt:      receipt,
	}
	vs, err := hexutil.DecodeBig(tx.Value)
	if err != nil {
		return nil, err
	}
	return &bchain.Tx{
		Blocktime:     blocktime,
		Confirmations: confirmations,
		// Hex
		// LockTime
		Time: blocktime,
		Txid: txid,
		Vin: []bchain.Vin{
			{
				Addresses: fa,
				// Coinbase
				// ScriptSig
				// Sequence
				// Txid
				// Vout
			},
		},
		Vout: []bchain.Vout{
			{
				N:        0, // there is always up to one To address
				ValueSat: *vs,
				ScriptPubKey: bchain.ScriptPubKey{
					// Hex
					Addresses: ta,
				},
			},
		},
		CoinSpecificData: ct,
	}, nil
}

func (p *TronParser) EthereumTypeGetTokenTransfersFromTx(tx *bchain.Tx) (bchain.TokenTransfers, error) {
	var transfers bchain.TokenTransfers
	var err error
	transfers, err = p.EthereumParser.EthereumTypeGetTokenTransfersFromTx(tx)

	if err != nil {
		return nil, err
	}

	// Post-process the transfers to convert addresses to Tron format
	for i, transfer := range transfers {
		// Convert Contract address
		if transfer.Contract != "" {
			contract, err := p.ConvertToBase58Descriptor(transfer.Contract)
			if err != nil {
				return nil, err
			}
			transfers[i].Contract = contract
		}

		// Convert From address
		if transfer.From != "" {
			from, err := p.ConvertToBase58Descriptor(transfer.From)
			if err != nil {
				return nil, err
			}
			transfers[i].From = from
		}

		// Convert To address
		if transfer.To != "" {
			to, err := p.ConvertToBase58Descriptor(transfer.To)
			if err != nil {
				return nil, err
			}
			transfers[i].To = to
		}

		// Convert type (ERC20 -> TRC20, etc.)
		// Convert type (TokenType) to Tron-specific names
		switch transfer.Type {
		case bchain.FungibleToken: // ERC20 equivalent
			transfers[i].Type = bchain.FungibleToken // TRC20 equivalent
		case bchain.NonFungibleToken: // ERC721 equivalent
			transfers[i].Type = bchain.NonFungibleToken // TRC721 equivalent
		case bchain.MultiToken: // ERC1155 equivalent
			transfers[i].Type = bchain.MultiToken // TRC1155 equivalent
		default:
			// Handle unknown or new types (optional)
			// Leave as is, or log/flag for further investigation
		}

	}

	return transfers, nil
}

func (p *TronParser) ConvertToBase58Descriptor(address string) (string, error) {
	txDesc, err := p.GetAddrDescFromAddress(address)
	if err != nil {
		return "", err
	}

	if !p.IsTronAddress(txDesc) {
		return "", errors.New("invalid Tron address descriptor: must start with '41' and have correct length")
	}

	return base58.Encode(txDesc), nil
}

// FormatAddressAlias adds .tron to a name alias
func (p *TronParser) FormatAddressAlias(address string, name string) string {
	return name + ".tron"
}

func (p *TronParser) IsTronAddress(desc bchain.AddressDescriptor) bool {
	return len(desc) == TronTypeAddressDescriptorLen && desc[0] == 0x41
}
