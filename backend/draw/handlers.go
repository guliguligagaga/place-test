package draw

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func modifyCell(c *gin.Context, state *CellBroadcast) {
	var drawReq Req
	if err := c.ShouldBindJSON(&drawReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := state.updateCell(&drawReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
