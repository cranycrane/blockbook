//go:build unittest
// +build unittest

package server

import (
	"github.com/golang/glog"
	"github.com/trezor/blockbook/bchain/coins/tron"
	"github.com/trezor/blockbook/tests/dbtestdata"
	"net/http"
	"net/http/httptest"
	"testing"
)

func httpTestsTron(t *testing.T, ts *httptest.Server) {
	tests := []httpTests{
		{
			name:        "apiAddress TronAddrTJ",
			r:           newGetRequest(ts.URL + "/api/v2/address/" + dbtestdata.TronAddrTJ),
			status:      http.StatusOK,
			contentType: "application/json; charset=utf-8",
			body: []string{
				`{"page":1,"totalPages":1,"itemsOnPage":1000,"address":"TJngGWiRMLgNFScEybQxLEKQMNdB4nR6Vx","balance":"123450096","unconfirmedBalance":"0","unconfirmedTxs":0,"txs":1,"nonTokenTxs":1,"txids":["0xc92919ad24ffd58f760b18df7949f06e1190cf54a50a0e3745a385608ed3cbf2"],"nonce":"96"}`,
			},
		},
		{
			name:        "apiAddress TronAddrTX",
			r:           newGetRequest(ts.URL + "/api/v2/address/" + dbtestdata.TronAddrTX + "?details=txs"),
			status:      http.StatusOK,
			contentType: "application/json; charset=utf-8",
			body: []string{
				`{"page":1,"totalPages":1,"itemsOnPage":1000,"address":"TXncUDXYkRCmwhFikxYMutwAy93fbhPbbv","balance":"123450239","unconfirmedBalance":"0","unconfirmedTxs":0,"txs":1,"nonTokenTxs":1,"transactions":[{"txid":"0xc92919ad24ffd58f760b18df7949f06e1190cf54a50a0e3745a385608ed3cbf2","vin":[{"n":0,"addresses":["TXncUDXYkRCmwhFikxYMutwAy93fbhPbbv"],"isAddress":true,"isOwn":true}],"vout":[{"value":"257","n":0,"addresses":["TJngGWiRMLgNFScEybQxLEKQMNdB4nR6Vx"],"isAddress":true}],"blockHeight":-1,"confirmations":0,"blockTime":0,"value":"257","fees":"27720","rbf":true,"coinSpecificData":{"tx":{"nonce":"0x0","gasPrice":"0x1a4","gas":"0x34dc","to":"TJngGWiRMLgNFScEybQxLEKQMNdB4nR6Vx","value":"0x101","input":"0x23b872dd00000000000000000000000034627862d50389c8d7a1ab5ef074b84ab4ddb9e90000000000000000000000000cecca0e53477d2b6c562ab68c3452fc99f7817e000000000000000000000000000000000000000000000000000000000000067f","hash":"0xc92919ad24ffd58f760b18df7949f06e1190cf54a50a0e3745a385608ed3cbf2","blockNumber":"0x186a0","from":"TXncUDXYkRCmwhFikxYMutwAy93fbhPbbv","transactionIndex":"0x0"},"receipt":{"gasUsed":"0x42","status":"0x1","logs":[]}},"ethereumSpecific":{"status":1,"nonce":0,"gasLimit":13532,"gasUsed":66,"gasPrice":"420","data":"0x23b872dd00000000000000000000000034627862d50389c8d7a1ab5ef074b84ab4ddb9e90000000000000000000000000cecca0e53477d2b6c562ab68c3452fc99f7817e000000000000000000000000000000000000000000000000000000000000067f","parsedData":{"methodId":"0x23b872dd","name":""}}}],"nonce":"239"}`,
			},
		},
	}

	performHttpTests(tests, t, ts)
}

func Test_PublicServer_Tron(t *testing.T) {
	timeNow = fixedTimeNow
	parser := tron.NewTronParser(1, true)
	chain, err := dbtestdata.NewFakeBlockChainTronType(parser)
	if err != nil {
		glog.Fatal("fakechain: ", err)
	}

	s, dbpath := setupPublicHTTPServer(parser, chain, t, false)
	defer closeAndDestroyPublicServer(t, s, dbpath)
	s.ConnectFullPublicInterface()

	ts := httptest.NewServer(s.https.Handler)
	defer ts.Close()

	httpTestsTron(t, ts)
}
