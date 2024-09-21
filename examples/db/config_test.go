package db

import (
	"encoding/json"
	"fmt"
	"frbg/examples/proto"
	"testing"
)

func TestGetGameList(t *testing.T) {
	var gameList = []*proto.GameInfo{
		{GameId: 1},
		{GameId: 2},
		{GameId: 3},
		{GameId: 4},
	}
	bs, _ := json.Marshal(gameList)
	redis_cli.Do("SET", "game_list", string(bs))

	get := GetGameList()
	if len(get) != len(gameList) {
		t.Errorf("eeeeee")
	}
	for i := range gameList {
		if gameList[i].GameId != get[i].GameId {
			t.Errorf("eeeeee")
		}
	}
}

func TestGetRoomList(t *testing.T) {
	gameId := int32(1)
	var roomList = []*proto.RoomInfo{
		{RoomId: 1},
		{RoomId: 2},
		{RoomId: 3},
		{RoomId: 4},
	}
	bs, _ := json.Marshal(roomList)
	redis_cli.Do("SET", fmt.Sprintf("room_list:%d", gameId), string(bs))

	get := GetRoomList(gameId)
	if len(get) != len(roomList) {
		t.Errorf("eeeeee")
	}
	for i := range roomList {
		if roomList[i].RoomId != get[i].RoomId {
			t.Errorf("eeeeee")
		}
	}
}
