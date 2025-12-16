package db

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type MatchingUser struct {
	MatchTime int64  `json:"matchTime"`
	PlayRound int32  `json:"playRound"`
	Uid       uint32 `json:"uid"`
}

func LoadMatchingUser(roomId int32) (*MatchingUser, error) {
	yamlFile, err := os.Open(fmt.Sprintf("matching_user_%d.yaml", roomId))
	if err != nil {
		return nil, err
	}
	defer yamlFile.Close()
	ret := new(MatchingUser)
	err = yaml.NewDecoder(yamlFile).Decode(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func StoreMatchingUser(roomId int32, mu []*MatchingUser) error {
	yamlFile, err := os.Create(fmt.Sprintf("matching_user_%d.yaml", roomId))
	if err != nil {
		return err
	}
	defer yamlFile.Close()
	err = yaml.NewEncoder(yamlFile).Encode(mu)
	if err != nil {
		return err
	}
	return nil
}
