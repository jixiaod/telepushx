package model

import (
	"database/sql"
	"fmt"
)

type ActiveContent struct {
	ID      int
	Title   string
	Content string
	Image   string
	Buttons []Button
}

type Button struct {
	Text string
	Link string
}

func GetActiveContentByID(db *sql.DB, id int) (ActiveContent, error) {
	var content ActiveContent
	err := db.QueryRow("SELECT id, activity_text, activity_image FROM activity WHERE id = ?", id).Scan(
		&content.ID, &content.Content, &content.Image)
	if err != nil {
		if err == sql.ErrNoRows {
			return ActiveContent{}, fmt.Errorf("no active content found with id %d", id)
		}
		return ActiveContent{}, fmt.Errorf("error querying active content: %w", err)
	}

	// Fetch buttons for this active content
	rows, err := db.Query("SELECT button_text, button_link FROM activity_button WHERE activity_id = ?", id)
	if err != nil {
		return ActiveContent{}, fmt.Errorf("error querying buttons: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var button Button
		if err := rows.Scan(&button.Text, &button.Link); err != nil {
			return ActiveContent{}, fmt.Errorf("error scanning button row: %w", err)
		}
		content.Buttons = append(content.Buttons, button)
	}

	if err := rows.Err(); err != nil {
		return ActiveContent{}, fmt.Errorf("error iterating button rows: %w", err)
	}

	return content, nil
}
