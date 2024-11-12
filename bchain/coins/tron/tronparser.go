package tron

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/decred/base58"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/eth"
	"strings"
)

// TronTypeAddressDescriptorLen - the AddressDescriptor of TronType has fixed length
// prefix - 0x41, sha256 checksum - 4 bytes
const TronTypeAddressDescriptorLen = 1 + 20 + 4

// TronAmountDecimalPoint defines number of decimal points in Tron amounts
// base unit is 'SUN', 1 TRX = 1,000,000 SUN
const TronAmountDecimalPoint = 6

// TronParser handle
type TronParser struct {
	*eth.EthereumParser // Kompozice EthereumParser
}

// NewTronParser vrací novou instanci TronParser
func NewTronParser(b int, addressAliases bool) *TronParser {
	ethParser := eth.NewEthereumParser(b, addressAliases)
	ethParser.AmountDecimalPoint = TronAmountDecimalPoint
	return &TronParser{
		EthereumParser: ethParser,
	}
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
	}

	if !strings.HasPrefix(address, "41") {
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
	leng := len(desc)
	if leng != TronTypeAddressDescriptorLen || desc[0] != 0x41 {
		return nil, false, errors.New("invalid Tron address: must start with '41' and have correct len")
	}

	base58Addr := base58.Encode(desc)
	return []string{base58Addr}, true, nil
}
