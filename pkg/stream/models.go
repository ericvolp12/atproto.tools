package stream

import (
	"time"

	"gorm.io/gorm"
)

type Record struct {
	ID        uint `gorm:"primarykey;index:idx_records_repo_id,priority:2,order:desc"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	FirehoseSeq int64  `gorm:"index"`
	Repo        string `gorm:"index:idx_path;index:idx_records_repo_id,priority:1"`
	Collection  string `gorm:"index:idx_path"`
	RKey        string `gorm:"index:idx_path"`
	Action      string
	Raw         []byte // Raw JSON data
}

type Event struct {
	ID        uint `gorm:"primarykey;index:idx_events_repo_id,priority:2,order:desc"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	FirehoseSeq int64  `gorm:"primaryKey"`
	Repo        string `gorm:"index;index:idx_events_repo_id,priority:1"`
	EventType   string `gorm:"index"`
	Error       string
	Time        int64 `gorm:"index"`
}

type Cursor struct {
	gorm.Model
	LastSeq int64
}
