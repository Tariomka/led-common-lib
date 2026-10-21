package lightshow

import (
	"strconv"

	"github.com/Tariomka/led-common-lib/pkg/lightshow/blob"
)

// Script VM opcodes (op << 16 | int16 argument).
const (
	opLdrSystemA     = 0
	opLdrLocalA      = 2
	opLdrLocalAddA   = 4
	opLdrStackBeginA = 5
	opLdrStackEndA   = 6
	opLdrSystemB     = 7
	opLdrLocalB      = 9
	opLdrLocalAddB   = 11
	opLdrStackBeginB = 12
	opLdrStackEndB   = 13
	opStrSystemA     = 14
	opStrStackBeginA = 15
	opStrStackEndA   = 16
	opSwapAB         = 17
	opPush           = 18
	opPop            = 19
	opPushA          = 20
	opPopA           = 21
	opGoto           = 22
	opCallSystem     = 24
	opCallLocal      = 26
	opRetFunction    = 27
	opRetEvent       = 28
	opCreateRef      = 29
	opDestroyRef     = 30
	opMinus          = 32
	opNot            = 33
	opLogicalNot     = 34
	opInc            = 35
	opDec            = 36
	opAdd            = 37
	opSub            = 38
	opMul            = 39
	opDiv            = 40
	opMod            = 41
	opShl            = 42
	opShr            = 43
	opAnd            = 44
	opOr             = 45
	opXor            = 46
	opEq             = 47
	opNe             = 48
	opLogicalAnd     = 49
	opLogicalOr      = 50
	opGt             = 51
	opGe             = 52
	opLt             = 53
	opLe             = 54
	opCopy           = 55
	opCopyStr        = 56
	opCat            = 57
	opJz             = 58
	opJnz            = 59
)

// Script system functions (CALL_SYSTEM index)
const (
	sysTraceN = iota
	sysString
	sysWait
	sysClean
	sysPoint
	sysLine
	sysGraph
	sysGetIndex
	sysGetRandomColor
	sysGetTick
	sysGetAdc
	sysGetRealAdc
	sysEnabledKey
	sysGetKey
)

// Virtual memory layout. Pointers held in A/B are byte addresses in this space.
const (
	stackWords   = 200
	heapWords    = 1024
	tempBuffers  = 5
	tempSize     = 128
	maxSteps     = 20000 // instructions per Step before yielding
	stackBase    = 0
	heapBase     = stackBase + 4*stackWords
	tempBase     = heapBase + 4*heapWords
	bodyBase     = tempBase + tempBuffers*tempSize
	sysVarsCount = 0
)

type vm struct {
	player  *Player
	assets  *assets
	mem     []byte
	bodyLen int
	a, b    int32
	pc      int // byte address
	sp      int // byte address
	heap    int
	temp    int
	phase   int // 0 = init, 1 = OnLoop, 2 = done
	onLoop  int
	failed  bool
}

func newVM(p *Player, body string, a *assets) *vm {
	v := &vm{player: p, assets: a, mem: make([]byte, bodyBase+len(body)), bodyLen: len(body)}
	copy(v.mem[bodyBase:], body)
	v.sp, v.heap = stackBase, heapBase
	if len(body) >= 8 {
		v.pc = bodyBase + 4*int(blob.U32(body, 0))
		v.onLoop = int(blob.U32(body, 4))
	} else {
		v.phase = 2
	}
	return v
}

func (this *vm) word(addr int) int32 {
	if addr < 0 || addr+4 > len(this.mem) {
		this.fail("bad address")
		return 0
	}

	return int32(uint32(this.mem[addr]) | uint32(this.mem[addr+1])<<8 | uint32(this.mem[addr+2])<<16 | uint32(this.mem[addr+3])<<24)
}

func (this *vm) setWord(addr int, val int32) {
	if addr < 0 || addr+4 > len(this.mem) {
		this.fail("bad address")
		return
	}

	this.mem[addr], this.mem[addr+1], this.mem[addr+2], this.mem[addr+3] = byte(val), byte(val>>8), byte(val>>16), byte(val>>24)
}

func (this *vm) fail(msg string) {
	if !this.failed {
		this.failed = true
		this.player.log("Script error: " + msg)
	}
}

func (this *vm) str(addr int32) string {
	start := int(addr)
	if start <= 0 || start >= len(this.mem) {
		return ""
	}

	end := start
	for end < len(this.mem) && this.mem[end] != 0 {
		end++
	}
	return string(this.mem[start:end])
}

