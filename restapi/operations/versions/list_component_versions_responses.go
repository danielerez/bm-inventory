// Code generated by go-swagger; DO NOT EDIT.

package versions

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/filanov/bm-inventory/models"
)

// ListComponentVersionsOKCode is the HTTP code returned for type ListComponentVersionsOK
const ListComponentVersionsOKCode int = 200

/*ListComponentVersionsOK Success.

swagger:response listComponentVersionsOK
*/
type ListComponentVersionsOK struct {

	/*
	  In: Body
	*/
	Payload models.ListVersions `json:"body,omitempty"`
}

// NewListComponentVersionsOK creates ListComponentVersionsOK with default headers values
func NewListComponentVersionsOK() *ListComponentVersionsOK {

	return &ListComponentVersionsOK{}
}

// WithPayload adds the payload to the list component versions o k response
func (o *ListComponentVersionsOK) WithPayload(payload models.ListVersions) *ListComponentVersionsOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the list component versions o k response
func (o *ListComponentVersionsOK) SetPayload(payload models.ListVersions) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ListComponentVersionsOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	payload := o.Payload
	if payload == nil {
		// return empty map
		payload = models.ListVersions{}
	}

	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}
