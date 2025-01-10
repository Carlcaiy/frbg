package db

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var gormcli *gorm.DB

func initSQL() {
	dsn := fmt.Sprintf("%s:%s@(%s)/%s?charset=%s&parseTime=True&loc=Local", "domino", "yAUpZwWnjfrPBsWD", "192.168.1.129:3306", "xx_joyfun", "utf8")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(db)
	}
	gormcli = db
}

type RoomConf struct {
	Id        int32  `json:"id"`
	Name      string `json:"name"`
	Level     int32  `json:"level"`
	Type      int32  `json:"type"`
	Needblind int64  `json:"needblind"`
	Blind     int64  `json:"blind"`
	Minmoney  int64  `json:"minmoney"`
	Maxmoney  int64  `json:"maxmoney"`
	Seat      int32  `json:"seat"`
	Audience  int32  `json:"audience"`
	Props     int64  `json:"props"`
	Fee       int64  `json:"fee"`
	Fast      bool   `json:"fast"`
	Status    int8   `json:"status"`
	Allin     bool   `json:"allin"`
	Pcheat    bool   `json:"cheat"`
	LeadChips int64  `json:"lead_chips"`
	Ext       string `json:"ext"`
	Quick     string `json:"quick"`
	ShowNo    string `json:"showno"`
	Ver       int32
}

func GetRoomConf() []*RoomConf {
	roomsConf := make([]*RoomConf, 0)
	gormcli.Table("room").Where("level=?", 101).Order("blind ASC").Find(&roomsConf)
	if len(roomsConf) == 0 {
		return nil
	}
	return roomsConf
}
