package db

import (
	"github.com/linxGnu/grocksdb"
	"github.com/trezor/blockbook/bchain"
)

// PutAddressAttribution uloží do cfAddressAttributions
func (d *RocksDB) PutAddressAttribution(
	wb *grocksdb.WriteBatch,
	addrDesc bchain.AddressDescriptor,
	attrib string,
) {
	wb.PutCF(d.cfh[cfAddressAttributions], addrDesc, []byte(attrib))
}

// GetAddressAttribution načte z cfAddressAttributions
func (d *RocksDB) GetAddressAttribution(
	addrDesc bchain.AddressDescriptor,
) (string, error) {
	val, err := d.db.GetCF(d.ro, d.cfh[cfAddressAttributions], addrDesc)
	if err != nil {
		return "", err
	}
	defer val.Free()
	return string(val.Data()), nil // prázdný řetězec = bez atribuce
}
