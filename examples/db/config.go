package db

import (
	"encoding/json"
	"frbg/def"
	"frbg/examples/proto"
	"time"

	"github.com/gomodule/redigo/redis"
)

var gameList = []*proto.GameInfo{
	{
		GameId:    def.SlotsFu,
		Status:    1,
		StartTime: uint32(time.Now().Unix()),
		EndTime:   0,
	}, {
		GameId:    def.MahjongBanbisan,
		Status:    1,
		StartTime: uint32(time.Now().Unix()),
		EndTime:   0,
	},
}

func GetGameList() []*proto.GameInfo {
	if gameList == nil {
		bs, err := redis.Bytes(redis_cli.Do("GET", "game_list"))
		if err != nil {
			return nil
		}
		err = json.Unmarshal(bs, &gameList)
		if err != nil {
			return nil
		}
	}
	return gameList
}

var roomList = []*proto.RoomInfo{
	{
		GameId:   def.MahjongBanbisan,
		ServerId: 1,
		RoomId:   1,
		Tag:      def.TagNormal,
		Info:     "半壁山麻将",
	},
}

func GetRoomList(gid uint32) []*proto.RoomInfo {
	if roomList == nil {
		bs, err := redis.Bytes(redis_cli.Do("GET", "room_list"))
		if err != nil {
			return nil
		}
		err = json.Unmarshal(bs, &roomList)
		if err != nil {
			return nil
		}
	}
	copy := make([]*proto.RoomInfo, 0)
	for _, ptr := range roomList {
		if ptr.GameId == gid {
			copy = append(copy, ptr)
		}
	}
	return copy
}
