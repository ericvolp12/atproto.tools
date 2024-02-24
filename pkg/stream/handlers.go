package stream

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/labstack/echo/v4"
)

type RecordsResponse struct {
	Records []Record `json:"records"`
	Error   string   `json:"error,omitempty"`
}

type RecordsQuery struct {
	DID        *syntax.DID
	Collection *syntax.NSID
	Rkey       *syntax.RecordKey
	Seq        *int64
	Limit      int
}

// HandleGetRecords handles the GET /records endpoint
func (s *Stream) HandleGetRecords(c echo.Context) error {
	// Parse the query parameters
	// did - Repo DID (optional)
	// collection - Collection NSID (optional)
	// rkey - Record Key (optional)
	// seq - Firehose sequence number (optional)
	// limit - Number of records to return (default=100)

	// Validate the query parameters
	didParam := c.QueryParam("did")
	collectionParam := c.QueryParam("collection")
	rkeyParam := c.QueryParam("rkey")
	seqParam := c.QueryParam("seq")
	limitParam := c.QueryParam("limit")

	resp := RecordsResponse{}

	query := RecordsQuery{}

	if didParam != "" {
		did, err := syntax.ParseDID(didParam)
		if err != nil {
			resp.Error = fmt.Sprintf("invalid DID: %s", err)
			return c.JSON(http.StatusBadRequest, resp)
		}
		query.DID = &did
	}

	if collectionParam != "" {
		collection, err := syntax.ParseNSID(collectionParam)
		if err != nil {
			resp.Error = fmt.Sprintf("invalid collection: %s", err)
			return c.JSON(http.StatusBadRequest, resp)
		}
		query.Collection = &collection
	}

	if rkeyParam != "" {
		rkey, err := syntax.ParseRecordKey(rkeyParam)
		if err != nil {
			resp.Error = fmt.Sprintf("invalid record key: %s", err)
			return c.JSON(http.StatusBadRequest, resp)
		}
		query.Rkey = &rkey
	}

	if seqParam != "" {
		seq, err := strconv.ParseInt(seqParam, 10, 64)
		if err != nil {
			resp.Error = fmt.Sprintf("invalid sequence number: %s", err)
			return c.JSON(http.StatusBadRequest, resp)
		}
		query.Seq = &seq
	}

	if limitParam != "" {
		limit, err := strconv.Atoi(limitParam)
		if err != nil {
			resp.Error = fmt.Sprintf("invalid limit: %s", err)
			return c.JSON(http.StatusBadRequest, resp)
		}
		query.Limit = limit
	} else {
		query.Limit = 100
	}

	if query.Limit < 1 {
		query.Limit = 100
	}

	if query.Limit > 1000 {
		query.Limit = 1000
	}

	// Query the database
	var records []Record
	q := s.db
	if query.DID != nil {
		q = q.Where("repo = ?", query.DID.String())
	}
	if query.Collection != nil {
		q = q.Where("collection = ?", query.Collection.String())
	}
	if query.Rkey != nil {
		q = q.Where("r_key = ?", query.Rkey.String())
	}
	if query.Seq != nil {
		q = q.Where("firehose_seq = ?", *query.Seq)
	}
	q = q.Order("firehose_seq DESC, collection DESC, r_key DESC").Limit(query.Limit).Find(&records)

	if q.Error != nil {
		resp.Error = q.Error.Error()
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Records = records
	return c.JSON(http.StatusOK, resp)
}
