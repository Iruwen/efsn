// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"strings"

	"github.com/FusionFoundation/efsn/common/hexutil"
	"github.com/FusionFoundation/efsn/crypto/sha3"
	"github.com/FusionFoundation/efsn/rlp"
)

// Lengths of hashes and addresses in bytes.
const (
	// HashLength is the expected length of the hash
	HashLength = 32
	// AddressLength is the expected length of the adddress
	AddressLength = 20
)

var (
	hashT    = reflect.TypeOf(Hash{})
	addressT = reflect.TypeOf(Address{})
)

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// BigToHash sets byte representation of b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }

// HexToHash sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) Hash { return BytesToHash(FromHex(s)) }

// Bytes gets the byte representation of the underlying hash.
func (h Hash) Bytes() []byte { return h[:] }

// Big converts a hash to a big integer.
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex converts a hash to a hex string.
func (h Hash) Hex() string { return hexutil.Encode(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x…%x", h[:3], h[29:])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()
}

// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
func (h Hash) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), h[:])
}

// UnmarshalText parses a hash in hex syntax.
func (h *Hash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Hash", input, h[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (h *Hash) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(hashT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h Hash) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// Generate implements testing/quick.Generator.
func (h Hash) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h))
	for i := len(h) - 1; i > m; i-- {
		h[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

// Scan implements Scanner for database/sql.
func (h *Hash) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("can't scan %T into Hash", src)
	}
	if len(srcB) != HashLength {
		return fmt.Errorf("can't scan []byte of len %d into Hash, want %d", len(srcB), HashLength)
	}
	copy(h[:], srcB)
	return nil
}

// Value implements valuer for database/sql.
func (h Hash) Value() (driver.Value, error) {
	return h[:], nil
}

// UnprefixedHash allows marshaling a Hash without 0x prefix.
type UnprefixedHash Hash

// UnmarshalText decodes the hash from hex. The 0x prefix is optional.
func (h *UnprefixedHash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedHash", input, h[:])
}

// MarshalText encodes the hash as hex.
func (h UnprefixedHash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

/////////// Address

// Address represents the 20 byte address of an Ethereum account.
type Address [AddressLength]byte

// BytesToAddress returns Address with value b.
// If b is larger than len(h), b will be cropped from the left.
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

// BigToAddress returns Address with byte values of b.
// If b is larger than len(h), b will be cropped from the left.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// HexToAddress returns Address with byte values of s.
// If s is larger than len(h), s will be cropped from the left.
func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

// IsHexAddress verifies whether a string can represent a valid hex-encoded
// Ethereum address or not.
func IsHexAddress(s string) bool {
	if hasHexPrefix(s) {
		s = s[2:]
	}
	return len(s) == 2*AddressLength && isHex(s)
}

// Bytes gets the string representation of the underlying address.
func (a Address) Bytes() []byte { return a[:] }

// Big converts an address to a big integer.
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }

// Hash converts an address to a hash by left-padding it with zeros.
func (a Address) Hash() Hash { return BytesToHash(a[:]) }

// Hex returns an EIP55-compliant hex string representation of the address.
func (a Address) Hex() string {
	unchecksummed := hex.EncodeToString(a[:])
	sha := sha3.NewKeccak256()
	sha.Write([]byte(unchecksummed))
	hash := sha.Sum(nil)

	result := []byte(unchecksummed)
	for i := 0; i < len(result); i++ {
		hashByte := hash[i/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if result[i] > '9' && hashByte > 7 {
			result[i] -= 32
		}
	}
	return "0x" + string(result)
}

// String implements fmt.Stringer.
func (a Address) String() string {
	return a.Hex()
}

// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
func (a Address) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), a[:])
}

// SetBytes sets the address to the value of b.
// If b is larger than len(a) it will panic.
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

// MarshalText returns the hex representation of a.
func (a Address) MarshalText() ([]byte, error) {
	return hexutil.Bytes(a[:]).MarshalText()
}

// UnmarshalText parses a hash in hex syntax.
func (a *Address) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Address", input, a[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (a *Address) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(addressT, input, a[:])
}

