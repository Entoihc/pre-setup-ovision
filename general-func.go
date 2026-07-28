package main

import (
	"fmt"
	"os"
	"time"
)

func log(filename, text string) error {

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Не удалось открыть лог-файл: %w", err)
	}
	defer file.Close()

	//	fmt.Println(text)
	_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - " + text + "\n")
	if err != nil {
		return fmt.Errorf("Не удалось записать в строку в лог-файл: %w", err)
	}

	return nil
}
