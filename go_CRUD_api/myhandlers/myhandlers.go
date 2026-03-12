package myhandlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/martoranam/hello_world/helloworlder"
)

// type route struct {
// 	method  string
// 	regex   *regexp.Regexp
// 	handler http.HandlerFunc
// }

// type homePage struct {
// 	Title   string
// 	HomeMsg string
// }

func Home(c *gin.Context) {
	c.HTML(http.StatusOK, "hometemplate.html", gin.H{
		"Title":   "Welcome to my Website!",
		"HomeMsg": "This is the HomePage",
	})
}

// type contactPage struct {
// 	Title    string
// 	Contacts string
// }

func Contact(c *gin.Context) {
	c.HTML(http.StatusOK, "contactstemplate.html", gin.H{
		"Title":    "Contact me via: ",
		"Contacts": "mark_tech@hotmail.com",
	})
}

// type helloworldPage struct {
// 	Title   string
// 	HWprint string
// }

func Helloworld(c *gin.Context) {
	var printable string
	helloworlder.Update(&printable)

	c.HTML(http.StatusOK, "helloworldtemplate.html", gin.H{
		"Title":   "This page should print a message below!",
		"HWprint": printable,
	})
}

// (gin.Context) {
// 	var printable string
// 	helloworlder.Update(&printable)

// 	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
// 	p := helloworldPage{Title: "This page should print a message below!", HWprint: printable}
// 	t, _ := template.ParseFiles("html/helloworldtemplate.html")
// 	fmt.Println(t.Execute(w, p))
// }
