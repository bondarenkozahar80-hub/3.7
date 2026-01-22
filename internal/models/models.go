package models

import (
	"time"
)

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleViewer  Role = "viewer"
	RoleAuditor Role = "auditor"
)

type Item struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Quantity    int       `json:"quantity" db:"quantity"`
	Price       float64   `json:"price" db:"price"`
	Location    string    `json:"location" db:"location"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy   string    `json:"created_by" db:"created_by"`
}

type ItemHistory struct {
	ID         int       `json:"id" db:"id"`
	ItemID     int       `json:"item_id" db:"item_id"`
	Action     string    `json:"action" db:"action"` // CREATE, UPDATE, DELETE
	ChangedBy  string    `json:"changed_by" db:"changed_by"`
	ChangedAt  time.Time `json:"changed_at" db:"changed_at"`
	OldData    string    `json:"old_data" db:"old_data"`     // JSON предыдущего состояния
	NewData    string    `json:"new_data" db:"new_data"`     // JSON нового состояния
	Changes    string    `json:"changes" db:"changes"`       // JSON измененных полей
}

type User struct {
	Username string `json:"username" db:"username"`
	Role     Role   `json:"role" db:"role"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Role     Role   `json:"role" binding:"required"`
}

type CreateItemRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity" binding:"required,min=0"`
	Price       float64 `json:"price" binding:"required,min=0"`
	Location    string  `json:"location"`
}

type UpdateItemRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Quantity    *int     `json:"quantity"`
	Price       *float64 `json:"price"`
	Location    *string  `json:"location"`
}

type HistoryFilter struct {
	ItemID   *int       `form:"item_id"`
	ChangedBy *string   `form:"changed_by"`
	Action   *string    `form:"action"`
	FromDate *time.Time `form:"from_date"`
	ToDate   *time.Time `form:"to_date"`
	Limit    int        `form:"limit" binding:"min=1,max=100"`
	Offset   int        `form:"offset" binding:"min=0"`
}

type DiffResponse struct {
	Field   string      `json:"field"`
	Old     interface{} `json:"old"`
	New     interface{} `json:"new"`
}