// Scan implements Scanner for database/sql.
func (a *Address) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("can't scan %T into Address", src)
	}
	if len(srcB) != AddressLength {
		return fmt.Errorf("can't scan []byte of len %d into Address, want %d", len(srcB), AddressLength)
	}
	copy(a[:], srcB)
	return nil
}

// Value implements valuer for database/sql.
func (a Address) Value() (driver.Value, error) {
	return a[:], nil
}

// UnprefixedAddress allows marshaling an Address without 0x prefix.
type UnprefixedAddress Address

// UnmarshalText decodes the address from hex. The 0x prefix is optional.
func (a *UnprefixedAddress) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedAddress", input, a[:])
}

// MarshalText encodes the address as hex.
func (a UnprefixedAddress) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(a[:])), nil
}

// MixedcaseAddress retains the original string, which may or may not be
// correctly checksummed
type MixedcaseAddress struct {
	addr     Address
	original string
}

// NewMixedcaseAddress constructor (mainly for testing)
func NewMixedcaseAddress(addr Address) MixedcaseAddress {
	return MixedcaseAddress{addr: addr, original: addr.Hex()}
}

// NewMixedcaseAddressFromString is mainly meant for unit-testing
func NewMixedcaseAddressFromString(hexaddr string) (*MixedcaseAddress, error) {
	if !IsHexAddress(hexaddr) {
		return nil, fmt.Errorf("Invalid address")
	}
	a := FromHex(hexaddr)
	return &MixedcaseAddress{addr: BytesToAddress(a), original: hexaddr}, nil
}

// UnmarshalJSON parses MixedcaseAddress
func (ma *MixedcaseAddress) UnmarshalJSON(input []byte) error {
	if err := hexutil.UnmarshalFixedJSON(addressT, input, ma.addr[:]); err != nil {
		return err
	}
	return json.Unmarshal(input, &ma.original)
}

// MarshalJSON marshals the original value
func (ma *MixedcaseAddress) MarshalJSON() ([]byte, error) {
	if strings.HasPrefix(ma.original, "0x") || strings.HasPrefix(ma.original, "0X") {
		return json.Marshal(fmt.Sprintf("0x%s", ma.original[2:]))
	}
	return json.Marshal(fmt.Sprintf("0x%s", ma.original))
}

// Address returns the address
func (ma *MixedcaseAddress) Address() Address {
	return ma.addr
}

// String implements fmt.Stringer
func (ma *MixedcaseAddress) String() string {
	if ma.ValidChecksum() {
		return fmt.Sprintf("%s [chksum ok]", ma.original)
	}
	return fmt.Sprintf("%s [chksum INVALID]", ma.original)
}

// ValidChecksum returns true if the address has valid checksum
func (ma *MixedcaseAddress) ValidChecksum() bool {
	return ma.original == ma.addr.Hex()
}

// Original returns the mixed-case input string
func (ma *MixedcaseAddress) Original() string {
	return ma.original
}

// FSNCallAddress wacom
var FSNCallAddress = HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")

// TicketLogAddress wacom
var TicketLogAddress = HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe")

// SystemAssetID wacom
var SystemAssetID = HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

var NotationKeyAddress = HexToAddress("0xfffffffffffffffffffffffffffffffffffffffd")
var AssetKeyAddress = HexToAddress("0xfffffffffffffffffffffffffffffffffffffffc")
var TicketKeyAddress = HexToAddress("0xfffffffffffffffffffffffffffffffffffffffb")
var SwapKeyAddress = HexToAddress("0xfffffffffffffffffffffffffffffffffffffffa")

var (
	// NotationKey wacom
	NotationKey = []byte{0x01}
	// AssetKey wacom
	AssetKey = []byte{0x02}
	// TicketKey wacom
	TicketKey = []byte{0x03}
	// SwapKey wacom
	SwapKey = []byte{0x06} // 4 was the old
	// Auto buy tickets
	AutoBuyTicket     = false
	AutoBuyTicketChan = make(chan int, 10)
)

// FSNCallFunc wacom
type FSNCallFunc uint8

