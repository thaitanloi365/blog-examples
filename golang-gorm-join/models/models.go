package models

import (
	"time"

	"gorm.io/gorm"
)

type Model struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type User struct {
	Model

	Name      string   `json:"name"`
	CompanyID uint     `json:"company_id"`
	Company   *Company `json:"company"`

	CreditCards []*CreditCard `json:"credit_cards"`
}

type Company struct {
	Model
	Name string `json:"name"`
}

type CreditCard struct {
	Model

	UserID uint   `json:"user_id"`
	Number string `json:"card_number"`
	CVV    string `json:"cvv"`
}
