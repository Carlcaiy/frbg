package db

import (
	"encoding/json"
	"frbg/examples/proto"

	"github.com/gomodule/redigo/redis"
)

var gameList []*proto.GameInfo

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

var roomList []*proto.RoomInfo

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
