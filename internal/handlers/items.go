package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"warehouse-system/internal/auth"
	"warehouse-system/internal/database"
	"warehouse-system/internal/models"

	"github.com/gin-gonic/gin"
)

func CreateItem(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if !auth.HasPermission(userClaims.Role, "create") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	var req models.CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var item models.Item
	err := database.DB.QueryRow(`
		INSERT INTO items (name, description, quantity, price, location, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, description, quantity, price, location, created_at, updated_at, created_by
	`, req.Name, req.Description, req.Quantity, req.Price, req.Location, userClaims.Username).
	Scan(&item.ID, &item.Name, &item.Description, &item.Quantity, &item.Price,
		&item.Location, &item.CreatedAt, &item.UpdatedAt, &item.CreatedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

func GetItems(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if !auth.HasPermission(userClaims.Role, "read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, name, description, quantity, price, location, created_at, updated_at, created_by
		FROM items
		ORDER BY id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Quantity,
			&item.Price, &item.Location, &item.CreatedAt, &item.UpdatedAt, &item.CreatedBy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

func UpdateItem(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if !auth.HasPermission(userClaims.Role, "update") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req models.UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем существование товара
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM items WHERE id = $1)", id).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// Собираем динамический запрос
	query := "UPDATE items SET "
	args := []interface{}{}
	argCount := 1

	if req.Name != nil {
		query += "name = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *req.Name)
		argCount++
	}
	if req.Description != nil {
		query += "description = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *req.Description)
		argCount++
	}
	if req.Quantity != nil {
		query += "quantity = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *req.Quantity)
		argCount++
	}
	if req.Price != nil {
		query += "price = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *req.Price)
		argCount++
	}
	if req.Location != nil {
		query += "location = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *req.Location)
		argCount++
	}

	query = query[:len(query)-2] // Убираем последнюю запятую и пробел
	query += " WHERE id = $" + strconv.Itoa(argCount)
	args = append(args, id)

	_, err = database.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем обновленный товар
	var item models.Item
	err = database.DB.QueryRow(`
		SELECT id, name, description, quantity, price, location, created_at, updated_at, created_by
		FROM items WHERE id = $1
	`, id).Scan(&item.ID, &item.Name, &item.Description, &item.Quantity,
		&item.Price, &item.Location, &item.CreatedAt, &item.UpdatedAt, &item.CreatedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func DeleteItem(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if !auth.HasPermission(userClaims.Role, "delete") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	// Получаем товар перед удалением для истории
	var item models.Item
	err = database.DB.QueryRow(`
		SELECT id, name, description, quantity, price, location, created_at, updated_at, created_by
		FROM items WHERE id = $1
	`, id).Scan(&item.ID, &item.Name, &item.Description, &item.Quantity,
		&item.Price, &item.Location, &item.CreatedAt, &item.UpdatedAt, &item.CreatedBy)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Удаляем товар (триггер запишет в историю)
	_, err = database.DB.Exec("DELETE FROM items WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item deleted successfully"})
}

func GetItemHistory(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if !auth.HasPermission(userClaims.Role, "history") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var filter models.HistoryFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if filter.Limit == 0 {
		filter.Limit = 50
	}

	query := `
		SELECT id, item_id, action, changed_by, changed_at, old_data, new_data, changes
		FROM item_history
		WHERE item_id = $1
	`
	args := []interface{}{id}
	argCount := 2

	if filter.ChangedBy != nil {
		query += " AND changed_by = $" + strconv.Itoa(argCount)
		args = append(args, *filter.ChangedBy)
		argCount++
	}
	if filter.Action != nil {
		query += " AND action = $" + strconv.Itoa(argCount)
		args = append(args, *filter.Action)
		argCount++
	}
	if filter.FromDate != nil {
		query += " AND changed_at >= $" + strconv.Itoa(argCount)
		args = append(args, *filter.FromDate)
		argCount++
	}
	if filter.ToDate != nil {
		query += " AND changed_at <= $" + strconv.Itoa(argCount)
		args = append(args, *filter.ToDate)
		argCount++
	}

	query += " ORDER BY changed_at DESC LIMIT $" + strconv.Itoa(argCount) + " OFFSET $" + strconv.Itoa(argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var history []models.ItemHistory
	for rows.Next() {
		var h models.ItemHistory
		err := rows.Scan(&h.ID, &h.ItemID, &h.Action, &h.ChangedBy,
			&h.ChangedAt, &h.OldData, &h.NewData, &h.Changes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		history = append(history, h)
	}

	c.JSON(http.StatusOK, history)
}

func GetHistoryDiff(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if !auth.HasPermission(userClaims.Role, "history") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	historyID, err := strconv.Atoi(c.Param("history_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid history ID"})
		return
	}

	var history models.ItemHistory
	err = database.DB.QueryRow(`
		SELECT old_data, new_data, changes
		FROM item_history
		WHERE id = $1
	`, historyID).Scan(&history.OldData, &history.NewData, &history.Changes)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "History record not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var changes map[string]interface{}
	if err := json.Unmarshal([]byte(history.Changes), &changes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse changes"})
		return
	}

	// Преобразуем изменения в удобный формат
	var diffs []models.DiffResponse
	for field, data := range changes {
		if dataMap, ok := data.(map[string]interface{}); ok {
			diffs = append(diffs, models.DiffResponse{
				Field: field,
				Old:   dataMap["old"],
				New:   dataMap["new"],
			})
		}
	}

	c.JSON(http.StatusOK, diffs)
}
