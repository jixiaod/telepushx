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
	err := db.QueryRow("SELECT id, title, content, image, buttons FROM active WHERE id = ?", id).Scan(
		&content.ID, &content.Title, &content.Content, &content.Image, &content.Buttons)
	if err != nil {
		if err == sql.ErrNoRows {
			return ActiveContent{}, fmt.Errorf("no active content found with id %d", id)
		}
		return ActiveContent{}, fmt.Errorf("error querying active content: %w", err)
	}

	// Fetch buttons for this active content
	rows, err := db.Query("SELECT button_text, button_link FROM buttons WHERE active_id = ?", id)
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
