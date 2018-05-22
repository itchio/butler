package wizparser

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/itchio/wizardry/wizardry"
)

// Spellbook contains a set of rules - at least one "" page, potentially others
type Spellbook map[string][]Rule

// AddRule appends a rule to the spellbook on the given page
func (sb Spellbook) AddRule(page string, rule Rule) {
	sb[page] = append(sb[page], rule)
}

// Rule is a single magic rule
type Rule struct {
	Line        string
	Level       int
	Offset      Offset
	Kind        Kind
	Description []byte
}

func (r Rule) String() string {
	return fmt.Sprintf("%s%s    %s    %s",
		strings.Repeat(">", r.Level),
		r.Offset, r.Kind, r.Description)
}

func (o Offset) String() string {
	s := ""

	switch o.OffsetType {
	case OffsetTypeDirect:
		s = fmt.Sprintf("0x%x", o.Direct)
	case OffsetTypeIndirect:
		s = "("
		indirect := o.Indirect
		if indirect.IsRelative {
			s += "&"
		}

		s += fmt.Sprintf("0x%x", indirect.OffsetAddress)
		s += "."

		switch indirect.ByteWidth {
		case 1:
			s += "byte"
		case 2:
			s += "short"
		case 4:
			s += "long"
		case 8:
			s += "quad"
		}
		if indirect.Endianness == LittleEndian {
			s += "le"
		} else {
			s += "be"
		}

		switch indirect.OffsetAdjustmentType {
		case AdjustmentAdd:
			s += "+"
		case AdjustmentSub:
			s += "-"
		case AdjustmentMul:
			s += "*"
		case AdjustmentDiv:
			s += "/"
		}

		if indirect.OffsetAdjustmentType != AdjustmentNone {
			if indirect.OffsetAdjustmentIsRelative {
				s += "("
			}
			s += fmt.Sprintf("%d", indirect.OffsetAdjustmentValue)
			if indirect.OffsetAdjustmentIsRelative {
				s += ")"
			}
		}

		s += ")"
	}

	if o.IsRelative {
		s = "&" + s
	}
	return s
}

// Equals returns true if and only if a and b point to exactly the same offset
func (o Offset) Equals(b Offset) bool {
	a := o

	if a.IsRelative != b.IsRelative {
		return false
	}

	if a.OffsetType != b.OffsetType {
		return false
	}

	if a.OffsetType == OffsetTypeDirect {
		return a.Direct == b.Direct
	}

	ai := a.Indirect
	bi := b.Indirect

	if ai.OffsetAddress != bi.OffsetAddress {
		return false
	}

	if ai.OffsetAdjustmentType != bi.OffsetAdjustmentType {
		return false
	}

	if ai.OffsetAdjustmentIsRelative != bi.OffsetAdjustmentIsRelative {
		return false
	}

	if ai.OffsetAdjustmentValue != bi.OffsetAdjustmentValue {
		return false
	}

	if ai.Endianness != bi.Endianness {
		return false
	}

	if ai.IsRelative != bi.IsRelative {
		return false
	}

	if ai.ByteWidth != bi.ByteWidth {
		return false
	}

	return true
}

func (k Kind) String() string {
	switch k.Family {
	case KindFamilyInteger:
		ik, _ := k.Data.(*IntegerKind)
		s := ""
		if !ik.Signed {
			s += "u"
		}
		switch ik.ByteWidth {
		case 1:
			s += "byte"
		case 2:
			s += "short"
		case 4:
			s += "long"
		case 8:
			s += "quad"
		}
		if ik.Endianness == LittleEndian {
			s += "le"
		} else {
			s += "be"
		}
		s += "    "
		s += fmt.Sprintf("%x", ik.Value)
		if ik.DoAnd {
			s += fmt.Sprintf("&0x%x", ik.AndValue)
		}
		return s
	case KindFamilyString:
		sk, _ := k.Data.(*StringKind)
		return fmt.Sprintf("string    %s", strconv.Quote(string(sk.Value)))
	case KindFamilySearch:
		sk, _ := k.Data.(*SearchKind)
		return fmt.Sprintf("search/0x%x    %s", sk.MaxLen, strconv.Quote(string(sk.Value)))
	case KindFamilyDefault:
		return "default"
	case KindFamilyClear:
		return "clear"
	case KindFamilyUse:
		uk, _ := k.Data.(*UseKind)
		s := "use   "
		if uk.SwapEndian {
			s += "\\^"
		}
		s += uk.Page
		return s
	case KindFamilySwitch:
		sk, _ := k.Data.(*SwitchKind)
		return fmt.Sprintf("switch with %d cases", len(sk.Cases))
	default:
		return fmt.Sprintf("kind family %d", k.Family)
	}
}

