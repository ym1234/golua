package stringlib

import (
	"bytes"
	"encoding/binary"
	"math"

	rt "github.com/arnodel/golua/runtime"
)

type packer struct {
	format       string           // Specifies the packing format
	i            int              // Current index in the format string
	byteOrder    binary.ByteOrder // Current byteOrder of outputting numbers
	maxAlignment uint             // Current max alignment (used in pack.align())
	err          error            // if non-nil, the error encountered while packing
	optSize      uint             // Value of current option size
	values       []rt.Value       // Lua values to be packed
	j            int              // Current index in the values above
	val          rt.Value         // Current value
	intVal       int64            // Current integral value (if applicable)
	floatVal     float64          // Current floating point value (if applicable)
	strVal       string           // Current string value (if applicable)
	w            bytes.Buffer     // Where the output is written
	alignOnly    bool             // true after "X" option is parsed
}

func PackValues(format string, values []rt.Value) (string, error) {
	p := &packer{
		format:       format,
		byteOrder:    nativeEndian,
		maxAlignment: defaultMaxAlignement,
		values:       values,
	}
	for p.err == nil && p.hasNext() {
		switch p.nextOption() {
		case '<':
			p.byteOrder = binary.LittleEndian
		case '>':
			p.byteOrder = binary.BigEndian
		case '=':
			p.byteOrder = nativeEndian
		case '!':
			if p.smallOptSize(defaultMaxAlignement) {
				p.maxAlignment = p.optSize
			}
		case 'b':
			_ = p.align(0) &&
				p.nextIntValue() &&
				p.checkBounds(math.MinInt8, math.MaxInt8) &&
				p.write(int8(p.intVal))
		case 'B':
			_ = p.align(0) &&
				p.nextIntValue() &&
				p.checkBounds(0, math.MaxUint8) &&
				p.write(uint8(p.intVal))
		case 'h':
			_ = p.align(2) &&
				p.nextIntValue() &&
				p.checkBounds(math.MinInt16, math.MaxInt16) &&
				p.write(int16(p.intVal))
		case 'H':
			_ = p.align(2) &&
				p.nextIntValue() &&
				p.checkBounds(0, math.MaxUint16) &&
				p.write(uint16(p.intVal))
		case 'l', 'j':
			_ = p.align(8) &&
				p.nextIntValue() &&
				p.write(p.intVal)
		case 'L', 'J', 'T':
			_ = p.align(8) &&
				p.nextIntValue() &&
				p.checkBounds(0, math.MaxInt64) &&
				p.write(uint64(p.intVal))
		case 'i':
			_ = p.smallOptSize(4) &&
				p.align(p.optSize) &&
				p.nextIntValue() &&
				p.packInt()
		case 'I':
			_ = p.smallOptSize(4) &&
				p.align(p.optSize) &&
				p.nextIntValue() &&
				p.packUint()
		case 'f':
			_ = p.align(4) &&
				p.nextFloatValue() &&
				p.checkFloatSize(math.MaxFloat32) &&
				p.write(float32(p.floatVal))
		case 'd', 'n':
			_ = p.align(8) &&
				p.nextFloatValue() &&
				p.write(p.floatVal)
		case 'c':
			_ = p.align(0) &&
				p.mustGetOptSize() &&
				p.nextStringValue() &&
				p.writeStr(p.optSize)
		case 'z':
			_ = p.align(0) &&
				p.nextStringValue() &&
				p.writeStr(0) &&
				p.writeByte(0)
		case 's':
			_ = p.smallOptSize(8) &&
				p.align(p.optSize) &&
				p.nextStringValue() &&
				p.packUint() &&
				p.writeStr(0)
		case 'x':
			_ = p.align(0) &&
				p.writeByte(0)
		case 'X':
			p.alignOnly = true
		case ' ':
			// ignored
		default:
			p.err = errBadFormatString
		}
		if p.err != nil {
			return "", p.err
		}
	}
	if p.alignOnly {
		return "", errExpectedOption
	}
	return p.w.String(), nil
}

func (p *packer) hasNext() bool {
	return p.i < len(p.format)
}

func (p *packer) nextOption() byte {
	opt := p.format[p.i]
	p.i++
	return opt
}

func (p *packer) smallOptSize(defaultSize uint) bool {
	p.getOptSize()
	if p.optSize > 16 {
		p.err = errBadOptionArg
		return false
	} else if p.optSize == 0 {
		if defaultSize == 0 {
			p.err = errMissingSize
			return false
		}
		p.optSize = defaultSize
	}
	return true
}

func (p *packer) getOptSize() bool {
	var n uint
	ok := false
	for ; p.i < len(p.format); p.i++ {
		c := p.format[p.i]
		if c >= '0' && c <= '9' {
			ok = true
			n = n*10 + uint(c-'0')
		} else {
			break
		}
	}
	p.optSize = n
	return ok
}

func (p *packer) mustGetOptSize() bool {
	ok := p.getOptSize()
	if !ok {
		p.err = errMissingSize
	}
	return ok
}

