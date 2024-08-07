package service

type Session struct {
	ID string
}

type Request struct {
	Session     Session
	Attrs       map[Param]int
	WaitRoomCfg WaitRoomConfig
}

type Param string

const (
	ParamTable  Param = "table"
	ParamLeague Param = "league"
	ParamLevel  Param = "level"
	ParamGame   Param = "game"
)

func (p Param) String() string {
	return string(p)
}

func (p Param) IsValid() bool {
	switch p {
	case ParamTable, ParamLeague, ParamLevel, ParamGame:
		return true
	}
	return false
}