const (
	// GenNotationFunc wacom
	GenNotationFunc = iota
	// GenAssetFunc wacom
	GenAssetFunc
	// SendAssetFunc wacom
	SendAssetFunc
	// TimeLockFunc wacom
	TimeLockFunc
	// BuyTicketFunc wacom
	BuyTicketFunc
	// OldAssetValueChangeFunc wacom
	OldAssetValueChangeFunc
	// MakeSwapFunc wacom
	MakeSwapFunc
	// RecallSwapFunc wacom
	RecallSwapFunc
	// TakeSwapFunc wacom
	TakeSwapFunc
	// EmptyFunc wacom
	EmptyFunc
	// MakeSwapFuncExt wacom
	MakeSwapFuncExt
	// TakeSwapFuncExt wacom
	TakeSwapFuncExt
	// AssetValueChangeFunc wacom
	AssetValueChangeFunc
	// TransferNotationFunc wacom
	TransferNotationFunc
)

// ParseBig256 parses s as a 256 bit integer in decimal or hexadecimal syntax.
// Leading zeros are accepted. The empty string parses as zero.
func ParseBig256(s string) (*big.Int, bool) {
	if s == "" {
		return new(big.Int), true
	}
	var bigint *big.Int
	var ok bool
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		bigint, ok = new(big.Int).SetString(s[2:], 16)
	} else {
		bigint, ok = new(big.Int).SetString(s, 10)
	}
	if ok && bigint.BitLen() > 256 {
		bigint, ok = nil, false
	}
	return bigint, ok
}

// TicketPrice  place holder for ticket price
func TicketPrice(blocknumber *big.Int) *big.Int {
	oneFSN := big.NewInt(1000000000000000000)
	return new(big.Int).Mul(big.NewInt(5000), oneFSN)
}

// FSNCallParam wacom
type FSNCallParam struct {
	Func FSNCallFunc
	Data []byte
}

// GenAssetParam wacom
type GenAssetParam struct {
	Name        string
	Symbol      string
	Decimals    uint8
	Total       *big.Int `json:",string"`
	CanChange   bool
	Description string
}

// TransferNotationParam wacom
type TransferNotationParam struct {
	Notation        uint64
	ToAddress      Address
}

// BuyTicketParam wacom
type BuyTicketParam struct {
	Start uint64
	End   uint64
}

// SendAssetParam wacom
type SendAssetParam struct {
	AssetID Hash
	To      Address
	Value   *big.Int `json:",string"`
}

// AssetValueChangeParam wacom
type AssetValueChangeParam struct {
	AssetID Hash
	To      Address
	Value   *big.Int `json:",string"`
	IsInc   bool
}

// AssetValueChangeExParam wacom
type AssetValueChangeExParam struct {
	AssetID     Hash
	To          Address
	Value       *big.Int `json:",string"`
	IsInc       bool
	TransacData string
}

// TimeLockParam wacom
type TimeLockParam struct {
	Type      TimeLockType
	AssetID   Hash
	To        Address
	StartTime uint64
	EndTime   uint64
	Value     *big.Int `json:",string"`
}

// MakeSwapParam wacom
type MakeSwapParam struct {
	FromAssetID   Hash
	FromStartTime uint64
	FromEndTime   uint64
	MinFromAmount *big.Int `json:",string"`
	ToAssetID     Hash
	ToStartTime   uint64
	ToEndTime     uint64
	MinToAmount   *big.Int `json:",string"`
	SwapSize      *big.Int `json:",string"`
	Targes        []Address
	Time          *big.Int
	Description   string
}

// RecallSwapParam wacom
type RecallSwapParam struct {
	SwapID Hash
}

// TakeSwapParam wacom
type TakeSwapParam struct {
	SwapID Hash
	Size   *big.Int `json:",string"`
}

