package entities

type Todo struct {
	ID          int32  `json:"id" gorm:"primaryKey;autoIncrement:true;unique"`
	Title       string `gorm:"not null"`
	Description string
}
