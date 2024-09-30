package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/ericvolp12/atproto.tools/pkg/plc"
	"github.com/labstack/echo/v4"
)

type API struct {
	plc *plc.PLC
}

func NewAPI(plc *plc.PLC) *API {
	return &API{plc: plc}
}

func (a *API) HandleGetDIDDoc(e echo.Context) error {
	// Unpack the DID from the path
	did := e.Param("did")

	_, err := syntax.ParseDID(did)
	if err != nil {
		return e.String(http.StatusBadRequest, fmt.Sprintf("invalid DID: %s", err))
	}

	doc, err := a.plc.GetDIDDocument(e.Request().Context(), did)
	if err != nil {
		if errors.Is(err, plc.ErrNotFound) {
			return e.String(http.StatusNotFound, fmt.Sprintf("DID not found: %s", did))
		}
		return e.String(http.StatusInternalServerError, fmt.Sprintf("failed to get DID document: %s", err))
	}

	return e.JSON(http.StatusOK, doc)
}
