// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

// Package protocol describes the REST protocol
package protocol

import (
	"errors"
	"net/mail"

	"github.com/rafaeljusto/shelter/model"
)

// List of possible errors that can occur when calling methods from this object. Other
// erros can also occurs from low level layers
var (
	// Error when an invalid language is given. List of possible values can be found in IANA
	// website
	ErrInvalidLanguage = errors.New("Invalid owner language")
)

// Owner object used in the protocol to determinate what the user can update, for this
// case, everything
type OwnerRequest struct {
	Email    string `json:"email,omitempty"`    // E-mail that the owner wants to be alerted
	Language string `json:"language,omitempty"` // Language that the owner wants to receive the messages
}

// Convert a owner request object into a owner model object. It can return errors related
// to the e-mail format
func (o *OwnerRequest) toOwnerModel() (model.Owner, error) {
	var owner model.Owner

	email, err := mail.ParseAddress(o.Email)
	if err != nil {
		return owner, err
	}

	if !model.IsValidLanguage(o.Language) {
		return owner, ErrInvalidLanguage
	}

	owner = model.Owner{
		Email:    email,
		Language: model.NormalizeLanguage(o.Language),
	}

	return owner, nil
}

// Convert a list of owner requests objects into a list of owner model objects. Useful
// when merging domain object from the network with a domain object from the database. It
// can return errors related to the e-mail format in one of the converted owners
func toOwnersModel(ownersRequest []OwnerRequest) ([]model.Owner, error) {
	var owners []model.Owner
	for _, ownerRequest := range ownersRequest {
		owner, err := ownerRequest.toOwnerModel()
		if err != nil {
			return nil, err
		}

		owners = append(owners, owner)
	}

	return owners, nil
}

// Owner object used in the protocol to determinate what the user can see
type OwnerResponse struct {
	Email    string `json:"email,omitempty"`    // E-mail that the owner wants to be alerted
	Language string `json:"language,omitempty"` // Language that the owner wants to receive the messages
}

// Convert a owner of the system into a format with limited information to return it to
// the user. For now we are not limiting any information
func toOwnerResponse(owner model.Owner) OwnerResponse {
	return OwnerResponse{
		Email:    owner.Email.Address,
		Language: owner.Language,
	}
}

// Convert a list of owners of the system into a format with limited information to return
// it to the user. This is only a easy way to call toOwnerResponse for each object in the
// list
func toOwnersResponse(owners []model.Owner) []OwnerResponse {
	var ownersResponse []OwnerResponse
	for _, owner := range owners {
		ownersResponse = append(ownersResponse, toOwnerResponse(owner))
	}
	return ownersResponse
}
