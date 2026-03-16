package sql_db

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
)

type Task struct {
	Id        string `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

var (
	task Task
)

func Crudin1(CrudDb *sql.DB) {
	var i Task
	var r Task
	InputTask(&i)
	AddTask(CrudDb, &i)
	GetTaskbyTitle(CrudDb, task.Title)
	ReturnToInput(&r, &i)

	CompleteTask(CrudDb, &i)
	GetAllTasks(CrudDb)
	DeleteTask(CrudDb, &i)
	GetAllTasks(CrudDb)
}

func InputTask(i *Task) {
	fmt.Print("\nEnter title of task: ")
	fmt.Scanln(&i.Title)
}

func ReturnToInput(r *Task, i *Task) {
	i.Id = r.Id
	i.Title = r.Title
	i.Completed = r.Completed
}

func querytoarr(results *sql.Rows) []Task {
	fmt.Printf("\nRetrieved these tasks:\n\n")
	var alltasks []Task
	for results.Next() {
		var task Task
		err := results.Scan(&task.Id, &task.Title, &task.Completed)
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(task.Id, task.Title, task.Completed)
		alltasks = append(alltasks, task)
	}
	fmt.Print("\n")
	return alltasks
}

func Count(db *sql.DB) int {
	rows, err := db.Query("SELECT COUNT(*) FROM tasks")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var count int

	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("Number of rows are %v\n", count)
	return count
}

func GetTaskbyTitle(db *sql.DB, title string) []Task {
	results, err := db.Query("SELECT * FROM tasks WHERE title IS NOT NULL AND title = (?)", title)
	if err != nil {
		panic(err.Error())
	}
	return querytoarr(results)
}

func GetTaskbyId(db *sql.DB, id string) []Task {
	results, err := db.Query("SELECT * FROM tasks WHERE id IS NOT NULL AND id = (?)", id)
	if err != nil {
		panic(err.Error())
	}
	return querytoarr(results)
}

func GetAllTasks(db *sql.DB) []Task {
	results, err := db.Query("SELECT * FROM tasks WHERE id IS NOT NULL")
	if err != nil {
		panic(err.Error())
	}
	return querytoarr(results)
}

func AddTask(db *sql.DB, i *Task) {
	insert, err := db.Query("INSERT INTO tasks (title,completed) VALUES(?,?)", i.Title, i.Completed)
	if err != nil {
		panic(err.Error())
	}
	task = Task{Title: i.Title, Completed: i.Completed}
	defer insert.Close()
	fmt.Printf("\nSuccesfully inserted: %v\n", i.Title)
}

func DeleteTask(db *sql.DB, i *Task) {
	intId, err := strconv.Atoi(i.Id)
	if err != nil {
		panic(err.Error())
	}

	delete, err := db.Query("DELETE FROM tasks WHERE id = (?)", intId)
	if err != nil {
		panic(err.Error())
	}
	defer delete.Close()
	fmt.Printf("\nSuccesfully completed task at ID: %v\n", i.Id)
}

func CompleteTask(db *sql.DB, i *Task) {
	intId, err := strconv.Atoi(i.Id)
	if err != nil {
		panic(err.Error())
	}
	i.Completed = !i.Completed
	insert, err := db.Query("UPDATE tasks SET completed = (?) WHERE id = (?)", i.Completed, intId)
	if err != nil {
		panic(err.Error())
	}
	defer insert.Close()
	fmt.Printf("\nSuccesfully updated completion of task at ID: %v\n", i.Id)
}
