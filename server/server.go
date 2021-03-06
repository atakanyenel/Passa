package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Cloud-Pie/Passa/database"
	"github.com/Cloud-Pie/Passa/ymlparser"
	"github.com/gin-gonic/gin"
)

var stateChannel chan *ymlparser.State

//SetupServer setups the web interface server
func SetupServer(sc chan *ymlparser.State) *gin.Engine {
	r := gin.Default()
	stateChannel = sc //left: global, right: func param

	r.GET("/", func(ctx *gin.Context) {

		ctx.JSON(200, r.Routes())
	})

	r.GET("/ui/timeline", func(ctx *gin.Context) {

		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(serverTemplate))
		/*ctx.JSON(200, gin.H{
			"timeline": "Not working will be fixed in v1.1",
		})
		*/
	})

	statesRest := r.Group("/api/states")
	{
		statesRest.POST("/", createState)
		statesRest.GET("/", getAllStates)
		statesRest.GET("/:name", getSingleState)
		statesRest.POST("/:name", updateState)
		statesRest.DELETE("/:name", deleteState)

	}

	r.GET("/api/invalidate/:timestamp", invalidateFutureStates) //FIXME: yesika wants this to be POST
	r.POST("/test", func(c *gin.Context) {
		var myState ymlparser.State
		c.BindJSON(&myState)
		fmt.Printf("%v", myState)
		c.JSON(200, myState)
	})
	return r
}

func createState(c *gin.Context) {
	var newState ymlparser.State
	err := c.BindJSON(&newState)
	fmt.Println(err)
	if newState.ISODate.IsZero() || newState.Services == nil { //input validation
		c.JSON(422, gin.H{
			"error": "Time or service field is empty",
		})
	} else {

		stateChannel <- &newState
		c.JSON(200, gin.H{
			"data": "success",
		})
	}
}

func getAllStates(c *gin.Context) {
	fmt.Printf("%+v", database.ReadAllStates())
	c.JSON(200, database.ReadAllStates())
}
func getSingleState(c *gin.Context) {
	name := c.Params.ByName("name")
	postToReturn := database.GetSingleState(name)
	c.JSON(200, postToReturn)

}
func updateState(c *gin.Context) {
	name := c.Params.ByName("name")
	var updatedState ymlparser.State
	c.BindJSON(&updatedState)
	fmt.Printf("%v", updatedState)

	database.UpdateState(updatedState, name)
	c.JSON(200, gin.H{
		"data": "success",
	})
}
func deleteState(c *gin.Context) {
	name := c.Params.ByName("name")
	err := database.DeleteState(name)
	if err != nil { //Not Found
		c.JSON(422, gin.H{"error": "Not Found"})
	} else {

		c.JSON(200, gin.H{"data": "success"})
	}
}

func invalidateFutureStates(c *gin.Context) {

	timestamp := c.Params.ByName("timestamp")
	invalidateTime, err := time.Parse(time.RFC822Z, timestamp)
	if err != nil {
		c.JSON(200, gin.H{
			"error": "Time is invalid",
		})
	}
	var invalidatedStateNum = 0
	for _, s := range database.ReadAllStates() {
		if s.ISODate.After(invalidateTime) {
			database.DeleteState(s.Name)
			invalidatedStateNum++
		}
	}

	c.JSON(200, gin.H{
		"invalidatedStates": invalidatedStateNum,
	})
}
