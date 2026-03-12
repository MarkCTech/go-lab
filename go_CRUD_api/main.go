package main

import (
	"github.com/martoranam/go_site/myhandlers"
	"github.com/martoranam/sql_db"

	"github.com/gin-gonic/gin"
)

func main() {
	myhandlers.Database = sql_db.Dbstart("todosdb")
	r := SetupRouter()
	r.Run(":5000")
}

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.LoadHTMLGlob("html/*")

	r.GET("/", myhandlers.Home)
	r.GET("/contact", myhandlers.Contact)
	r.GET("/helloworld", myhandlers.Helloworld)
	r.GET("/todos", myhandlers.GetAllTodos)
	r.GET("/todo/:id", myhandlers.GetTodobyId)

	r.POST("/todos", myhandlers.AddTodo)
	r.POST("/todo/:id", myhandlers.AddTodo)

	r.POST("/todos/status", myhandlers.CompletebyId)
	r.POST("/todo/status", myhandlers.CompletebyId)

	r.POST("todos/delete", myhandlers.DeletebyId)
	r.POST("todo/delete", myhandlers.DeletebyId)
	return r
}
