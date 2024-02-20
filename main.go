package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "modernc.org/sqlite"
)

var db *sql.DB

type Todo struct {
	ID int64 `json:"id"`;
	Description string `json:"description"`;
	Completed bool `json:"completed"`;
}

func main() {
	var err error
	db, err = sql.Open("sqlite", "./todos.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS todos (description TEXT, completed BOOLEAN)")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("./static")))

	mux.HandleFunc("/todos", func(w http.ResponseWriter, r *http.Request) {
		todos, err := todos()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(todos)
	})

	mux.HandleFunc("/todos/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		todo, err := todoByID(id)
		if err != nil {
			if err.Error() == fmt.Sprintf("todoByID %d: no such todo", id) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(todo)
	})

	mux.HandleFunc("PUT /todos/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var todo Todo
		if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		todo.ID = id
		if err := updateTodo(todo); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("DELETE /todos/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := deleteTodo(id); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /todos", func(w http.ResponseWriter, r *http.Request) {
		var todo Todo
		if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		id, err := addTodo(todo)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Location", fmt.Sprintf("/todos/%d", id))
		w.WriteHeader(http.StatusCreated)
	})

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func todos() ([]Todo, error) {
	rows, err := db.Query("SELECT rowid, description, completed FROM todos")
	if err != nil {
		return nil, fmt.Errorf("todos: %v", err)
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Description, &todo.Completed); err != nil {
			return nil, fmt.Errorf("todos: %v", err)
		}
		todos = append(todos, todo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("todos: %v", err)
	}
	return todos, nil
}

func todoByID(id int64) (Todo, error) {
	var todo Todo
	row := db.QueryRow("SELECT rowid, description, completed FROM todos WHERE rowid = ?", id)
	if err := row.Scan(&todo.ID, &todo.Description, &todo.Completed); err != nil {
		if err == sql.ErrNoRows {
			return todo, fmt.Errorf("todoByID %d: no such todo", id)
		}
		return todo, fmt.Errorf("todoByID %d: %v", id, err)
	}
	return todo, nil
}

func addTodo(todo Todo) (int64, error) {
	result, err := db.Exec("INSERT INTO todos (description, completed) VALUES (?, ?)", todo.Description, todo.Completed)
	if err != nil {
		return 0, fmt.Errorf("addTodo: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("addTodo: %v", err)
	}
	return id, nil
}

func updateTodo(todo Todo) error {
	_, err := db.Exec("UPDATE todos SET description = ?, completed = ? WHERE rowid = ?", todo.Description, todo.Completed, todo.ID)
	if err != nil {
		return fmt.Errorf("updateTodo: %v", err)
	}
	return nil
}

func deleteTodo(id int64) error {
	_, err := db.Exec("DELETE FROM todos WHERE rowid = ?", id)
	if err != nil {
		return fmt.Errorf("deleteTodo: %v", err)
	}
	return nil
}
