package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

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

func (a *API) HandleReverseSimple(e echo.Context) error {
	handleOrDID, err := url.PathUnescape(e.Param("handleOrDID"))
	if err != nil {
		return e.String(http.StatusBadRequest, fmt.Sprintf("invalid handle or DID: %s", err))
	}

	_, err = syntax.ParseDID(handleOrDID)
	if err != nil {
		// Try to parse as a handle
		handle, handleErr := syntax.ParseHandle(handleOrDID)
		if handleErr != nil {
			return e.String(http.StatusBadRequest, fmt.Sprintf("invalid DID or Handle: %s | %s", err, handleErr))
		}

		did, err := a.plc.GetDIDByHandle(e.Request().Context(), handle.String())
		if err != nil {
			if errors.Is(err, plc.ErrNotFound) {
				return e.String(http.StatusNotFound, fmt.Sprintf("Handle not found: %s", handle))
			}
			return e.String(http.StatusInternalServerError, fmt.Sprintf("failed to get DID by handle: %s", err))
		}

		return e.JSON(http.StatusOK, map[string]string{"did": did})
	}

	handle, err := a.plc.GetHandleByDID(e.Request().Context(), handleOrDID)
	if err != nil {
		if errors.Is(err, plc.ErrNotFound) {
			return e.String(http.StatusNotFound, fmt.Sprintf("DID not found: %s", handleOrDID))
		}
		return e.String(http.StatusInternalServerError, fmt.Sprintf("failed to get handle by DID: %s", err))
	}

	return e.JSON(http.StatusOK, map[string]string{"handle": handle})
}
