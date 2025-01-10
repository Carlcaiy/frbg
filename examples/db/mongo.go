package db

import (
	"context"
	"log"

	"github.com/qiniu/qmgo"
)

var mongcli *qmgo.QmgoClient

func initMG() {
	ctx := context.Background()
	c, err := qmgo.Open(ctx, &qmgo.Config{Uri: "mongodb://172.20.11.80:28000", Database: "frbgdb", Coll: "users"})
	if err != nil {
		panic(err)
	}
	mongcli = c
}

func SlotsLog(l interface{}) {
	_, err := mongcli.InsertOne(context.Background(), l)
	if err != nil {
		log.Println(err)
	}
}
