// Code generated by "enumer -type ProfileMode lib/profile.go"; DO NOT EDIT.

package lib

import (
	"fmt"
	"strings"
)

const _ProfileModeName = "cpumemHeapmemAllocsmutexblocktracethreadCreategoroutine"

var _ProfileModeIndex = [...]uint8{0, 3, 10, 19, 24, 29, 34, 46, 55}

const _ProfileModeLowerName = "cpumemheapmemallocsmutexblocktracethreadcreategoroutine"

func (i ProfileMode) String() string {
	if i < 0 || i >= ProfileMode(len(_ProfileModeIndex)-1) {
		return fmt.Sprintf("ProfileMode(%d)", i)
	}
	return _ProfileModeName[_ProfileModeIndex[i]:_ProfileModeIndex[i+1]]
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _ProfileModeNoOp() {
	var x [1]struct{}
	_ = x[cpu-(0)]
	_ = x[memHeap-(1)]
	_ = x[memAllocs-(2)]
	_ = x[mutex-(3)]
	_ = x[block-(4)]
	_ = x[trace-(5)]
	_ = x[threadCreate-(6)]
	_ = x[goroutine-(7)]
}

var _ProfileModeValues = []ProfileMode{cpu, memHeap, memAllocs, mutex, block, trace, threadCreate, goroutine}

var _ProfileModeNameToValueMap = map[string]ProfileMode{
	_ProfileModeName[0:3]:        cpu,
	_ProfileModeLowerName[0:3]:   cpu,
	_ProfileModeName[3:10]:       memHeap,
	_ProfileModeLowerName[3:10]:  memHeap,
	_ProfileModeName[10:19]:      memAllocs,
	_ProfileModeLowerName[10:19]: memAllocs,
	_ProfileModeName[19:24]:      mutex,
	_ProfileModeLowerName[19:24]: mutex,
	_ProfileModeName[24:29]:      block,
	_ProfileModeLowerName[24:29]: block,
	_ProfileModeName[29:34]:      trace,
	_ProfileModeLowerName[29:34]: trace,
	_ProfileModeName[34:46]:      threadCreate,
	_ProfileModeLowerName[34:46]: threadCreate,
	_ProfileModeName[46:55]:      goroutine,
	_ProfileModeLowerName[46:55]: goroutine,
}

var _ProfileModeNames = []string{
	_ProfileModeName[0:3],
	_ProfileModeName[3:10],
	_ProfileModeName[10:19],
	_ProfileModeName[19:24],
	_ProfileModeName[24:29],
	_ProfileModeName[29:34],
	_ProfileModeName[34:46],
	_ProfileModeName[46:55],
}

// ProfileModeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func ProfileModeString(s string) (ProfileMode, error) {
	if val, ok := _ProfileModeNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _ProfileModeNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to ProfileMode values", s)
}

// ProfileModeValues returns all values of the enum
func ProfileModeValues() []ProfileMode {
	return _ProfileModeValues
}

// ProfileModeStrings returns a slice of all String values of the enum
func ProfileModeStrings() []string {
	strs := make([]string, len(_ProfileModeNames))
	copy(strs, _ProfileModeNames)
	return strs
}

// IsAProfileMode returns "true" if the value is listed in the enum definition. "false" otherwise
func (i ProfileMode) IsAProfileMode() bool {
	for _, v := range _ProfileModeValues {
		if i == v {
			return true
		}
	}
	return false
}
