package model

import (
	"one-api/common"
	"strconv"

	"gorm.io/gorm"
)

type Channel struct {
	Id                 int     `json:"id"`
	Type               int     `json:"type" gorm:"default:0"`
	Key                string  `json:"key" gorm:"not null;index"`
	Status             int     `json:"status" gorm:"default:1"`
	Name               string  `json:"name" gorm:"index"`
	Weight             *uint   `json:"weight" gorm:"default:0"`
	CreatedTime        int64   `json:"created_time" gorm:"bigint"`
	TestTime           int64   `json:"test_time" gorm:"bigint"`
	ResponseTime       int     `json:"response_time"` // in milliseconds
	BaseURL            *string `json:"base_url" gorm:"column:base_url;default:''"`
	Other              string  `json:"other"`
	Balance            float64 `json:"balance"` // in USD
	BalanceUpdatedTime int64   `json:"balance_updated_time" gorm:"bigint"`
	Models             string  `json:"models"`
	Group              string  `json:"group" gorm:"type:varchar(32);default:'default'"`
	UsedQuota          int64   `json:"used_quota" gorm:"bigint;default:0"`
	ModelMapping       *string `json:"model_mapping" gorm:"type:varchar(1024);default:''"`
	Priority           *int64  `json:"priority" gorm:"bigint;default:0"`
	AllowStreaming     int     `json:"allow_streaming" gorm:"default:1"`
	AllowNonStreaming  int     `json:"allow_non_streaming" gorm:"default:1"`
}

func GetAllChannels(startIdx int, num int, selectAll bool) ([]*Channel, error) {
	var channels []*Channel
	var err error
	if selectAll {
		err = DB.Order("id desc").Find(&channels).Error
	} else {
		err = DB.Order("id desc").Limit(num).Offset(startIdx).Omit("key").Find(&channels).Error
	}
	return channels, err
}

func SearchChannels(keyword string) (channels []*Channel, err error) {
	idKeyword, err := strconv.Atoi(keyword)

	err = DB.Omit("key").Where("name LIKE ?", keyword+"%").Or(&Channel{Id: idKeyword}).Or(&Channel{Key: keyword}).Find(&channels).Error
	return channels, err
}

func GetChannelById(id int, selectAll bool) (*Channel, error) {
	channel := Channel{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(&channel, "id = ?", id).Error
	} else {
		err = DB.Omit("key").First(&channel, "id = ?", id).Error
	}
	return &channel, err
}

func GetRandomChannel() (*Channel, error) {
	channel := Channel{}
	var err error = nil
	if common.UsingPostgreSQL {
		err = DB.Where(&Channel{Status: common.ChannelStatusEnabled, Group: "default"}).Order("RANDOM()").Limit(1).First(&channel).Error
	} else if common.UsingSQLite {
		err = DB.Where(&Channel{Status: common.ChannelStatusEnabled, Group: "default"}).Order("RANDOM()").Limit(1).First(&channel).Error
	} else {
		err = DB.Where(&Channel{Status: common.ChannelStatusEnabled, Group: "default"}).Order("RAND()").Limit(1).First(&channel).Error
	}
	return &channel, err
}

func BatchInsertChannels(channels []Channel) error {
	var err error
	err = DB.Create(&channels).Error
	if err != nil {
		return err
	}
	for _, channel_ := range channels {
		err = channel_.AddAbilities()
		if err != nil {
			return err
		}
	}
	return nil
}

func (channel *Channel) GetPriority() int64 {
	if channel.Priority == nil {
		return 0
	}
	return *channel.Priority
}

func (channel *Channel) GetBaseURL() string {
	if channel.BaseURL == nil {
		return ""
	}
	return *channel.BaseURL
}

func (channel *Channel) GetModelMapping() string {
	if channel.ModelMapping == nil {
		return ""
	}
	return *channel.ModelMapping
}

func (channel *Channel) Insert() error {
	var err error
	err = DB.Create(channel).Error
	if err != nil {
		return err
	}
	err = channel.AddAbilities()
	return err
}

func (channel *Channel) Update() error {
	var err error
	err = DB.Model(channel).Updates(channel).Error
	if err != nil {
		return err
	}
	DB.Model(channel).First(channel, "id = ?", channel.Id)
	err = channel.UpdateAbilities()
	return err
}

func (channel *Channel) UpdateResponseTime(responseTime int64) {
	err := DB.Model(channel).Select("response_time", "test_time").Updates(Channel{
		TestTime:     common.GetTimestamp(),
		ResponseTime: int(responseTime),
	}).Error
	if err != nil {
		common.SysError("failed to update response time: " + err.Error())
	}
}

func (channel *Channel) UpdateBalance(balance float64) {
	err := DB.Model(channel).Select("balance_updated_time", "balance").Updates(Channel{
		BalanceUpdatedTime: common.GetTimestamp(),
		Balance:            balance,
	}).Error
	if err != nil {
		common.SysError("failed to update balance: " + err.Error())
	}
}

func (channel *Channel) Delete() error {
	var err error
	err = DB.Delete(channel).Error
	if err != nil {
		return err
	}
	err = channel.DeleteAbilities()
	return err
}

func UpdateChannelStatusById(id int, status int) {
	err := UpdateAbilityStatus(id, status == common.ChannelStatusEnabled)
	if err != nil {
		common.SysError("failed to update ability status: " + err.Error())
	}
	err = DB.Model(&Channel{}).Where("id = ?", id).Update("status", status).Error
	if err != nil {
		common.SysError("failed to update channel status: " + err.Error())
	}
}

func UpdateChannelUsedQuota(id int, quota int) {
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeChannelUsedQuota, id, quota)
		return
	}
	updateChannelUsedQuota(id, quota)
}

func updateChannelUsedQuota(id int, quota int) {
	err := DB.Model(&Channel{}).Where("id = ?", id).Update("used_quota", gorm.Expr("used_quota + ?", quota)).Error
	if err != nil {
		common.SysError("failed to update channel used quota: " + err.Error())
	}
}

func DeleteChannelByStatus(status int64) (int64, error) {
	result := DB.Where("status = ?", status).Delete(&Channel{})
	return result.RowsAffected, result.Error
}
