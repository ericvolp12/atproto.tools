package stream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/labstack/echo/v4"
)

type JSONRecord struct {
	FirehoseSeq int64                  `json:"seq"`
	Repo        string                 `json:"repo"`
	Collection  string                 `json:"collection"`
	RKey        string                 `json:"rkey"`
	Action      string                 `json:"action"`
	Raw         map[string]interface{} `json:"raw,omitempty"`
}

type RecordsResponse struct {
	Records []JSONRecord `json:"records"`
	Error   string       `json:"error,omitempty"`
}

type RecordsQuery struct {
	DID        *syntax.DID
	Collection *syntax.NSID
	Rkey       *syntax.RecordKey
	Seq        *int64
	Limit      int
}

func dbRecordToJSONRecord(r Record) JSONRecord {
	rec := JSONRecord{
		FirehoseSeq: r.FirehoseSeq,
		Repo:        r.Repo,
		Collection:  r.Collection,
		RKey:        r.RKey,
		Action:      r.Action,
	}

	if r.Raw != nil {
		// Convert the RAW field to a JSON object
		var rawAsJSON map[string]interface{}
		err := json.Unmarshal(r.Raw, &rawAsJSON)
		if err != nil {
			rawAsJSON = map[string]interface{}{"error": err.Error()}
		}
		rec.Raw = rawAsJSON
	}

	return rec
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

	// Only allow querying by collection if a DID is provided
	// Only allow querying by rkey if a DID and collection are provided
	if query.Collection != nil && query.DID == nil {
		resp.Error = "cannot query by collection without a DID"
		return c.JSON(http.StatusBadRequest, resp)
	}
	if query.Rkey != nil && (query.DID == nil || query.Collection == nil) {
		resp.Error = "cannot query by rkey without a DID and collection"
		return c.JSON(http.StatusBadRequest, resp)
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
	q := s.reader
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
	q = q.Order("id DESC").Limit(query.Limit).Find(&records)

	if q.Error != nil {
		resp.Error = q.Error.Error()
		return c.JSON(http.StatusInternalServerError, resp)
	}

	// Convert the records to JSON
	resp.Records = make([]JSONRecord, len(records))
	for i, r := range records {
		resp.Records[i] = dbRecordToJSONRecord(r)
	}
	return c.JSON(http.StatusOK, resp)
}

type JSONEvent struct {
	FirehoseSeq int64   `json:"seq"`
	Repo        string  `json:"repo"`
	EventType   string  `json:"event_type"`
	Error       string  `json:"error,omitempty"`
	Time        int64   `json:"time"`
	Since       *string `json:"since"`
}

type EventsResponse struct {
	Events []JSONEvent `json:"events"`
	Error  string      `json:"error,omitempty"`
}

type EventsQuery struct {
	DID       *syntax.DID
	EventType *string
	Seq       *int64
	Limit     int
}

func dbEventToJSONEvent(e Event) JSONEvent {
	return JSONEvent{
		FirehoseSeq: e.FirehoseSeq,
		Repo:        e.Repo,
		EventType:   e.EventType,
		Error:       e.Error,
		Time:        e.Time,
		Since:       e.Since,
	}
}

// HandleGetEvents handles the GET /events endpoint
func (s *Stream) HandleGetEvents(c echo.Context) error {
	// Parse the query parameters
	// did - Repo DID (optional)
	// event_type - Event type (optional)
	// seq - Firehose sequence number (optional)
	// limit - Number of events to return (default=100)

	// Validate the query parameters
	didParam := c.QueryParam("did")
	eventTypeParam := c.QueryParam("event_type")
	seqParam := c.QueryParam("seq")
	limitParam := c.QueryParam("limit")

	resp := EventsResponse{}

	query := EventsQuery{}

	if didParam != "" {
		did, err := syntax.ParseDID(didParam)
		if err != nil {
			resp.Error = fmt.Sprintf("invalid DID: %s", err)
			return c.JSON(http.StatusBadRequest, resp)
		}
		query.DID = &did
	}

	if eventTypeParam != "" {
		query.EventType = &eventTypeParam
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
	var events []Event
	q := s.reader
	if query.DID != nil {
		q = q.Where("repo = ?", query.DID.String())
	}
	if query.EventType != nil {
		q = q.Where("event_type = ?", *query.EventType)
	}
	if query.Seq != nil {
		q = q.Where("firehose_seq = ?", *query.Seq)
	}
	q = q.Order("firehose_seq DESC").Limit(query.Limit).Find(&events)

	if q.Error != nil {
		resp.Error = q.Error.Error()
		return c.JSON(http.StatusInternalServerError, resp)
	}

	// Convert the events to JSON
	resp.Events = make([]JSONEvent, len(events))
	for i, e := range events {
		resp.Events[i] = dbEventToJSONEvent(e)
	}
	return c.JSON(http.StatusOK, resp)
}

type JSONIdentity struct {
	DID       string    `json:"did"`
	Handle    string    `json:"handle"`
	PDS       string    `json:"pds"`
	UpdatedAt time.Time `json:"updated_at"`
}

type IdentitiesResponse struct {
	Identities []JSONIdentity `json:"identities"`
	Error      string         `json:"error,omitempty"`
}

type IdentitiesQuery struct {
	DID    *syntax.DID
	Handle *syntax.Handle
	PDS    *string
	Limit  int
}

func dbIdentityToJSONIdentity(i Identity) JSONIdentity {
	return JSONIdentity{
		DID:       i.DID,
		Handle:    i.Handle,
		PDS:       i.PDS,
		UpdatedAt: i.UpdatedAt,
	}
}

func (s *Stream) HandleGetIdentities(c echo.Context) error {
	// Parse the query parameters
	// did - Repo DID (optional)
	// handle - Repo Handle (optional)
	// pds - Rep PDS endpoint (optional)
	// limit - Number of identities to return (default=100)

	// Validate the query parameters
	didParam := c.QueryParam("did")
	handleParam := c.QueryParam("handle")
	pdsParam := c.QueryParam("pds")
	limitParam := c.QueryParam("limit")

	resp := IdentitiesResponse{}

	query := IdentitiesQuery{}

	if didParam != "" {
		did, err := syntax.ParseDID(didParam)
		if err != nil {
			resp.Error = fmt.Sprintf("invalid DID: %s", err)
			return c.JSON(http.StatusBadRequest, resp)
		}
		query.DID = &did
	}

	if handleParam != "" {
		handle, err := syntax.ParseHandle(handleParam)
		if err != nil {
			resp.Error = fmt.Sprintf("invalid handle: %s", err)
			return c.JSON(http.StatusBadRequest, resp)
		}
		query.Handle = &handle
	}

	if pdsParam != "" {
		query.PDS = &pdsParam
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
	var identities []Identity
	q := s.reader
	if query.DID != nil {
		q = q.Where("d_id = ?", query.DID.String())
	}
	if query.Handle != nil {
		q = q.Where("handle = ?", query.Handle.String())
	}
	if query.PDS != nil {
		q = q.Where("pds = ?", *query.PDS)
	}
	q = q.Order("created_at DESC").Limit(query.Limit).Find(&identities)

	if q.Error != nil {
		resp.Error = q.Error.Error()
		return c.JSON(http.StatusInternalServerError, resp)
	}

	// Convert the identities to JSON
	resp.Identities = make([]JSONIdentity, len(identities))
	for i, id := range identities {
		resp.Identities[i] = dbIdentityToJSONIdentity(id)
	}
	return c.JSON(http.StatusOK, resp)
}