func (p *packer) nextValue() bool {
	if len(p.values) > p.j {
		p.val = p.values[p.j]
		p.j++
		return true
	}
	p.err = errNotEnoughValues
	return false
}

func (p *packer) nextIntValue() bool {
	if !p.nextValue() {
		return false
	}
	n, tp := rt.ToInt(p.val)
	if tp != rt.IsInt {
		p.err = errBadType
		return false
	}
	p.intVal = int64(n)
	return true
}

func (p *packer) nextFloatValue() bool {
	if !p.nextValue() {
		return false
	}
	f, ok := rt.ToFloat(p.val)
	if !ok {
		p.err = errBadType
		return false
	}
	p.floatVal = float64(f)
	return true
}

func (p *packer) nextStringValue() bool {
	if !p.nextValue() {
		return false
	}
	s, ok := rt.AsString(p.val)
	if !ok {
		p.err = errBadType
		return false
	}
	p.strVal = string(s)
	p.intVal = int64(len(s))
	return true
}

func (p *packer) checkBounds(min, max int64) bool {
	ok := p.intVal >= min && p.intVal <= max
	if !ok {
		p.err = errOutOfBounds
	}
	return ok
}

func (p *packer) checkFloatSize(max float64) bool {
	ok := p.floatVal >= -max && p.floatVal <= max
	if !ok {
		p.err = errOutOfBounds
	}
	return ok
}

func (p *packer) writeByte(b byte) bool {
	p.w.WriteByte(b)
	return true
}

func (p *packer) write(x interface{}) bool {
	p.err = binary.Write(&p.w, p.byteOrder, x)
	return p.err == nil
}

func (p *packer) writeStr(maxLen uint) bool {
	diff := 0
	if maxLen > 0 {
		diff = int(maxLen) - len(p.strVal)
	}
	if diff < 0 {
		p.err = errOutOfBounds
		return false
	}
	p.w.Write([]byte(p.strVal))
	if diff > 0 {
		p.fill(uint(diff), 0)
	}
	return true
}

func (p *packer) align(n uint) bool {
	if n != 0 {
		if n > p.maxAlignment {
			n = p.maxAlignment
		}
		if (n-1)&n != 0 { // (n-1)&n == 0 iff n is a power of 2 (or 0)
			p.err = errBadAlignment
			return false
		}
		if r := uint(p.w.Len()) % n; r != 0 {
			p.fill(n-r, 0)
		}
	}
	if p.alignOnly {
		p.alignOnly = false
		return false
	}
	return true
}

func (p *packer) fill(n uint, c byte) {
	for ; n > 0; n-- {
		p.w.WriteByte(c)
	}
}

func (p *packer) packInt() bool {
	switch n := p.optSize; {
	case n == 4:
		// It's an int32
		return p.checkBounds(math.MinInt32, math.MaxInt32) && p.write(int32(p.intVal))
	case n == 8:
		// It's an int64
		return p.write(p.intVal)
	case n >= 8:
		// Pad to make up the length
		var fill byte
		if p.intVal < 0 {
			fill = 255
		}
		if p.byteOrder == binary.BigEndian {
			p.fill(n-8, fill)
		}
		if !p.write(p.intVal) {
			return false
		}
		if p.byteOrder == binary.LittleEndian {
			p.fill(n-8, fill)
		}
	default:
		// n < 8 so truncate
		max := int64(1) << (n<<3 - 1)
		if !p.checkBounds(-max, max-1) {
			return false
		}
		var ww bytes.Buffer
		if err := binary.Write(&ww, p.byteOrder, p.intVal); err != nil {
			p.err = err
			return false
		}
		switch p.byteOrder {
		case binary.LittleEndian:
			p.w.Write(ww.Bytes()[:n])
		default:
			p.w.Write(ww.Bytes()[8-n:])
		}
	}
	return true
}

func (p *packer) packUint() bool {
	switch n := p.optSize; {
	case n == 4:
		// It's an uint32
		return p.checkBounds(0, math.MaxUint32) && p.write(uint32(p.intVal))
	case n == 8:
		// It's an uint64
		return p.checkBounds(0, math.MaxInt64) && p.write(uint64(p.intVal))
	case n > 8:
		// Pad to make up the length
		if p.byteOrder == binary.BigEndian {
			p.fill(n-8, 0)
		}
		if !p.write(uint64(p.intVal)) {
			return false
		}
		if p.byteOrder == binary.LittleEndian {
			p.fill(n-8, 0)
		}
	default:
		// n < 8 so truncate
		max := int64(1) << (n << 3)
		if !p.checkBounds(0, max-1) {
			return false
		}
		var ww bytes.Buffer
		if err := binary.Write(&ww, p.byteOrder, uint64(p.intVal)); err != nil {
			p.err = err
			return false
		}
		switch p.byteOrder {
		case binary.LittleEndian:
			p.w.Write(ww.Bytes()[:n])
		default:
			p.w.Write(ww.Bytes()[8-n:])
		}
	}
	return true
}
