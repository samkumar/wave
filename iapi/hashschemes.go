package iapi

import (
	"fmt"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/immesys/asn1"
	"github.com/immesys/wave/serdes"
)

func HashSchemeFor(h asn1.External) HashScheme {
	switch {
	case h.OID.Equal(serdes.Keccak_256OID):
		return KECCAK256
	case h.OID.Equal(serdes.Sha3_256OID):
		return SHA3
	default:
		return &UnsupportedHashScheme{}
	}
}
func NewHashScheme(oid asn1.ObjectIdentifier) HashScheme {
	panic("ni")
}
func HashSchemeInstanceFor(h *asn1.External) HashSchemeInstance {
	switch {
	case h.OID.Equal(serdes.Keccak_256OID):
		ba, ok := h.Content.(serdes.Keccak_256)
		if !ok || len(ba) != 32 {
			return &UnsupportedHashSchemeInstance{}
		}
		return &HashSchemeInstance_Keccak_256{Val: ba}
	case h.OID.Equal(serdes.Sha3_256OID):
		ba, ok := h.Content.(serdes.Sha3_256)
		if !ok || len(ba) != 32 {
			return &UnsupportedHashSchemeInstance{}
		}
		return &HashSchemeInstance_Sha3_256{Val: ba}
	default:
		return &UnsupportedHashSchemeInstance{}
	}
}

var _ HashScheme = &UnsupportedHashScheme{}

type UnsupportedHashScheme struct{}

func (hs *UnsupportedHashScheme) Supported() bool {
	return false
}
func (hs *UnsupportedHashScheme) Instance(input []byte) (HashSchemeInstance, error) {
	return nil, fmt.Errorf("unsupported hash scheme")
}
func (hs *UnsupportedHashScheme) OID() asn1.ObjectIdentifier {
	return nil
}

var _ HashScheme = &HashScheme_Sha3_256{}

type HashScheme_Sha3_256 struct{}

var SHA3 = &HashScheme_Sha3_256{}

func (hs *HashScheme_Sha3_256) Supported() bool {
	return true
}
func (hs *HashScheme_Sha3_256) Instance(input []byte) (HashSchemeInstance, error) {
	hash := sha3.Sum256(input)
	return &HashSchemeInstance_Sha3_256{Val: hash[:]}, nil
}
func (hs *HashScheme_Sha3_256) OID() asn1.ObjectIdentifier {
	return serdes.Sha3_256OID
}

var _ HashScheme = &HashScheme_Keccak_256{}

type HashScheme_Keccak_256 struct{}

var KECCAK256 = &HashScheme_Keccak_256{}

func (hs *HashScheme_Keccak_256) Supported() bool {
	return true
}
func (hs *HashScheme_Keccak_256) Instance(input []byte) (HashSchemeInstance, error) {
	eng := sha3.NewKeccak256()
	eng.Write(input)
	hash := eng.Sum(nil)
	return &HashSchemeInstance_Keccak_256{Val: hash[:]}, nil
}
func (hs *HashScheme_Keccak_256) OID() asn1.ObjectIdentifier {
	return serdes.Keccak_256OID
}

type UnsupportedHashSchemeInstance struct{}

func (hs *UnsupportedHashSchemeInstance) Supported() bool {
	return false
}
func (hs *UnsupportedHashSchemeInstance) Value() []byte {
	panic("Value() on unsupported hash scheme instance")
}
func (hs *UnsupportedHashSchemeInstance) CanonicalForm() (*asn1.External, error) {
	return nil, fmt.Errorf("unsupported hash scheme instance")
}
func (hs *UnsupportedHashSchemeInstance) OID() asn1.ObjectIdentifier {
	return nil
}

var _ HashSchemeInstance = &HashSchemeInstance_Sha3_256{}

type HashSchemeInstance_Sha3_256 struct {
	Val []byte
}

func (hs *HashSchemeInstance_Sha3_256) Supported() bool {
	return true
}
func (hs *HashSchemeInstance_Sha3_256) Value() []byte {
	return hs.Val
}
func (hs *HashSchemeInstance_Sha3_256) CanonicalForm() (*asn1.External, error) {
	ex := asn1.NewExternal(serdes.Sha3_256(hs.Val))
	return &ex, nil
}
func (hs *HashSchemeInstance_Sha3_256) OID() asn1.ObjectIdentifier {
	return serdes.Sha3_256OID
}

var _ HashSchemeInstance = &HashSchemeInstance_Keccak_256{}

type HashSchemeInstance_Keccak_256 struct {
	Val []byte
}

func (hs *HashSchemeInstance_Keccak_256) Supported() bool {
	return true
}
func (hs *HashSchemeInstance_Keccak_256) Value() []byte {
	return hs.Val
}
func (hs *HashSchemeInstance_Keccak_256) CanonicalForm() (*asn1.External, error) {
	ex := asn1.NewExternal(serdes.Keccak_256(hs.Val))
	return &ex, nil
}
func (hs *HashSchemeInstance_Keccak_256) OID() asn1.ObjectIdentifier {
	return serdes.Keccak_256OID
}