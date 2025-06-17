package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type Process struct {
	Title   string `json:"title"`
	Command string `json:"command"`
}

func main() {
	// Получаем текущего пользователя
	currentUser, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current user: %v\n", err)
		os.Exit(1)
	}

	// Читаем директорию /proc
	procDir := "/proc"
	entries, err := os.ReadDir(procDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading /proc directory: %v\n", err)
		os.Exit(1)
	}

	var processes []Process

	// Перебираем все записи в /proc
	for _, entry := range entries {
		// Проверяем, является ли имя директории числом (PID)
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // Пропускаем нечисловые директории
		}

		// Читаем файл status для получения UID
		statusPath := filepath.Join(procDir, entry.Name(), "status")
		statusData, err := os.ReadFile(statusPath)
		if err != nil {
			continue
		}

		// Проверяем, принадлежит ли процесс текущему пользователю
		uid := getUIDFromStatus(string(statusData))
		if uid != currentUser.Uid {
			continue
		}

		// Пытаемся получить имя из /proc/<pid>/exe
		exePath := filepath.Join(procDir, entry.Name(), "exe")
		name, err := os.Readlink(exePath)
		if err == nil {
			name = filepath.Base(name) // Берем только имя файла
		} else {
			// Fallback: используем имя из /proc/<pid>/stat
			statPath := filepath.Join(procDir, entry.Name(), "stat")
			data, err := os.ReadFile(statPath)
			if err != nil {
				continue
			}
			parts := strings.Fields(string(data))
			if len(parts) < 2 {
				continue
			}
			name = strings.Trim(parts[1], "()")
		}

		// Формируем структуру Process с PID в title
		process := Process{
			Title:   fmt.Sprintf("%s (%d)", name, pid),
			Command: fmt.Sprintf("kill %d", pid),
		}
		processes = append(processes, process)
	}

	// Преобразуем в JSON
	output, err := json.MarshalIndent(processes, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Выводим результат
	fmt.Println(string(output))
}

// getUIDFromStatus извлекает UID из файла status
func getUIDFromStatus(status string) string {
	lines := strings.Split(status, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				return fields[1] // Реальный UID находится во втором поле
			}
		}
	}
	return ""
}
