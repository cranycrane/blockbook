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
const TronTypeAddressDescriptorLen = 20

// TronAddressLen - length of Tron Base58 address
const TronAddressLen = 34

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
	ethParser.FormatAddressFunc = ToTronAddressFromAddress
	ethParser.FromDescToAddressFunc = ToTronAddressFromDesc
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

func has0xPrefix(s string) bool {
	return len(s) >= 2 && s[0] == '0' && (s[1]|32) == 'x'
}

func (p *TronParser) GetAddrDescFromAddress(address string) (bchain.AddressDescriptor, error) {
	if has0xPrefix(address) {
		address = address[2:]
	}

	if len(address) == TronAddressLen {
		decoded := base58.Decode(address)
		if len(decoded) != 25 || decoded[0] != 0x41 {
			return nil, errors.New("invalid Tron base58 address")
		}
		return decoded[1:21], nil
	} else if len(address) != TronTypeAddressDescriptorLen*2 {
		return nil, bchain.ErrAddressMissing
	}

	return hex.DecodeString(address)
}

// GetAddressesFromAddrDesc checks len and prefix and converts to base58
func (p *TronParser) GetAddressesFromAddrDesc(desc bchain.AddressDescriptor) ([]string, bool, error) {
	if len(desc) != TronTypeAddressDescriptorLen {
		return nil, false, bchain.ErrAddressMissing
	}

	return []string{ToTronAddressFromDesc(desc)}, true, nil
}

func ToTronAddressFromDesc(addrDesc bchain.AddressDescriptor) string {
	withPrefix := append([]byte{0x41}, addrDesc...)

	firstSHA := sha256.Sum256(withPrefix)
	secondSHA := sha256.Sum256(firstSHA[:])
	checksum := secondSHA[:4]

	fullAddress := append(withPrefix, checksum...)

	base58Addr := base58.Encode(fullAddress)

	return base58Addr
}

func ToTronAddressFromAddress(address string) string {
	if has0xPrefix(address) {
		address = address[2:]
	}
	b, err := hex.DecodeString(address)
	if err != nil {
		return address
	}
	return ToTronAddressFromDesc(b)
}

func (p *TronParser) FromTronAddressToHex(addr string) string {
	desc, err := p.GetAddrDescFromAddress(addr)
	if err != nil {
		return addr
	}
	return "0x" + hex.EncodeToString(desc)
}

// FormatAddressAlias adds .tron to a name alias
func (p *TronParser) FormatAddressAlias(address string, name string) string {
	return name + ".tron"
}

func (p *TronParser) IsTronAddress(desc bchain.AddressDescriptor) bool {
	return len(desc) == TronTypeAddressDescriptorLen && desc[0] == 0x41
}

// todo possibly only need to transfer addresses
func (p *TronParser) TxToTx(tx *bchain.RpcTransaction, receipt *bchain.RpcReceipt, internalData *bchain.EthereumInternalData, blocktime int64, confirmations uint32, fixEIP55 bool) (*bchain.Tx, error) {
	return p.EthereumParser.TxToTx(tx, receipt, internalData, blocktime, confirmations, true)
}

func (p *TronParser) ParseInputData(signatures *[]bchain.FourByteSignature, data string) *bchain.EthereumParsedInputData {
	parsed := p.EthereumParser.ParseInputData(signatures, data)

	if parsed == nil {
		return nil
	}

	for i, param := range parsed.Params {
		if param.Type == "address" || strings.HasPrefix(param.Type, "address[") {
			for j, v := range param.Values {
				parsed.Params[i].Values[j] = ToTronAddressFromAddress(v)
			}
		}
	}

	return parsed
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
			contract := ToTronAddressFromAddress(transfer.Contract)
			transfers[i].Contract = contract
		}

		// Convert From address
		if transfer.From != "" {
			from := ToTronAddressFromAddress(transfer.From)
			transfers[i].From = from
		}

		// Convert To address
		if transfer.To != "" {
			to := ToTronAddressFromAddress(transfer.To)
			transfers[i].To = to
		}

	}

	return transfers, nil
}

func (p *TronParser) PackTx(tx *bchain.Tx, height uint32, blockTime int64) ([]byte, error) {
	r, ok := tx.CoinSpecificData.(bchain.EthereumSpecificData)
	if !ok {
		return nil, errors.New("Missing CoinSpecificData")
	}
	r.Tx.AccountNonce = SanitizeHexUint64String(r.Tx.AccountNonce)

	r.Tx.From = p.FromTronAddressToHex(r.Tx.From)
	r.Tx.To = p.FromTronAddressToHex(r.Tx.To)

	for _, l := range r.Receipt.Logs {
		l.Address = p.FromTronAddressToHex(l.Address)
	}

	return p.EthereumParser.PackTx(tx, height, blockTime)
}

func SanitizeHexUint64String(s string) string {
	if strings.HasPrefix(s, "0x") {
		sanitized := strings.TrimLeft(s[2:], "0")
		if sanitized == "" {
			return "0x0"
		}
		return "0x" + sanitized
	}
	return s
}