// Endianness describes the order in which a multi-byte number is stored
type Endianness int

// ByteOrder translates our in-house Endianness constant into a binary.ByteOrder decoder
func (en Endianness) ByteOrder() binary.ByteOrder {
	if en == BigEndian {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// Swapped returns LittleEndian if you give it BigEndian, and vice versa
func (en Endianness) Swapped() Endianness {
	if en == BigEndian {
		return LittleEndian
	}
	return BigEndian
}

// MaybeSwapped returns swapped endianness if swap is true
func (en Endianness) MaybeSwapped(swap bool) Endianness {
	if !swap {
		return en
	}
	return en.Swapped()
}

func (en Endianness) String() string {
	if en == BigEndian {
		return "big-endian"
	}
	return "little-endian"
}

const (
	// LittleEndian numbers are stored with the least significant byte first
	LittleEndian Endianness = iota
	// BigEndian numbers are stored with the most significant byte first
	BigEndian
)

// Kind describes the type of tests a magic rule performs
type Kind struct {
	Family KindFamily
	Data   interface{}
}

// IntegerKind describes how to perform a test on an integer
type IntegerKind struct {
	ByteWidth       int
	Endianness      Endianness
	Signed          bool
	DoAnd           bool
	AndValue        uint64
	IntegerTest     IntegerTest
	Value           int64
	MatchAny        bool
	AdjustmentType  Adjustment
	AdjustmentValue int64
}

type SwitchKind struct {
	ByteWidth  int
	Endianness Endianness
	Signed     bool
	Cases      []*SwitchCase
}

type SwitchCase struct {
	Value       int64
	Description []byte
}

// IntegerTest describes which comparison to perform on an integer
type IntegerTest int

const (
	// IntegerTestEqual tests that two integers have the same value
	IntegerTestEqual IntegerTest = iota
	// IntegerTestNotEqual tests that two integers have different values
	IntegerTestNotEqual
	// IntegerTestLessThan tests that one integer is less than the other
	IntegerTestLessThan
	// IntegerTestGreaterThan tests that one integer is greater than the other
	IntegerTestGreaterThan
	// IntegerTestAnd tests that all the bits in the pattern are set
	IntegerTestAnd
)

// StringKind describes how to match a string pattern
type StringKind struct {
	Value  []byte
	Negate bool
	Flags  wizardry.StringTestFlags
}

// SearchKind describes how to look for a fixed pattern
type SearchKind struct {
	Value  []byte
	MaxLen int64
}

// KindFamily groups tests in families (all integer tests, for example)
type KindFamily int

const (
	// KindFamilyInteger tests numbers for equality, inequality, etc.
	KindFamilyInteger KindFamily = iota
	// KindFamilyString looks for a string, with casing and whitespace rules
	KindFamilyString
	// KindFamilySearch looks for a precise string in a slice of the target
	KindFamilySearch
	// KindFamilyDefault succeeds if no tests succeeded before on that level
	KindFamilyDefault
	// KindFamilyClear resets the matched flag for that level
	KindFamilyClear
	// KindFamilyName always succeeds
	KindFamilyName
	// KindFamilyUse acts like a subroutine call, to peruse another page of rules
	KindFamilyUse

	// Compiler additions begin

	// KindFamilySwitch is a series of merged KindFamilyInteger
	KindFamilySwitch
)

// Offset describes where to look to compare something
type Offset struct {
	OffsetType OffsetType
	IsRelative bool
	Direct     int64
	Indirect   *IndirectOffset
}

// OffsetType describes whether an offset is direct or indirect
type OffsetType int

const (
	// OffsetTypeIndirect is an offset read from somewhere in a file
	OffsetTypeIndirect OffsetType = iota
	// OffsetTypeDirect is an offset directly specified by the magic
	OffsetTypeDirect
)

// IndirectOffset indicates where to look in a file to find the real offset
type IndirectOffset struct {
	IsRelative                 bool
	ByteWidth                  int
	Endianness                 Endianness
	OffsetAddress              int64
	OffsetAdjustmentType       Adjustment
	OffsetAdjustmentIsRelative bool
	OffsetAdjustmentValue      int64
}

// Adjustment describes which operation to apply to an offset
type Adjustment int

const (
	// AdjustmentNone is a no-op
	AdjustmentNone Adjustment = iota
	// AdjustmentAdd adds a value
	AdjustmentAdd
	// AdjustmentSub subtracts a value
	AdjustmentSub
	// AdjustmentMul multiplies by a value
	AdjustmentMul
	// AdjustmentDiv divides by a value
	AdjustmentDiv
)

// UseKind describes which page of the spellbook to use, and whether or not to swap endianness
type UseKind struct {
	SwapEndian bool
	Page       string
}
