package uart

import (
	"strconv"
	"strings"

	"github.com/Tariomka/led-common-lib/pkg/common"
)

// MaxCommandLine is the longest command line the cube accepts.
const MaxCommandLine = 180

// GetCommand wraps a console command line as "#?line?!".
func GetCommand(line string) ([]byte, error) {
	if len(line) == 0 || len(line) > MaxCommandLine || strings.IndexByte(line, '?') >= 0 {
		return nil, common.ErrCommand
	}

	return command(line), nil
}

// GetCommandKey presses KEY1..KEY7 like the physical key (cascaded cubes follow it).
func GetCommandKey(n int) []byte { return command("Key " + strconv.Itoa(n)) }

// GetCommandAction runs a key function code (1..19) on this cube only.
func GetCommandAction(code int) []byte { return command("Action " + strconv.Itoa(code)) }

// GetCommandCartoon plays cartoon index (0-based) of the current show.
func GetCommandCartoon(index int) []byte { return command("Cartoon " + strconv.Itoa(index)) }

// GetCommandStatus asks for the player state; the cube answers with a "Status:" line.
func GetCommandStatus() []byte { return command("Status") }

func GetCommandHelp() []byte { return command("Help") }

func GetCommandRegList() []byte { return command("RegList") }

func GetCommandClrReg() []byte { return command("ClrReg") }

func GetCommandGetReg(key string) ([]byte, error) { return registry("GetReg", key) }

func GetCommandDelReg(key string) ([]byte, error) { return registry("DelReg", key) }

// GetCommandSetReg stores a registry value; key and value can't contain spaces.
func GetCommandSetReg(key, value string) ([]byte, error) {
	if !argument(key) || !argument(value) {
		return nil, common.ErrCommand
	}

	return GetCommand("SetReg " + key + " " + value)
}

func command(line string) []byte {
	return []byte("#?" + line + "?!")
}

func registry(name, key string) ([]byte, error) {
	if !argument(key) {
		return nil, common.ErrCommand
	}

	return GetCommand(name + " " + key)
}

func argument(value string) bool {
	return len(value) > 0 && strings.IndexByte(value, ' ') < 0
}
