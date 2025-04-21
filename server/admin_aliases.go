package server

import (
	"encoding/csv"
	"github.com/linxGnu/grocksdb"
	"io"
	"net/http"

	"github.com/golang/glog"
)

// GET → zobrazí formulář;  POST → zpracuje CSV a přidá aliasy
func (s *PublicServer) explorerAdminAliases(w http.ResponseWriter, r *http.Request) (tpl, *TemplateData, error) {
	if r.Method == http.MethodGet {
		return adminAliasesTpl, s.newTemplateData(r), nil
	}

	f, _, err := r.FormFile("csv")
	if err != nil {
		return errorTpl, nil, err
	}
	defer f.Close()

	rdr := csv.NewReader(f)
	wb := grocksdb.NewWriteBatch()
	defer wb.Destroy()

	for {
		rec, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			glog.Error(err)
			continue
		}

		addr, alias := rec[0], rec[1]
		desc, err := s.chainParser.GetAddrDescFromAddress(addr)
		if err != nil {
			glog.Warning("skip ", addr, ": ", err)
			continue
		}

		s.db.PutAddressAttribution(wb, desc, alias)
	}
	if err := s.db.WriteBatch(wb); err != nil {
		return errorTpl, nil, err
	}
	return adminAliasesTpl, s.newTemplateData(r), nil
}
