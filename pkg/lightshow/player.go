// Package lightshow runs the original iCubeSmart show engine: it walks the cartoons of a show,
// renders their frames and executes key functions. It is a non-blocking port of the factory
// player loop. It knows nothing about the hardware: frames are drawn into a render.Canvas.
//
// Player is not safe for concurrent use: call Key/Action from the same loop that calls Step
// (queue key presses from interrupts and apply them before Step).
package lightshow

import (
	"strconv"

	"github.com/Tariomka/led-common-lib/pkg/common"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/blob"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/render"
)

// Key function codes stored in the show config (KEY1..KEY7).
const (
	ActionNone        = 0
	ActionReplay      = 1
	ActionPrevious    = 2
	ActionNext        = 3
	ActionSpeedUp     = 4
	ActionSpeedReset  = 5
	ActionSpeedDown   = 6
	ActionPlayModel   = 7
	ActionPause       = 8
	ActionDisplayOff  = 9
	ActionJumpCartoon = 10 // 10..19 jump to cartoon 0..9 of the group
)

// Speed factors in tenths: 0.1, 0.2, 0.5, 0.7, 1.0, 1.5, 2.0, 3.0
var speedTenths = [8]int{1, 2, 5, 7, 10, 15, 20, 30}

const defaultSpeed = 4

// pollMs is how long to wait before calling Step again when nothing was drawn.
const pollMs = 100

type Player struct {
	show    Show
	canvas  *render.Canvas
	cartoon Cartoon // the running cartoon

	current int
	running bool // current cartoon has been started
	change  bool // a key selected another (or the same) cartoon
	loop    int
	frame   int
	drawn   bool
	until   uint32 // audio cartoon end time, 0 = forever

	vm *vm

	PlayAll    bool // play model: true = advance through the group, false = repeat one cartoon
	Paused     bool
	DisplayOff bool
	speed      int

	keyCapture  bool
	capturedKey int
	colorStep   int

	// AudioFrame selects the frame of Audio cartoons (0..63) for the cartoon's level mode
	// (header Loop field). The vendor PC simulator always uses 0; on the cube it follows the
	// microphone level.
	AudioFrame func(mode int) int
	// ADC feeds the script functions GetAdc/GetRealAdc.
	ADC func(channel int, raw bool) int
	// Log receives the engine's console messages.
	Log func(msg string)
}

// New creates a player for show drawing into canvas, with default settings.
func New(show Show, canvas *render.Canvas) *Player {
	p := &Player{canvas: canvas, PlayAll: true, speed: defaultSpeed}
	p.SetShow(show)
	return p
}

// SetShow restarts the player on the first cartoon of another show; speed, play model, pause
// and display off are kept.
func (this *Player) SetShow(show Show) {
	if reporter, ok := show.(missingReporter); ok {
		reporter.SetMissing(func(kind, name string) { this.log("Not found " + kind + ":'" + name + "'") })
	}
	this.show, this.cartoon = show, nil
	this.current, this.running, this.change = 0, false, false
	this.keyCapture, this.capturedKey = false, 0
	this.vm = nil
}

func (this *Player) Current() int { return this.current }

// Count is the number of cartoons in the show.
func (this *Player) Count() int { return this.show.Count() }

func (this *Player) Speed() int { return this.speed }

// SetSpeed sets the speed index 0..7 (4 = normal).
func (this *Player) SetSpeed(speed int) {
	if speed >= 0 && speed < len(speedTenths) {
		this.speed = speed
	}
}

// Select restarts the show on cartoon n.
func (this *Player) Select(n int) {
	if n >= 0 && n < this.show.Count() {
		this.current, this.change = n, true
	}
}

func (this *Player) log(msg string) {
	if this.Log != nil {
		this.Log(msg)
	}
}

// Step advances the show. When rendered is true a new frame is in the canvas and should be
// shown for waitMs; otherwise call Step again after waitMs. now is a millisecond clock.
func (this *Player) Step(now uint32) (rendered bool, waitMs int) {
	if this.Paused {
		return false, pollMs
	}

	count := this.show.Count()
	if count == 0 {
		return false, pollMs
	}

	for tries := 0; tries <= 2*count+1; tries++ {
		if this.change {
			this.change, this.running, this.vm = false, false, nil
		}
		if !this.running {
			this.start(now)
		}
		c := this.cartoon
		switch c.Kind() {
		case blob.Normal:
			if rendered, waitMs = this.stepNormal(c); rendered {
				return
			}
		case blob.Audio:
			if rendered, waitMs = this.stepAudio(c, now); rendered {
				return
			}
		case blob.Script:
			var done bool
			if rendered, waitMs, done = this.stepScript(now); !done {
				return
			}
		}
		this.finish()
	}

	return false, pollMs
}

