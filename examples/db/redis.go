package db

import (
	"fmt"
	"frbg/local"
	"log"

	"github.com/gomodule/redigo/redis"
)

var redis_cli = local.NewRedis(":6379")

func keyUserOnline(uid uint32) string {
	return fmt.Sprintf("online:%d", uid)
}

func keyRoomMatch(roomId uint32, gameId uint32) string {
	return fmt.Sprintf("match:%d:%d", roomId, gameId)
}

// 更新金币
func UpdateMoney(uid uint32, change int64, from string) (int64, error) {
	log.Printf("uid:%d change:%d from:%s", uid, change, from)
	return redis.Int64(redis_cli.Do("HINCRBY", keyUserOnline(uid), "money", change))
}

// 添加匹配队列
func AddQueen(uid uint32, weight int64, roomId uint32, gameId uint32) error {
	_, err := redis_cli.Do("ZADD", keyRoomMatch(roomId, gameId), weight, uid)
	return err
}

// 获取队列成员数量
func GetQueenCnt(roomId uint32, gameId uint32) (int, error) {
	return redis.Int(redis_cli.Do("ZCARD", keyRoomMatch(roomId, gameId)))
}

// 获取队列成员
func GetQueenUsers(roomId uint32, gameId uint32, count uint32) ([]uint32, error) {
	if slice, err := redis.Ints(redis_cli.Do("ZPOPMIN", keyRoomMatch(roomId, gameId), count)); err != nil {
		return nil, err
	} else {
		ret := make([]uint32, count)
		for i := range slice {
			ret[i] = uint32(slice[i])
		}
		return ret, nil
	}
}

// 设置玩家桌子
func SetDesk(uid uint32, deskId uint32) error {
	_, err := redis_cli.Do("HSET", keyUserOnline(uid), "deskId", deskId)
	return err
}

// 获取玩家桌子
func GetDesk(uid uint32) (uint32, error) {
	deskId, err := redis.Int64(redis_cli.Do("HGET", keyUserOnline(uid), "deskId"))
	return uint32(deskId), err
}

// 设置玩家桌子
func SetGate(uid uint32, gateId uint8) error {
	_, err := redis_cli.Do("HSET", keyUserOnline(uid), "gateId", gateId)
	return err
}

// 获取玩家桌子
func GetGate(uid uint32) uint8 {
	deskId, _ := redis.Int64(redis_cli.Do("HGET", keyUserOnline(uid), "gateId"))
	return uint8(deskId)
}

// 设置玩家桌子
func SetGame(uid uint32, gameId uint8, deskId uint32) error {
	_, err := redis_cli.Do("HSET", keyUserOnline(uid), "gameId", gameId, "deskId", deskId)
	return err
}

// 获取玩家桌子
func GetGame(uid uint32) uint8 {
	gameId, _ := redis.Int64(redis_cli.Do("HGET", keyUserOnline(uid), "gameId"))
	return uint8(gameId)
}

func keyUser(uid uint32) string {
	return fmt.Sprintf("user:%d", uid)
}

func GetUser(uid uint32, data interface{}) error {
	all, err := redis.Values(redis_cli.Do("HGETALL", keyUser(uid)))
	if err != nil {
		return err
	}
	return redis.ScanStruct(all, data)
}

func SetUser(uid uint32, data interface{}) error {
	_, err := redis_cli.Do("HSET", redis.Args{}.Add(keyUser(uid)).AddFlat(data)...)
	return err
}

func GenUserId() (uint32, error) {
	uid, err := redis.Int64(redis_cli.Do("INCRBY", "uid", 1))
	if uid < 100000 {
		uid, err = redis.Int64(redis_cli.Do("INCRBY", "uid", 100000))
	}
	return uint32(uid), err
}
