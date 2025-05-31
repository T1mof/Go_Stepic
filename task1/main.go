package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

func printDir(path, indent string, printFiles bool, out io.Writer) error {
	// Определяем абсолютный путь к каталогу
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	files, err := os.ReadDir(absPath)
	if err != nil {
		return err
	}

	if printFiles {
		// Для каждого элемента в текущем каталоге
		for i, file := range files {
			// Создаем новый отступ в зависимости от того, последний ли это элемент
			var newIndent string
			if i == len(files)-1 {
				newIndent = indent + "└───"
			} else {
				newIndent = indent + "├───"
			}

			// Если элемент является каталогом, рекурсивно выводим его содержимое
			if file.IsDir() {
				out.Write([]byte(newIndent + file.Name() + "\n"))
				subPath := filepath.Join(absPath, file.Name())
				if i != len(files)-1 {
					newIndent = indent + "│\t"
				} else {
					newIndent = indent + "\t"
				}
				if err := printDir(subPath, newIndent, printFiles, out); err != nil {
					return err
				}
			} else {
				fileInfo, err := os.Stat(filepath.Join(absPath, file.Name()))
				if err != nil {
					return err
				}
				out.Write([]byte(newIndent + file.Name()))
				x := fileInfo.Size()
				if x == 0 {
					out.Write([]byte(" (" + "empty" + ")" + "\n"))
				} else {
					out.Write([]byte(" (" + strconv.FormatInt(x, 10) + "b" + ")" + "\n"))
				}
			}
		}
	} else {
		var directories []os.DirEntry

		for _, file := range files {
			// Проверяем, является ли элемент каталогом
			if file.IsDir() {
				// Добавляем информацию о каталоге в срез
				directories = append(directories, file)
			}
		}

		files = directories

		// Для каждой папки
		for i, file := range files {
			// Создаем новый отступ в зависимости от того, последний ли это элемент
			var newIndent string
			if i == len(directories)-1 {
				newIndent = indent + "└───"
			} else {
				newIndent = indent + "├───"
			}

			out.Write([]byte(newIndent + file.Name() + "\n"))
			subPath := filepath.Join(absPath, file.Name())
			if i != len(files)-1 {
				newIndent = indent + "│\t"
			} else {
				newIndent = indent + "\t"
			}
			if err := printDir(subPath, newIndent, printFiles, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	if err := printDir(path, "", printFiles, out); err != nil {
		return fmt.Errorf("Error:")
	}
	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
