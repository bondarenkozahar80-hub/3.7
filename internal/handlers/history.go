package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"warehouse-system/internal/auth"
	"warehouse-system/internal/database"
	"warehouse-system/internal/models"

	"github.com/gin-gonic/gin"
)

// GetHistory возвращает историю изменений с фильтрацией
func GetHistory(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if !auth.HasPermission(userClaims.Role, "history") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
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

	// Построение динамического запроса
	query := `
		SELECT 
			h.id, 
			h.item_id, 
			h.action, 
			h.changed_by, 
			h.changed_at, 
			h.old_data, 
			h.new_data, 
			h.changes,
			i.name as item_name
		FROM item_history h
		LEFT JOIN items i ON h.item_id = i.id
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if filter.ItemID != nil {
		query += fmt.Sprintf(" AND h.item_id = $%d", argCount)
		args = append(args, *filter.ItemID)
		argCount++
	}

	if filter.ChangedBy != nil {
		query += fmt.Sprintf(" AND h.changed_by = $%d", argCount)
		args = append(args, *filter.ChangedBy)
		argCount++
	}

	if filter.Action != nil {
		query += fmt.Sprintf(" AND h.action = $%d", argCount)
		args = append(args, *filter.Action)
		argCount++
	}

	if filter.FromDate != nil {
		query += fmt.Sprintf(" AND h.changed_at >= $%d", argCount)
		args = append(args, *filter.FromDate)
		argCount++
	}

	if filter.ToDate != nil {
		query += fmt.Sprintf(" AND h.changed_at <= $%d", argCount)
		args = append(args, *filter.ToDate)
		argCount++
	}

	query += fmt.Sprintf(" ORDER BY h.changed_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var history []struct {
		models.ItemHistory
		ItemName string `json:"item_name"`
	}

	for rows.Next() {
		var h struct {
			models.ItemHistory
			ItemName string `json:"item_name"`
		}
		err := rows.Scan(
			&h.ID,
			&h.ItemID,
			&h.Action,
			&h.ChangedBy,
			&h.ChangedAt,
			&h.OldData,
			&h.NewData,
			&h.Changes,
			&h.ItemName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		history = append(history, h)
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   len(history),
		"limit":   filter.Limit,
		"offset":  filter.Offset,
	})
}

// ExportHistory экспортирует историю в CSV
func ExportHistory(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if !auth.HasPermission(userClaims.Role, "history") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Item ID is required"})
		return
	}

	// Преобразуем ID в число
	id, err := strconv.Atoi(itemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var filter models.HistoryFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter.ItemID = &id
	if filter.Limit == 0 {
		filter.Limit = 1000 // Большой лимит для экспорта
	}

	// Получаем историю
	rows, err := database.DB.Query(`
		SELECT 
			h.id,
			h.item_id,
			h.action,
			h.changed_by,
			h.changed_at,
			h.old_data,
			h.new_data,
			h.changes,
			i.name as item_name
		FROM item_history h
		LEFT JOIN items i ON h.item_id = i.id
		WHERE h.item_id = $1
		ORDER BY h.changed_at DESC
		LIMIT $2
	`, id, filter.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Создаем CSV writer
	c.Writer.Header().Set("Content-Type", "text/csv")
	c.Writer.Header().Set("Content-Disposition", 
		fmt.Sprintf("attachment; filename=history_item_%s_%s.csv", 
			itemID, time.Now().Format("20060102_150405")))
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Записываем заголовки
	headers := []string{
		"ID",
		"Item ID",
		"Item Name",
		"Action",
		"Changed By",
		"Changed At",
		"Old Data",
		"New Data",
		"Changes",
		"Changed Fields Count",
	}
	if err := writer.Write(headers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV"})
		return
	}

	// Записываем данные
	for rows.Next() {
		var (
			historyID int
			itemID    int
			action    string
			changedBy string
			changedAt time.Time
			oldData   string
			newData   string
			changes   string
			itemName  string
		)

		err := rows.Scan(
			&historyID,
			&itemID,
			&action,
			&changedBy,
			&changedAt,
			&oldData,
			&newData,
			&changes,
			&itemName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Подсчитываем количество измененных полей
		changedFields := 0
		if changes != "" {
			// Простой подсчет - считаем количество двоеточий в JSON (приблизительно)
			// В реальном приложении нужно парсить JSON
			for i := 0; i < len(changes); i++ {
				if changes[i] == ':' {
					changedFields++
				}
			}
			// Примерно: 2 символа на поле в минимальном JSON
			if changedFields > 0 {
				changedFields /= 2
			}
		}

		record := []string{
			strconv.Itoa(historyID),
			strconv.Itoa(itemID),
			itemName,
			action,
			changedBy,
			changedAt.Format("2006-01-02 15:04:05"),
			oldData,
			newData,
			changes,
			strconv.Itoa(changedFields),
		}

		if err := writer.Write(record); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV row"})
			return
		}
	}

	// Проверяем ошибки итерации
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// GetHistoryStats возвращает статистику по истории
func GetHistoryStats(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if !auth.HasPermission(userClaims.Role, "history") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	// Статистика по действиям
	rows, err := database.DB.Query(`
		SELECT 
			action,
			COUNT(*) as count,
			COUNT(DISTINCT changed_by) as unique_users,
			MIN(changed_at) as first_change,
			MAX(changed_at) as last_change
		FROM item_history
		GROUP BY action
		ORDER BY count DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type ActionStats struct {
		Action       string    `json:"action"`
		Count        int       `json:"count"`
		UniqueUsers  int       `json:"unique_users"`
		FirstChange  time.Time `json:"first_change"`
		LastChange   time.Time `json:"last_change"`
	}

	var stats []ActionStats
	for rows.Next() {
		var s ActionStats
		err := rows.Scan(&s.Action, &s.Count, &s.UniqueUsers, &s.FirstChange, &s.LastChange)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		stats = append(stats, s)
	}

	// Статистика по пользователям
	userRows, err := database.DB.Query(`
		SELECT 
			changed_by,
			COUNT(*) as change_count,
			COUNT(DISTINCT item_id) as items_affected,
			STRING_AGG(DISTINCT action, ', ') as actions_performed
		FROM item_history
		GROUP BY changed_by
		ORDER BY change_count DESC
		LIMIT 10
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer userRows.Close()

	type UserStats struct {
		Username      string `json:"username"`
		ChangeCount   int    `json:"change_count"`
		ItemsAffected int    `json:"items_affected"`
		Actions       string `json:"actions"`
	}

	var userStats []UserStats
	for userRows.Next() {
		var u UserStats
		err := userRows.Scan(&u.Username, &u.ChangeCount, &u.ItemsAffected, &u.Actions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		userStats = append(userStats, u)
	}

	// Общая статистика
	var totalChanges int
	err = database.DB.QueryRow("SELECT COUNT(*) FROM item_history").Scan(&totalChanges)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var firstRecord time.Time
	err = database.DB.QueryRow("SELECT MIN(changed_at) FROM item_history").Scan(&firstRecord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var lastRecord time.Time
	err = database.DB.QueryRow("SELECT MAX(changed_at) FROM item_history").Scan(&lastRecord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_changes": totalChanges,
		"first_record":  firstRecord,
		"last_record":   lastRecord,
		"action_stats":  stats,
		"user_stats":    userStats,
	})
}

// SearchHistory расширенный поиск по истории
func SearchHistory(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if !auth.HasPermission(userClaims.Role, "history") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	type SearchRequest struct {
		Query      string    `form:"q"`
		ItemName   string    `form:"item_name"`
		FromDate   time.Time `form:"from_date"`
		ToDate     time.Time `form:"to_date"`
		Actions    []string  `form:"actions[]"`
		Users      []string  `form:"users[]"`
		Limit      int       `form:"limit" binding:"min=1,max=500"`
		Offset     int       `form:"offset" binding:"min=0"`
	}

	var req SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Limit == 0 {
		req.Limit = 50
	}

	query := `
		SELECT 
			h.id,
			h.item_id,
			h.action,
			h.changed_by,
			h.changed_at,
			h.old_data,
			h.new_data,
			h.changes,
			i.name as item_name
		FROM item_history h
		LEFT JOIN items i ON h.item_id = i.id
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	// Текстовый поиск
	if req.Query != "" {
		query += fmt.Sprintf(` AND (
			i.name ILIKE $%d OR
			h.changed_by ILIKE $%d OR
			h.old_data::text ILIKE $%d OR
			h.new_data::text ILIKE $%d
		)`, argCount, argCount, argCount, argCount)
		args = append(args, "%"+req.Query+"%")
		argCount++
	}

	// Поиск по названию товара
	if req.ItemName != "" {
		query += fmt.Sprintf(" AND i.name ILIKE $%d", argCount)
		args = append(args, "%"+req.ItemName+"%")
		argCount++
	}

	// По дате
	if !req.FromDate.IsZero() {
		query += fmt.Sprintf(" AND h.changed_at >= $%d", argCount)
		args = append(args, req.FromDate)
		argCount++
	}

	if !req.ToDate.IsZero() {
		query += fmt.Sprintf(" AND h.changed_at <= $%d", argCount)
		args = append(args, req.ToDate)
		argCount++
	}

	// По действиям
	if len(req.Actions) > 0 {
		query += fmt.Sprintf(" AND h.action = ANY($%d)", argCount)
		args = append(args, req.Actions)
		argCount++
	}

	// По пользователям
	if len(req.Users) > 0 {
		query += fmt.Sprintf(" AND h.changed_by = ANY($%d)", argCount)
		args = append(args, req.Users)
		argCount++
	}

	query += fmt.Sprintf(" ORDER BY h.changed_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, req.Limit, req.Offset)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var history []struct {
		models.ItemHistory
		ItemName string `json:"item_name"`
	}

	for rows.Next() {
		var h struct {
			models.ItemHistory
			ItemName string `json:"item_name"`
		}
		err := rows.Scan(
			&h.ID,
			&h.ItemID,
			&h.Action,
			&h.ChangedBy,
			&h.ChangedAt,
			&h.OldData,
			&h.NewData,
			&h.Changes,
			&h.ItemName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		history = append(history, h)
	}

	// Получаем общее количество для пагинации
	countQuery := `
		SELECT COUNT(*)
		FROM item_history h
		LEFT JOIN items i ON h.item_id = i.id
		WHERE 1=1
	`
	countArgs := args[:len(args)-2] // Убираем LIMIT и OFFSET
	// Копируем условия без LIMIT/OFFSET
	countQuery += query[len("SELECT ... FROM item_history h LEFT JOIN items i ON h.item_id = i.id WHERE 1=1"):
		len(query)-len(fmt.Sprintf(" ORDER BY h.changed_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1))]

	var total int
	err = database.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		// Если не удалось посчитать, просто возвращаем результаты
		c.JSON(http.StatusOK, gin.H{
			"history": history,
			"total":   len(history),
			"limit":   req.Limit,
			"offset":  req.Offset,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   total,
		"limit":   req.Limit,
		"offset":  req.Offset,
	})
}

// RevertChange откатывает изменение (только для админов)
func RevertChange(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*auth.Claims)
	if userClaims.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can revert changes"})
		return
	}

	historyID := c.Param("history_id")
	if historyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "History ID is required"})
		return
	}

	// Получаем запись истории
	var (
		itemID   int
		oldData  string
		action   string
		itemName string
	)
	err := database.DB.QueryRow(`
		SELECT h.item_id, h.old_data, h.action, i.name
		FROM item_history h
		LEFT JOIN items i ON h.item_id = i.id
		WHERE h.id = $1
	`, historyID).Scan(&itemID, &oldData, &action, &itemName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "History record not found"})
		return
	}

	// Проверяем, можно ли откатить
	if action != "UPDATE" && action != "DELETE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only UPDATE and DELETE actions can be reverted"})
		return
	}

	if oldData == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No old data available for revert"})
		return
	}

	// Для UPDATE: восстанавливаем старые значения
	if action == "UPDATE" {
		// Парсим старые данные и обновляем товар
		// В реальном приложении нужно аккуратно обработать JSON
		_, err := database.DB.Exec(`
			UPDATE items 
			SET 
				name = (old_data->>'name')::text,
				description = (old_data->>'description')::text,
				quantity = (old_data->>'quantity')::integer,
				price = (old_data->>'price')::decimal,
				location = (old_data->>'location')::text,
				updated_at = NOW(),
				created_by = $2
			WHERE id = $1
		`, itemID, userClaims.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revert: " + err.Error()})
			return
		}
	} 
	// Для DELETE: восстанавливаем удаленный товар
	else if action == "DELETE" {
		_, err := database.DB.Exec(`
			INSERT INTO items (name, description, quantity, price, location, created_by)
			SELECT 
				(old_data->>'name')::text,
				(old_data->>'description')::text,
				(old_data->>'quantity')::integer,
				(old_data->>'price')::decimal,
				(old_data->>'location')::text,
				$2
			FROM item_history
			WHERE id = $1
		`, historyID, userClaims.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore: " + err.Error()})
			return
		}
	}

	// Логируем откат
	_, err = database.DB.Exec(`
		INSERT INTO item_history (item_id, action, changed_by, old_data, new_data, changes)
		VALUES ($1, 'REVERT', $2, 
			(SELECT new_data FROM item_history WHERE id = $3),
			(SELECT old_data FROM item_history WHERE id = $3),
			jsonb_build_object('reverted_from', $3)
		)
	`, itemID, userClaims.Username, historyID)
	if err != nil {
		// Не прерываем операцию, если не удалось залогировать
		fmt.Printf("Failed to log revert: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Change reverted successfully",
		"item_id":   itemID,
		"item_name": itemName,
		"action":    action,
	})
}
