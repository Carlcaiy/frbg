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
func SetGame(uid uint32, gameId uint8, roomId uint32) error {
	_, err := redis_cli.Do("HSET", keyUserOnline(uid), "gameId", gameId, "roomId", roomId)
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
