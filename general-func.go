package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"time"
)

func NewLogger(filename string) *Logger {
	return &Logger{filename: filename}
}

func (l *Logger) Log(text string) error {
	file, err := os.OpenFile(l.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("не удалось открыть лог-файл: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - " + text + "\n")
	if err != nil {
		return fmt.Errorf("не удалось записать в лог-файл: %w", err)
	}

	return nil
}

func ParseCSVToMaps(filePath string) (*CSVData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Ошибка открытия файла: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Читаем заголовки
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения заголовков: %w", err)
	}

	// Читаем данные и преобразуем в map
	var rows []map[string]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Ошибка чтения записи: %w", err)
		}

		row := make(map[string]string)
		for i, value := range record {
			if i < len(header) {
				row[header[i]] = value
			}
		}
		rows = append(rows, row)
	}

	return &CSVData{
		Header: header,
		Rows:   rows,
	}, nil
}