func (this *vm) setStr(addr int, s string) {
	if addr <= 0 || addr+len(s)+1 > len(this.mem) {
		this.fail("bad string address")
		return
	}

	copy(this.mem[addr:], s)
	this.mem[addr+len(s)] = 0
}

// tempString stores s in one of the rotating temporary buffers and returns its address.
func (this *vm) tempString(s string) int32 {
	if len(s) >= tempSize {
		s = s[:tempSize-1]
	}
	addr := tempBase + this.temp*tempSize
	this.temp = (this.temp + 1) % tempBuffers
	this.setStr(addr, s)
	return int32(addr)
}

func (this *vm) arg(n int) int32 { return this.word(this.sp - 4*n) }

func (this *vm) stackOut() { this.fail("Stack Out!") }

func (this *vm) dst(useB bool, val int32) {
	if useB {
		this.b = val
	} else {
		this.a = val
	}
}

func b2i(b bool) int32 {
	if b {
		return 1
	}

	return 0
}

// exec runs one instruction; yielded reports a Wait call with its duration.
func (this *vm) exec(now uint32) (wait int, yielded bool) {
	ins := uint32(this.word(this.pc))
	if this.failed {
		return 0, false
	}

	this.pc += 4
	op, arg := int(ins>>16), int(int16(ins))
	toB := arg != 0
	switch op {
	case opLdrSystemA, opLdrSystemB:
		this.dst(op == opLdrSystemB, 0)
	case opLdrLocalA, opLdrLocalB:
		this.dst(op == opLdrLocalB, this.word(bodyBase+4*arg))
	case opLdrLocalAddA, opLdrLocalAddB:
		this.dst(op == opLdrLocalAddB, int32(bodyBase+4*arg))
	case opLdrStackBeginA, opLdrStackBeginB:
		this.dst(op == opLdrStackBeginB, this.word(stackBase+4*arg))
	case opLdrStackEndA, opLdrStackEndB:
		this.dst(op == opLdrStackEndB, this.word(this.sp+4*arg))
	case opStrSystemA:
	case opStrStackBeginA:
		this.setWord(stackBase+4*arg, this.a)
	case opStrStackEndA:
		this.setWord(this.sp+4*arg, this.a)
	case opSwapAB:
		this.a, this.b = this.b, this.a
	case opPush:
		this.sp += 4 * arg
		if this.sp > stackBase+4*stackWords {
			this.stackOut()
		}
	case opPop:
		this.sp -= 4 * arg
	case opPushA:
		if this.sp >= stackBase+4*stackWords-4 {
			this.stackOut()
			break
		}

		this.setWord(this.sp, this.a)
		this.sp += 4
	case opPopA:
		this.sp -= 4
		this.a = this.word(this.sp)
	case opGoto:
		this.pc += 4 * arg
	case opCallSystem:
		return this.system(arg, now)
	case opCallLocal:
		if this.sp >= stackBase+4*stackWords {
			this.stackOut()
			break
		}

		this.setWord(this.sp, int32(this.pc))
		this.sp += 4
		this.pc = bodyBase + 4*arg
	case opRetFunction:
		this.sp -= 4
		this.pc = int(this.word(this.sp))
	case opRetEvent:
		if this.phase == 0 && this.onLoop != 0 {
			this.phase, this.pc = 1, bodyBase+4*this.onLoop
		} else {
			this.phase = 2
		}
	case opCreateRef:
		if this.heap+4*arg > tempBase {
			this.fail("out of string memory")
			break
		}

		for i := range arg {
			this.setWord(this.heap+4*i, 0)
		}
		this.setWord(this.sp, int32(this.heap))
		this.sp += 4
		this.heap += 4 * arg
	case opDestroyRef:
		this.heap -= 4 * arg
	case opMinus:
		this.dst(toB, -this.a)
	case opNot:
		this.dst(toB, ^this.a)
	case opLogicalNot:
		this.dst(toB, b2i(this.a == 0))
	case opInc:
		this.dst(toB, this.a+1)
	case opDec:
		this.dst(toB, this.a-1)
	case opAdd:
		this.dst(toB, this.a+this.b)
	case opSub:
		this.dst(toB, this.a-this.b)
	case opMul:
		this.dst(toB, this.a*this.b)
	case opDiv, opMod:
		if this.b == 0 {
			this.fail("division by zero")
			break
		}

		if op == opDiv {
			this.dst(toB, this.a/this.b)
		} else {
			this.dst(toB, this.a%this.b)
		}
	case opShl:
		this.dst(toB, this.a<<(uint32(this.b)&31))
	case opShr:
		this.dst(toB, this.a>>(uint32(this.b)&31))
	case opAnd:
		this.dst(toB, this.a&this.b)
	case opOr:
		this.dst(toB, this.a|this.b)
	case opXor:
		this.dst(toB, this.a^this.b)
	case opEq:
		this.dst(toB, b2i(this.a == this.b))
	case opNe:
		this.dst(toB, b2i(this.a != this.b))
	case opLogicalAnd:
		this.dst(toB, b2i(this.a != 0 && this.b != 0))
	case opLogicalOr:
		this.dst(toB, b2i(this.a != 0 || this.b != 0))
	case opGt:
		this.dst(toB, b2i(this.a > this.b))
	case opGe:
		this.dst(toB, b2i(this.a >= this.b))
	case opLt:
		this.dst(toB, b2i(this.a < this.b))
	case opLe:
		this.dst(toB, b2i(this.a <= this.b))
	case opCopy:
		for i := range arg {
			this.setWord(int(this.a)+4*i, this.word(int(this.b)+4*i))
		}
	case opCopyStr:
		this.setStr(int(this.a), this.str(this.b))
	case opCat:
		s := this.tempString(this.str(this.a) + this.str(this.b))
		this.dst(toB, s)
	case opJz:
		if this.a == 0 {
			this.pc += 4 * arg
		}
	case opJnz:
		if this.a != 0 {
			this.pc += 4 * arg
		}
	}
	return 0, false
}

