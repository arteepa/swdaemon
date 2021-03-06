package model

import (
	l4g "github.com/alecthomas/log4go"
	"encoding/json"
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/rocketjourney/swdaemon/network"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

type Model struct {
	DB                 gorm.DB
	DateOfLastGet      time.Time
	Net                network.Network
	RJClubId           int
	Delay              int
	Query              string
	TimeFormat         string
	StandByStartHour   int
	StandByStartMinute int
	StandByEndHour     int
	StandByEndMinute   int
}

const (
	VERSION       = "0.12"
	SERVER        = "https://app.rocketjourney.com"
	UPDATE_SERVER = "https://s3.rocketjourney.com"
	UPDATE_PATH   = "/swdaemon/version.json"
)

func (m *Model) SetupModel() error {
	s := m.ReadSettings()
	l4g.Info(s)
	db, err := gorm.Open("mysql", s.User+":"+s.Password+"@tcp("+s.Server+":"+s.Port+")/"+s.DB_name+"?charset=utf8&parseTime=True&loc=Local")
	db.LogMode(true)
	m.DB = db
	m.Net = network.Network{}
	m.Net.Server = SERVER
	m.RJClubId = s.Spot_id
	l4g.Info("Rocket server: %s", m.Net.Server)
	m.Delay = s.Delay
	m.Query = s.Query
	m.TimeFormat = s.Timeformat
	m.DateOfLastGet = time.Now()
	startHour := strings.Split(s.Standbystart, ":")
	endHour := strings.Split(s.Standbyend, ":")
	m.StandByStartHour, _ = strconv.Atoi(startHour[0])
	m.StandByStartMinute, _ = strconv.Atoi(startHour[1])
	m.StandByEndHour, _ = strconv.Atoi(endHour[0])
	m.StandByEndMinute, _ = strconv.Atoi(endHour[1])

	if err != nil {
		l4g.Info(err)
		return err
	} else {
		ping_err := db.DB().Ping()
		if ping_err != nil {
			l4g.Info(ping_err)
			return ping_err
		}
	}
	return nil
}

func (m *Model) SearchAccess() {

	access := []Register{}
	const shortForm = "2006-01-02"
	offsetDateOfLastGet := m.DateOfLastGet
	offsetDateOfLastGet.Add(-5 * time.Second)
	searchDate := offsetDateOfLastGet.Format(shortForm)
	searchHour := offsetDateOfLastGet.Format(m.TimeFormat)
	l4g.Trace("Searching access with offset after: %+v", offsetDateOfLastGet)
	//m.DB.Select("idSentido, idUn, idPersona").Where(m.Query, searchDate, searchHour).Find(&access)
	//limitDate := time.Now()
	offSetLimitDate := time.Now().Add(-5 * time.Second)
	searchlimitHour := offSetLimitDate.Format(m.TimeFormat)
	l4g.Info("Perform search:", m.Query, searchDate, searchHour, searchlimitHour)
	m.DB.Where(m.Query, searchDate, searchHour, searchlimitHour).Find(&access)
	l4g.Info("Number of access found: %+v", len(access))
	m.DateOfLastGet = offSetLimitDate
	for _, r := range access {
		l4g.Trace("%+v", r)
		l4g.Info("Sending check params: way_id: %+v spot_id: %+v user_id: %+v", r.WayId, m.RJClubId, r.UserId)
		m.Net.SendCheck(r.WayId, m.RJClubId, r.UserId)
	}
}

func (m *Model) ReadSettings() *Settings {
	dat, _ := ioutil.ReadFile("./config/config.json")
	settings := Settings{}
	err := json.Unmarshal(dat, &settings)
	if err != nil {
		l4g.Info("error:", err)
	}
	l4g.Info("%+v", settings)
	return &settings
}
