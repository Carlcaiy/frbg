package db

import (
	"encoding/json"
	"frbg/def"
	"frbg/examples/pb"
	"time"

	"github.com/gomodule/redigo/redis"
)

var gameList = []*pb.GameInfo{
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

func GetGameList() []*pb.GameInfo {
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

var roomList = []*pb.RoomInfo{
	{
		GameId:   def.MahjongBanbisan,
		ServerId: 1,
		RoomId:   1,
		Tag:      def.TagNormal,
		Info:     "半壁山麻将",
	},
}

func GetRoomList(gid uint32) []*pb.RoomInfo {
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
	copy := make([]*pb.RoomInfo, 0)
	for _, ptr := range roomList {
		if ptr.GameId == gid {
			copy = append(copy, ptr)
		}
	}
	return copy
}
