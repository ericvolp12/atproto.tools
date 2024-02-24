package stream

import "gorm.io/gorm"

type Record struct {
	gorm.Model
	FirehoseSeq int64  `gorm:"index"`
	Repo        string `gorm:"index:idx_path"`
	Collection  string `gorm:"index:idx_path"`
	RKey        string `gorm:"index:idx_path"`
	Action      string
	Raw         []byte // Raw JSON data
}

type Event struct {
	gorm.Model
	FirehoseSeq int64  `gorm:"primaryKey"`
	Repo        string `gorm:"index"`
	EventType   string `gorm:"index"`
	Error       string
	Time        int64 `gorm:"index"`
}

type Cursor struct {
	gorm.Model
	LastSeq int64
}