func (this *Player) start(now uint32) {
	cartoon := this.show.Cartoon(this.current)
	this.cartoon = cartoon
	this.running, this.loop, this.frame, this.drawn, this.until = true, 0, 0, false, 0
	if cartoon.Kind() == blob.Audio {
		if secs := cartoon.Header().FrameCount; secs != 0 {
			this.until = now + uint32(secs)*1000
		}
	}
	if script, ok := cartoon.(scriptCartoon); ok && cartoon.Kind() == blob.Script {
		body, a := script.script()
		this.vm = newVM(this, body, a)
	}
}

// finish ends the current cartoon; the next one follows the play model.
func (this *Player) finish() {
	this.running, this.vm = false, nil
	if this.PlayAll {
		this.current = (this.current + 1) % this.show.Count()
	}
}

func (this *Player) frameWait(delay int) int {
	return delay * 10 / speedTenths[this.speed]
}

func (this *Player) stepNormal(cartoon Cartoon) (bool, int) {
	header := cartoon.Header()
	loops := int(uint16(header.Loop))
	if header.FrameCount <= 0 || this.loop >= loops {
		return false, 0
	}

	this.canvas.Clear(0)
	cartoon.Render(this.frame, this.canvas)
	this.frame++
	if this.frame >= header.FrameCount {
		this.frame = 0
		this.loop++
	}
	return true, this.frameWait(header.Delay)
}

func (this *Player) stepAudio(cartoon Cartoon, now uint32) (bool, int) {
	if this.drawn && this.until != 0 && int32(now-this.until) >= 0 {
		return false, 0
	}

	frame := 0
	if this.AudioFrame != nil {
		frame = this.AudioFrame(int(cartoon.Header().Loop))
	}
	this.canvas.Clear(0)
	cartoon.Render(frame, this.canvas)
	this.drawn = true
	return true, this.frameWait(cartoon.Header().Delay)
}

// Key runs the configured function of KEY1..KEY7 (key 0..6).
func (this *Player) Key(key int) {
	this.Action(this.show.KeyFunction(key))
}

func (this *Player) Action(action int) {
	if action < 1 {
		return
	}

	if this.keyCapture {
		this.capturedKey = action
		return
	}

	n := this.show.Count()
	if n == 0 {
		return
	}

	if action >= ActionJumpCartoon && action < ActionJumpCartoon+10 {
		target := action - ActionJumpCartoon
		if target >= n {
			return
		}

		this.log("Change Cartoon " + strconv.Itoa(target))
		this.current, this.change = target, true
		return
	}

	// Messages match the original firmware's console output
	switch action {
	case ActionReplay:
		this.log("Replay Cartoon")
		this.change = true
	case ActionPrevious:
		this.log("Previous Cartoon")
		this.current = (this.current + n - 1) % n
		this.change = true
	case ActionNext:
		this.log("Next Cartoon")
		this.current = (this.current + 1) % n
		this.change = true
	case ActionSpeedUp:
		if this.speed < len(speedTenths)-1 {
			this.speed++
			this.log("Add speed to " + this.SpeedText())
		}
	case ActionSpeedReset:
		if this.speed != defaultSpeed {
			this.speed = defaultSpeed
			this.log("Speed reset to " + this.SpeedText())
		}
	case ActionSpeedDown:
		if this.speed > 0 {
			this.speed--
			this.log("Sub speed to " + this.SpeedText())
		}
	case ActionPlayModel:
		this.PlayAll = !this.PlayAll
		this.log("Play Model=" + common.Flag(this.PlayAll))
	case ActionPause:
		this.Paused = !this.Paused
		this.log("DeviceSuppend:" + common.Flag(this.Paused))
	case ActionDisplayOff:
		this.DisplayOff = !this.DisplayOff
		this.log("Display off=" + common.Flag(this.DisplayOff))
	}
}

// SpeedText formats the speed factor like "%.2f" (e.g. "1.50").
func (this *Player) SpeedText() string {
	tenths := speedTenths[this.speed]
	return strconv.Itoa(tenths/10) + "." + strconv.Itoa(tenths%10) + "0"
}

// stepScript runs the script until it waits, ends or uses up its instruction budget.
func (this *Player) stepScript(now uint32) (rendered bool, waitMs int, done bool) {
	v := this.vm
	if v == nil {
		return false, 0, true
	}
	for i := 0; i < maxSteps; i++ {
		if v.phase == 2 || v.failed {
			return false, 0, true
		}
		if wait, yielded := v.exec(now); yielded {
			return true, this.frameWait(wait), false
		}
	}
	return false, 0, false
}