// ToBytes wacom
func (p *FSNCallParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToBytes wacom
func (p *GenAssetParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToBytes wacom
func (p *TransferNotationParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToBytes wacom
func (p *SendAssetParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToBytes wacom
func (p *TimeLockParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToBytes wacom
func (p *BuyTicketParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToBytes wacom
func (p *AssetValueChangeParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToBytes wacom
func (p *AssetValueChangeExParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToBytes wacom
func (p *MakeSwapParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToBytes wacom
func (p *RecallSwapParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToBytes wacom
func (p *TakeSwapParam) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(p)
}

// ToAsset wacom
func (p *GenAssetParam) ToAsset() Asset {
	return Asset{
		Name:        p.Name,
		Symbol:      p.Symbol,
		Decimals:    p.Decimals,
		Total:       p.Total,
		CanChange:   p.CanChange,
		Description: p.Description,
	}
}

// Asset wacom
type Asset struct {
	ID          Hash
	Owner       Address
	Name        string
	Symbol      string
	Decimals    uint8
	Total       *big.Int `json:",string"`
	CanChange   bool
	Description string
}

func (u *Asset) DeepCopy() Asset {
	total := *u.Total
	return Asset{
		ID:        u.ID,
		Owner:     u.Owner,
		Name:      u.Name,
		Symbol:    u.Symbol,
		Decimals:  u.Decimals,
		Total:     &total,
		CanChange: u.CanChange,
	}
}

func (u *Asset) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID          Hash
		Owner       Address
		Name        string
		Symbol      string
		Decimals    uint8
		Total       string
		CanChange   bool
		Description string
	}{
		ID:          u.ID,
		Owner:       u.Owner,
		Name:        u.Name,
		Symbol:      u.Symbol,
		Decimals:    u.Decimals,
		Total:       u.Total.String(),
		CanChange:   u.CanChange,
		Description: u.Description,
	})
}

// SystemAsset wacom
var SystemAsset = Asset{
	Name:        "Fusion",
	Symbol:      "FSN",
	Decimals:    18,
	Total:       new(big.Int).Mul(big.NewInt(81920000), big.NewInt(1000000000000000000)),
	ID:          SystemAssetID,
	Description: "https://fusion.org",
}

// Ticket wacom
type Ticket struct {
	ID         Hash
	Owner      Address
	Height     *big.Int `json:",string"`
	StartTime  uint64
	ExpireTime uint64
	Value      *big.Int `json:",string"`
	weight     *big.Int `json:",string"`
}

func (t *Ticket) DeepCopy() Ticket {
	height := *t.Height
	value := *t.Value
	w := t.Weight()
	wt := new(big.Int)
	if w != nil {
		wt = &(*w)
	}
	return Ticket{
		ID:         t.ID,
		Owner:      t.Owner,
		Height:     &height,
		StartTime:  t.StartTime,
		ExpireTime: t.ExpireTime,
		Value:      &value,
		weight:     wt,
	}
}

// SetWeight wacom
func (t *Ticket) SetWeight(value *big.Int) {
	t.weight = value
}

// Weight wacom
func (t *Ticket) Weight() *big.Int {
	return t.weight
}

type TicketStruct struct {
	Hash
	Ticket
}

type TicketStructSlice []TicketStruct
type TicketSlice []Ticket

func (t *Ticket) toTicketStruct() TicketStruct {
	return TicketStruct{t.ID, *t}
}

func (s TicketStructSlice) ToTicketSlice() TicketSlice {
	r := make(TicketSlice, 0, len(s))
	for _, t := range s {
		r = append(r, t.Ticket)
	}
	return r
}

func (s TicketSlice) ToTicketStructSlice() TicketStructSlice {
	r := make(TicketStructSlice, 0, len(s))
	for _, t := range s {
		r = append(r, t.toTicketStruct())
	}
	return r
}

func (s TicketSlice) Len() int {
	return len(s)
}

func (s TicketSlice) Less(i, j int) bool {
	a, _ := new(big.Int).SetString(s[i].ID.Hex(), 0)
	b, _ := new(big.Int).SetString(s[j].ID.Hex(), 0)
	return a.Cmp(b) < 0
}

func (s TicketSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s TicketSlice) ToMap() map[Hash]Ticket {
	r := make(map[Hash]Ticket, len(s))
	for _, t := range s {
		r[t.ID] = t
	}
	return r
}

func (s TicketSlice) DeepCopy() TicketSlice {
	if s == nil || len(s) == 0 {
		return nil
	}
	r := make(TicketSlice, 0, len(s))
	for _, t := range s {
		r = append(r, t)
	}
	return r
}

func (s TicketSlice) Get(tid Hash) (*Ticket, bool) {
	for _, t := range s {
		if t.ID == tid {
			return &t, true
		}
	}
	return nil, false
}

func (s TicketSlice) Add(ticket *Ticket) TicketSlice {
	s = append(s, *ticket)
	return s
}

func (s TicketSlice) Delete(tid Hash) TicketSlice {
	for i, t := range s {
		if t.ID == tid {
			s = append(s[:i], s[i+1:]...)
			break
		}
	}
	return s
}

// Swap wacom
type Swap struct {
	ID            Hash
	Owner         Address
	FromAssetID   Hash
	FromStartTime uint64
	FromEndTime   uint64
	MinFromAmount *big.Int `json:",string"`
	ToAssetID     Hash
	ToStartTime   uint64
	ToEndTime     uint64
	MinToAmount   *big.Int `json:",string"`
	SwapSize      *big.Int `json:",string"`
	Targes        []Address
	Time          *big.Int // Provides information for TIME
	Description   string
}

func (s *Swap) DeepCopy() Swap {
	minFromAmount := *s.MinFromAmount
	minToAmount := *s.MinToAmount
	swapSize := *s.SwapSize
	swapTime := *s.Time
	targets := make([]Address, len(s.Targes))
	copy(targets, s.Targes)

	return Swap{
		ID:            s.ID,
		Owner:         s.Owner,
		FromAssetID:   s.FromAssetID,
		FromStartTime: s.FromStartTime,
		FromEndTime:   s.FromEndTime,
		MinFromAmount: &minFromAmount,
		ToAssetID:     s.ToAssetID,
		ToStartTime:   s.ToStartTime,
		ToEndTime:     s.ToEndTime,
		MinToAmount:   &minToAmount,
		SwapSize:      &swapSize,
		Targes:        targets,
		Time:          &swapTime,
		Description:   s.Description,
	}
}

// KeyValue wacom
type KeyValue struct {
	Key   string
	Value interface{}
}

// NewKeyValue wacom
func NewKeyValue(name string, v interface{}) *KeyValue {

	return &KeyValue{Key: name, Value: v}

}

func (p *FSNCallParam) Check(blockNumber *big.Int) error {
	return nil
}

func (p *GenAssetParam) Check(blockNumber *big.Int) error {
	if len(p.Name) == 0 || len(p.Symbol) == 0 || p.Total == nil || p.Total.Cmp(Big0) < 0 {
		return fmt.Errorf("GenAssetFunc name, symbol and total must be set")
	}
	if p.Decimals > 18 {
		return fmt.Errorf("GenAssetFunc decimals must be between 0 and 18")
	}
	if len(p.Description) > 1024 {
		return fmt.Errorf("GenAsset description length is greater than 1024 chars")
	}
	if len(p.Name) > 128 {
		return fmt.Errorf("GenAsset name length is greater than 128 chars")
	}
	if len(p.Symbol) > 64 {
		return fmt.Errorf("GenAsset symbol length is greater than 64 chars")

	}
	return nil
}

func (p *SendAssetParam) Check(blockNumber *big.Int) error {
	if p.Value == nil || p.Value.Cmp(Big0) <= 0 {
		return fmt.Errorf("Value must be set and greater than 0")
	}
	if p.To == (Address{}) {
		return fmt.Errorf("receiver address must be set and not zero address")
	}
	return nil
}

func (p *TimeLockParam) Check(blockNumber *big.Int, timestamp uint64) error {

	if p.Value == nil || p.Value.Cmp(Big0) <= 0 {
		return fmt.Errorf("Value must be set and greater than 0")
	}
	if p.StartTime > p.EndTime {
		return fmt.Errorf("StartTime must be less than or equal to EndTime")
	}
	if p.EndTime < timestamp {
		return fmt.Errorf("EndTime must be greater than latest block time")
	}

	return nil
}

func (p *BuyTicketParam) Check(blockNumber *big.Int, timestamp uint64, adjust int64) error {
	start, end := p.Start, p.End
	// check lifetime too short ticket
	if end <= start || end < start+30*24*3600 {
		return fmt.Errorf("BuyTicket end must be lower than start + 1 month")
	}
	if timestamp != 0 {
		// check future ticket
		if start > timestamp+3*3600 {
			return fmt.Errorf("BuyTicket start must be lower than latest block time + 3 hour")
		}
		// check expired soon ticket
		if end < timestamp+uint64(7*24*3600+adjust) {
			return fmt.Errorf("BuyTicket end must be greater than latest block time + 1 week")
		}
	}
	return nil
}

func (p *AssetValueChangeParam) Check(blockNumber *big.Int) error {
	if p.Value == nil || p.Value.Cmp(Big0) <= 0 {
		return fmt.Errorf("Value must be set and greater than 0")
	}
	return nil
}

func (p *AssetValueChangeExParam) Check(blockNumber *big.Int) error {
	if p.Value == nil || p.Value.Cmp(Big0) <= 0 {
		return fmt.Errorf("Value must be set and greater than 0")
	}
	if len(p.TransacData) > 256 {
		return fmt.Errorf("TransacData must not be greater than 256")
	}
	return nil
}

func (p *MakeSwapParam) Check(blockNumber *big.Int, timestamp uint64) error {
	if p.MinFromAmount == nil || p.MinFromAmount.Cmp(Big0) <= 0 ||
		p.MinToAmount == nil || p.MinToAmount.Cmp(Big0) <= 0 ||
		p.SwapSize == nil || p.SwapSize.Cmp(Big0) <= 0 {
		return fmt.Errorf("MinFromAmount,MinToAmount and SwapSize must be ge 1")
	}
	if len(p.Description) > 1024 {
		return fmt.Errorf("MakeSwap description length is greater than 1024 chars")
	}
	total := new(big.Int).Mul(p.MinFromAmount, p.SwapSize)
	if total.Cmp(Big0) <= 0 {
		return fmt.Errorf("size * MinFromAmount too large")
	}
	
	if p.FromStartTime > p.FromEndTime {
		return fmt.Errorf("MakeSwap FromStartTime > FromEndTime")
	}
	if p.ToStartTime > p.ToEndTime {
		return fmt.Errorf("MakeSwap ToStartTime > ToEndTime")
	}


	if p.FromEndTime <= timestamp {
		return fmt.Errorf("MakeSwap FromEndTime <= latest blockTime")
	}
	if p.ToEndTime <= timestamp {
		return fmt.Errorf("MakeSwap ToEndTime <= latest blockTime")
	}

	return nil
}

func (p *RecallSwapParam) Check(blockNumber *big.Int, swap *Swap) error {
	if swap.MinFromAmount == nil || swap.MinFromAmount.Cmp(Big0) <= 0 {
		return fmt.Errorf("swap illegal: MinFromAmount must be set and greater than 0")
	}
	if swap.SwapSize == nil || swap.SwapSize.Cmp(Big0) <= 0 {
		return fmt.Errorf("swap illegal: SwapSize must be set and greater than 0")
	}
	total := new(big.Int).Mul(swap.MinFromAmount, swap.SwapSize)
	if total.Cmp(Big0) <= 0 {
		return fmt.Errorf("size * minFromAmount too large")
	}
	return nil
}

func (p *TakeSwapParam) Check(blockNumber *big.Int, swap *Swap, timestamp uint64) error {
	if p.Size == nil || p.Size.Cmp(Big0) <= 0 ||
		swap.SwapSize == nil || p.Size.Cmp(swap.SwapSize) > 0 {
		
		return fmt.Errorf("Size must be ge 1 and le Swapsize")
	}
	if swap.MinFromAmount == nil || swap.MinFromAmount.Cmp(Big0) <= 0 {
		return fmt.Errorf("MinFromAmount less than  equal to zero")
	}
	if swap.MinToAmount == nil || swap.MinToAmount.Cmp(Big0) <= 0 {
		return fmt.Errorf("MinToAmount less than  equal to zero")
	}

	fromTotal := new(big.Int).Mul(swap.MinFromAmount, p.Size)
	if fromTotal.Cmp(Big0) <= 0 {
		return fmt.Errorf("fromTotal less than  equal to zero")
	}

	toTotal := new(big.Int).Mul(swap.MinToAmount, p.Size)
	if toTotal.Cmp(Big0) <= 0 {
		return fmt.Errorf("toTotal less than  equal to zero")
	}
	
	if swap.FromEndTime <= timestamp {
		return fmt.Errorf("swap expired: FromEndTime <= latest blockTime")
	}
	if swap.ToEndTime <= timestamp {
		return fmt.Errorf("swap expired: ToEndTime <= latest blockTime")
	}

	return nil
}
