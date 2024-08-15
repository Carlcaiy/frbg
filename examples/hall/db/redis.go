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

// 更新金币
func UpdateMoney(uid uint32, change int64, from string) (int64, error) {
	log.Printf("uid:%d change:%d from:%s", uid, change, from)
	return redis.Int64(redis_cli.Do("HINCRBY", keyUserOnline(uid), "money", change))
}

// 设置玩家桌子
func SetRoom(uid uint32, roomId uint32) error {
	_, err := redis_cli.Do("HSET", keyUserOnline(uid), "roomId", roomId)
	return err
}

// 获取玩家桌子
func GetRoom(uid uint32) (uint32, error) {
	roomId, err := redis.Int64(redis_cli.Do("HGET", keyUserOnline(uid), "roomId"))
	return uint32(roomId), err
}

// 设置玩家桌子
func SetGate(uid uint32, gateId uint32) error {
	_, err := redis_cli.Do("HSET", keyUserOnline(uid), "gateId", gateId)
	return err
}

// 获取玩家桌子
func GetGate(uid uint32) (uint32, error) {
	deskId, err := redis.Int64(redis_cli.Do("HGET", keyUserOnline(uid), "gateId"))
	return uint32(deskId), err
}

// 设置玩家桌子
func SetGame(uid uint32, gameId uint32) error {
	_, err := redis_cli.Do("HSET", keyUserOnline(uid), "gameId", gameId)
	return err
}

// 获取玩家桌子
func GetGame(uid uint32) (uint32, error) {
	gameId, err := redis.Int64(redis_cli.Do("HGET", keyUserOnline(uid), "gameId"))
	return uint32(gameId), err
}