func (this *vm) system(fn int, now uint32) (int, bool) {
	p := this.player
	switch fn {
	case sysTraceN:
		p.log(this.str(this.arg(1)))
	case sysString:
		this.a = this.tempString(strconv.Itoa(int(this.arg(1))))
	case sysWait:
		return int(this.arg(1)), true
	case sysClean:
		p.canvas.Clear(byte(this.arg(1)))
	case sysPoint:
		p.canvas.Point(int(this.arg(1)), int(this.arg(2)), int(this.arg(3)), byte(this.arg(4)), int(this.arg(5)))
	case sysLine:
		p.canvas.Line(int(this.arg(1)), int(this.arg(2)), int(this.arg(3)), int(this.arg(4)), int(this.arg(5)), int(this.arg(6)),
			byte(this.arg(7)), int(this.arg(8)))
	case sysGraph:
		g, ok := this.assets.graph(this.str(this.arg(1)))
		if !ok {
			break
		}

		pal, ok := this.assets.palette(this.str(this.arg(2)))
		if !ok {
			break
		}

		p.canvas.Graph(g, pal, int(this.arg(3)), int(this.arg(4)), int(this.arg(5)), int(this.arg(6)), int(this.arg(7)), int(this.arg(8)))
	case sysGetIndex:
		this.a = 0
		if idx, ok := this.assets.index(this.str(this.arg(1))); ok {
			this.a = int32(blob.IndexValue(idx, int(this.arg(2)), int(this.arg(3))))
		}
	case sysGetRandomColor:
		this.a = int32(p.randomColor())
	case sysGetTick:
		this.a = int32(now)
	case sysGetAdc, sysGetRealAdc:
		this.a = 0
		if p.ADC != nil {
			this.a = int32(p.ADC(int(this.arg(1)), fn == sysGetRealAdc))
		}
	case sysEnabledKey:
		p.capturedKey = 0
		p.keyCapture = this.arg(1) != 0
	case sysGetKey:
		this.a = int32(p.capturedKey)
		p.capturedKey = 0
	}
	return 0, false
}

// randomColor steps through the vendor's fixed 31-color cycle.
func (this *Player) randomColor() byte {
	this.colorStep++
	if this.colorStep > 30 {
		this.colorStep = 0
	}
	u := this.colorStep + 1
	if u < 29 {
		if u > 23 {
			u = this.colorStep + 3
		}
	} else {
		u = this.colorStep + 4
	}
	r := u % 4
	switch u / 4 {
	case 0:
		return byte(r)
	case 1:
		return byte(r<<2 | 3)
	case 2:
		return byte(r<<4 | 0xF)
	case 3:
		return byte((3-r)*4) | 0x33
	case 4:
		return byte(3-r) | 0x30
	case 5:
		if r != 3 {
			return byte(^(r << 4) & 0x30)
		}
		return 4
	case 6:
		return byte(r << 2)
	case 7:
		return byte(r<<4 | 0xC)
	case 8:
		return byte((r*-0x10 - 0xD | (3-r)*4) & 0x3C)
	}
	return 0
}
