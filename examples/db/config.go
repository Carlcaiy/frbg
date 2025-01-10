package db

import (
	"encoding/json"
	"frbg/examples/proto"
	"time"

	"github.com/gomodule/redigo/redis"
)

var gameList []*proto.GameInfo = []*proto.GameInfo{{
	GameId:    1,
	Status:    1,
	StartTime: uint32(time.Now().Unix()),
	EndTime:   0,
}}

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

var roomList []*proto.RoomInfo = []*proto.RoomInfo{
	{
		ServerId: 1,
		RoomId:   1,
		Tag:      1,
	},
}

func GetRoomList(gid int32) []*proto.RoomInfo {
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
	return roomList
}
