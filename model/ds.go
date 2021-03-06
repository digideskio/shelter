// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

// Package model describes the objects of the system
package model

import (
	"time"
)

// List of possible DS algorithms (RFC 4034 - A.1, RFC 5155, RFC 5702, RFC 5933 and RFC
// 6605). Only algorithms used for signing were listed here
const (
	DSAlgorithmRSAMD5       DSAlgorithm = 1   // RSA/MD5
	DSAlgorithmDH           DSAlgorithm = 2   // DH
	DSAlgorithmDSASHA1      DSAlgorithm = 3   // DSA/SHA-1 [DSA]
	DSAlgorithmRSASHA1      DSAlgorithm = 5   // RSA/SHA-1 [RSASHA1]
	DSAlgorithmDSASHA1NSEC3 DSAlgorithm = 6   // DSA/SHA1-NSEC3
	DSAlgorithmRSASHA1NSEC3 DSAlgorithm = 7   // RSA/SHA1-NSEC3 [RSASHA1-NSEC3]
	DSAlgorithmRSASHA256    DSAlgorithm = 8   // RSA/SHA-256 [RSASHA256]
	DSAlgorithmRSASHA512    DSAlgorithm = 10  // RSA/SHA-512 [RSASHA512]
	DSAlgorithmECCGOST      DSAlgorithm = 12  // GOST R 34.10-2001
	DSAlgorithmECDSASHA256  DSAlgorithm = 13  // ECDSA/SHA-256 - Elliptic Curve Digital Signature
	DSAlgorithmECDSASHA384  DSAlgorithm = 14  // ECDSA/SHA-384 - Elliptic Curve Digital Signature
	DSAlgorithmIndirect     DSAlgorithm = 252 // Indirect
	DSAlgorithmPrivateDNS   DSAlgorithm = 253 // Private [PRIVATEDNS]
	DSAlgorithmPrivateOID   DSAlgorithm = 254 // Private [PRIVATEOID]
)

// DSAlgorithm is a number that represents one of the possible DS algorithms listed in the
// constant group above
type DSAlgorithm uint8

// When converting from user input a number to a DS algorithm, we need to check if this
// number is one of the supported algorithms from the system, so we use this function to
// check it. It will be probably called in the protocol layer
func IsValidDSAlgorithm(algorithm uint8) bool {
	switch algorithm {
	case uint8(DSAlgorithmRSAMD5),
		uint8(DSAlgorithmDH),
		uint8(DSAlgorithmDSASHA1),
		uint8(DSAlgorithmRSASHA1),
		uint8(DSAlgorithmDSASHA1NSEC3),
		uint8(DSAlgorithmRSASHA1NSEC3),
		uint8(DSAlgorithmRSASHA256),
		uint8(DSAlgorithmRSASHA512),
		uint8(DSAlgorithmECCGOST),
		uint8(DSAlgorithmECDSASHA256),
		uint8(DSAlgorithmECDSASHA384),
		uint8(DSAlgorithmIndirect),
		uint8(DSAlgorithmPrivateDNS),
		uint8(DSAlgorithmPrivateOID):
		return true

	default:
		return false
	}
}

// List of possible digest types according to RFCs 3658, 4034, 4035
const (
	DSDigestTypeReserved DSDigestType = 0
	DSDigestTypeSHA1     DSDigestType = 1 // Digest with 20 bytes (RFC 4034)
	DSDigestTypeSHA256   DSDigestType = 2 // Digest with 32 bytes (RFC 4509)
	DSDigestTypeGOST94   DSDigestType = 3 // Digest with 32 bytes (RFC 5933)
	DSDigestTypeSHA384   DSDigestType = 4 // Digest with 96 bytes (Experimental)
	DSDigestTypeSHA512   DSDigestType = 5 // Digest with 128 bytes (Experimental)
)

// DSDigestType is a number that represents one of the possible DS digest's types listed
// in the constant group above. It is useful when generating a DS from a DNSKEY for
// comparisson validation
type DSDigestType uint8

// When converting from user input a number to a DS digest type, we need to check if this
// number is one of the supported digest types from the system, so we use this function to
// check it. It will be probably called in the protocol layer
func IsValidDSDigestType(digestType uint8) bool {
	switch digestType {
	case uint8(DSDigestTypeReserved),
		uint8(DSDigestTypeSHA1),
		uint8(DSDigestTypeSHA256),
		uint8(DSDigestTypeGOST94),
		uint8(DSDigestTypeSHA384),
		uint8(DSDigestTypeSHA512):
		return true

	default:
		return false
	}
}

// List of possible DS status
const (
	DSStatusNotChecked       = iota // DS record not checked yet
	DSStatusOK                      // DNSSEC configuration for this DS is OK
	DSStatusTimeout                 // Network timeout while trying to retrieve the DNSKEY
	DSStatusNoSignature             // No RRSIG records found for the related DNSKEY
	DSStatusExpiredSignature        // At least one RRSIG record was expired
	DSStatusNoKey                   // No DNSKEY was found with the keytag of the DS
	DSStatusNoSEP                   // DNSKEY related to DS does not have the bit SEP on
	DSStatusSignatureError          // Error while checking DNSKEY signatures
	DSStatusDNSError                // DNS error (check nameserver status)
)

// DSStatus is a number that represents one of the possible DS status listed in the
// constant group above
type DSStatus int

// Convert the DS status enum to text for printing in reports or debugging
func DSStatusToString(status DSStatus) string {
	switch status {
	case DSStatusNotChecked:
		return "NOTCHECKED"
	case DSStatusOK:
		return "OK"
	case DSStatusTimeout:
		return "TIMEOUT"
	case DSStatusNoSignature:
		return "NOSIG"
	case DSStatusExpiredSignature:
		return "EXPSIG"
	case DSStatusNoKey:
		return "NOKEY"
	case DSStatusNoSEP:
		return "NOSEP"
	case DSStatusSignatureError:
		return "SIGERR"
	case DSStatusDNSError:
		return "DNSERR"
	}

	return ""
}

// DS store the information necessary to validate if a domain is configured correctly with
// DNSSEC, and it also stores the results of the validations. When the hosts have multiple
// DNSSEC problems, the worst problem (using a priority algorithm) will be stored in the
// DS
type DS struct {
	Keytag      uint16       // DNSKEY's identification number
	Algorithm   DSAlgorithm  // DNSKEY's algorithm
	Digest      string       // Hash of the DNSKEY content
	DigestType  DSDigestType // Hash type decided by user when generating the DS
	ExpiresAt   time.Time    // DNSKEY's signature expiration date
	LastStatus  DSStatus     // Result of the last configuration check
	LastCheckAt time.Time    // Time of the last configuration check
	LastOKAt    time.Time    // Last time that the DNSSEC configuration was OK
}

// ChangeStatus is a easy way to change the status of a DS because it also updates the
// last check date
func (d *DS) ChangeStatus(status DSStatus) {
	d.LastStatus = status
	d.LastCheckAt = time.Now()

	if status == DSStatusOK {
		d.LastOKAt = d.LastCheckAt
	}
}
